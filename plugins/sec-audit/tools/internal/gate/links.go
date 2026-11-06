package gate

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/report"
)

const githubHost = "github.com"

// hexRE is what a link ref must look like to stand for the commit ID it is
// a prefix of.
var hexRE = regexp.MustCompile(`^[0-9a-f]{4,39}$`)

// linker checks code links. Each git lookup is cached: a ref per (repository
// directory, ref), a file's line count per (repository directory, ref,
// path), so a report's cost in git calls grows with the files it cites, not
// with its links.
type linker struct {
	git    Runner
	report string
	repos  []RepoRef
	allow  []string
	refs   map[[2]string]string
	files  map[[3]string]int
}

func newLinker(ctx context.Context, opts Options) (*linker, error) {
	l := &linker{
		git:    opts.Git,
		report: opts.Report,
		repos:  append([]RepoRef{opts.Target}, opts.Deps...),
		refs:   map[[2]string]string{},
		files:  map[[3]string]int{},
	}
	for _, h := range opts.AllowHosts {
		l.allow = append(l.allow, strings.ToLower(h))
	}
	for _, r := range l.repos {
		if _, err := l.git.Git(ctx, r.Dir, "rev-parse", "--git-dir"); err != nil {
			return nil, fmt.Errorf("%s: %s is not a git repository: %w", r, r.Dir, err)
		}
	}
	return l, nil
}

func (l *linker) check(ctx context.Context, links []report.CodeLink) []Problem {
	var problems []Problem
	for _, cl := range links {
		if msg := l.link(ctx, cl); msg != "" {
			problems = append(problems, Problem{Check: CheckLinks, Where: where(l.report, cl.Line), Message: msg})
		}
	}
	return problems
}

// link returns what is wrong with one code link, or "".
func (l *linker) link(ctx context.Context, cl report.CodeLink) string {
	if cl.Host != githubHost {
		if slices.Contains(l.allow, cl.Host) {
			return ""
		}
		return fmt.Sprintf("unverifiable code link on %s: the gate resolves only github.com blob, blame and raw links; "+
			"pass --allow-host %s to accept this host's links unverified", cl.Host, cl.Host)
	}
	name := cl.Owner + "/" + cl.Repo
	var mapped []RepoRef
	for _, r := range l.repos {
		if strings.EqualFold(r.Owner, cl.Owner) && strings.EqualFold(r.Repo, cl.Repo) {
			mapped = append(mapped, r)
		}
	}
	if len(mapped) == 0 {
		return fmt.Sprintf("github.com/%s is neither the --target nor a --dep", name)
	}
	repo, ref, path, ok := pick(mapped, cl)
	if !ok {
		return fmt.Sprintf("ref mismatch: the link cites %s at %q, mapped at %s", name, cited(cl, mapped), quoteRefs(mapped))
	}
	if path == "" {
		return fmt.Sprintf("code link to %s names no file", name)
	}
	commit, ok := l.resolve(ctx, repo.Dir, repo.Ref)
	if !ok {
		return fmt.Sprintf("ref %q not found in %s (a shallow clone, or a missing tag?)", repo.Ref, repo.Dir)
	}
	if ref != repo.Ref {
		if short, found := l.resolve(ctx, repo.Dir, ref); !found || short != commit {
			return fmt.Sprintf("ref mismatch: the link cites %s at %q, which is not an unambiguous prefix of %q",
				name, ref, repo.Ref)
		}
	}
	lines, ok := l.lines(ctx, repo.Dir, repo.Ref, path)
	if !ok {
		return fmt.Sprintf("no file %s at %s in %s", path, repo.Ref, repo)
	}
	if last := max(cl.L1, cl.L2); last > lines {
		return fmt.Sprintf("line %d is past the end of %s (%d lines)", last, path, lines)
	}
	return ""
}

// pick returns the mapping the link cites, the ref it cites it at and the
// path it cites there. The link's ref and path may start with a mapped ref,
// slashes and all, and the longest such ref wins; failing that, the link's
// ref may be a hex prefix of a mapped commit ID.
func pick(mapped []RepoRef, cl report.CodeLink) (repo RepoRef, ref, path string, ok bool) {
	for _, r := range mapped {
		if p, found := cl.PathAt(r.Ref); found && (!ok || len(r.Ref) > len(repo.Ref)) {
			repo, ref, path, ok = r, r.Ref, p, true
		}
	}
	if ok {
		return repo, ref, path, true
	}
	if hexRE.MatchString(cl.Ref) {
		for _, r := range mapped {
			if strings.HasPrefix(r.Ref, cl.Ref) {
				return r, cl.Ref, cl.Path, true
			}
		}
	}
	return RepoRef{}, "", "", false
}

// cited is the ref a link that matched no mapping cites, read as having as
// many path elements as the mapped ref with the most, so that a link to
// lib/v0.3.0 beside a mapping at lib/v0.2.0 names lib/v0.3.0, not lib.
func cited(cl report.CodeLink, mapped []RepoRef) string {
	n := 1
	for _, r := range mapped {
		n = max(n, strings.Count(r.Ref, "/")+1)
	}
	elems := strings.Split(strings.TrimSuffix(cl.Ref+"/"+cl.Path, "/"), "/")
	return strings.Join(elems[:min(n, len(elems))], "/")
}

func quoteRefs(mapped []RepoRef) string {
	refs := make([]string, len(mapped))
	for i, r := range mapped {
		refs[i] = fmt.Sprintf("%q", r.Ref)
	}
	return strings.Join(refs, " or ")
}

// resolve returns the commit ref names in the repository at dir, or false
// when it names none there or is ambiguous.
func (l *linker) resolve(ctx context.Context, dir, ref string) (string, bool) {
	key := [2]string{dir, ref}
	if commit, ok := l.refs[key]; ok {
		return commit, commit != ""
	}
	out, err := l.git.Git(ctx, dir, "rev-parse", "--verify", "--quiet", "--end-of-options", ref+"^{commit}")
	commit := ""
	if err == nil {
		commit = strings.TrimSpace(string(out))
	}
	l.refs[key] = commit
	return commit, commit != ""
}

// lines returns the line count of the file at path at ref, or false when
// there is no such file.
func (l *linker) lines(ctx context.Context, dir, ref, path string) (int, bool) {
	key := [3]string{dir, ref, path}
	if n, ok := l.files[key]; ok {
		return n, n >= 0
	}
	n := -1
	if blob, err := l.git.Git(ctx, dir, "cat-file", "blob", "--end-of-options", ref+":"+path); err == nil {
		n = bytes.Count(blob, []byte("\n"))
		if len(blob) > 0 && blob[len(blob)-1] != '\n' {
			n++
		}
	}
	l.files[key] = n
	return n, n >= 0
}
