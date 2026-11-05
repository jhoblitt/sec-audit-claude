package gate_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/gate"
	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/gittest"
)

// fixture is a copy of one testdata report and table, with git repositories
// built from testdata/repos for the code links to resolve against. A spec
// edits report or table, then runs the gate over the edited copy.
type fixture struct {
	dir           string
	report, table string
	opts          gate.Options
}

// newFixture copies testdata/<name> and builds the target, example/widget at
// v1.0.0, and its one dependency, example/widget-lib at v0.3.0.
func newFixture(ctx context.Context, name string) *fixture {
	GinkgoHelper()
	dir := GinkgoT().TempDir()
	widget := filepath.Join(dir, "widget")
	lib := filepath.Join(dir, "widget-lib")
	gittest.Repo(ctx, "testdata/repos/widget", widget, "v1.0.0")
	gittest.Repo(ctx, "testdata/repos/widget-lib", lib, "v0.3.0")
	return &fixture{
		dir:    dir,
		report: readFile(filepath.Join("testdata", name, "REPORT.md")),
		table:  readFile(filepath.Join("testdata", name, "findings.tsv")),
		opts: gate.Options{
			Report: filepath.Join(dir, "REPORT.md"),
			Table:  filepath.Join(dir, "findings.tsv"),
			Target: gate.RepoRef{Owner: "example", Repo: "widget", Dir: widget, Ref: "v1.0.0"},
			Deps:   []gate.RepoRef{{Owner: "example", Repo: "widget-lib", Dir: lib, Ref: "v0.3.0"}},
		},
	}
}

func readFile(path string) string {
	GinkgoHelper()
	data, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

// replace swaps the one occurrence of old in *s for repl; an edit whose
// anchor text is missing or repeated fails the spec rather than silently
// testing the unedited fixture.
func replace(s *string, old, repl string) {
	GinkgoHelper()
	Expect(strings.Count(*s, old)).To(Equal(1), "occurrences of %q", old)
	*s = strings.Replace(*s, old, repl, 1)
}

// check writes the fixture's report and table and runs the gate over them.
func (f *fixture) check(ctx context.Context) []gate.Problem {
	GinkgoHelper()
	Expect(os.WriteFile(f.opts.Report, []byte(f.report), 0o600)).To(Succeed())
	Expect(os.WriteFile(f.opts.Table, []byte(f.table), 0o600)).To(Succeed())
	problems, err := gate.Check(ctx, f.opts)
	Expect(err).NotTo(HaveOccurred())
	return problems
}

// reportLines are the report locations of the lines holding substr.
func (f *fixture) reportLines(substr string) []string {
	GinkgoHelper()
	var where []string
	for i, line := range strings.Split(f.report, "\n") {
		if strings.Contains(line, substr) {
			where = append(where, fmt.Sprintf("%s:%d", f.opts.Report, i+1))
		}
	}
	Expect(where).NotTo(BeEmpty(), "report lines holding %q", substr)
	return where
}

// reportLine is the report location of the first line holding substr.
func (f *fixture) reportLine(substr string) string {
	GinkgoHelper()
	return f.reportLines(substr)[0]
}

// problem matches a Problem by its check and location, and by a fragment of
// its message.
func problem(check, where, message string) types.GomegaMatcher {
	return And(
		HaveField("Check", check),
		HaveField("Where", where),
		HaveField("Message", ContainSubstring(message)),
	)
}

var _ = Describe("Check", func() {
	It("finds no problem in the clean fixture", Label("TestCheckGoldenClean"), func(ctx SpecContext) {
		Expect(newFixture(ctx, "clean").check(ctx)).To(BeEmpty())
	})

	It("finds no problem in a fixture scored with CVSS 3.1 only", Label("TestCheckGoldenNo40"), func(ctx SpecContext) {
		f := newFixture(ctx, "no40")
		f.opts.Deps = nil
		Expect(f.check(ctx)).To(BeEmpty())
	})

	It("reports a report it cannot read as an error", func(ctx SpecContext) {
		f := newFixture(ctx, "clean")
		f.opts.Report = filepath.Join(f.dir, "absent.md")
		Expect(os.WriteFile(f.opts.Table, []byte(f.table), 0o600)).To(Succeed())
		_, err := gate.Check(ctx, f.opts)
		Expect(err).To(MatchError(os.ErrNotExist))
	})

	It("reports a table it cannot read as an error", func(ctx SpecContext) {
		f := newFixture(ctx, "clean")
		Expect(os.WriteFile(f.opts.Report, []byte(f.report), 0o600)).To(Succeed())
		_, err := gate.Check(ctx, f.opts)
		Expect(err).To(MatchError(os.ErrNotExist))
	})

	It("reports a mapped directory that is not a git repository as an error", func(ctx SpecContext) {
		f := newFixture(ctx, "clean")
		f.opts.Deps[0].Dir = GinkgoT().TempDir()
		Expect(os.WriteFile(f.opts.Report, []byte(f.report), 0o600)).To(Succeed())
		Expect(os.WriteFile(f.opts.Table, []byte(f.table), 0o600)).To(Succeed())
		_, err := gate.Check(ctx, f.opts)
		Expect(err).To(MatchError(ContainSubstring("example/widget-lib")))
		Expect(err).To(MatchError(ContainSubstring("not a git repository")))
	})

	It("reports a malformed table as a problem and still runs the report checks", func(ctx SpecContext) {
		f := newFixture(ctx, "clean")
		replace(&f.table, "\tretired\t", "\tdropped\t")
		replace(&f.report, "(#limitations)", "(#limits)")
		Expect(f.check(ctx)).To(ConsistOf(
			problem("consistency", f.opts.Table+":5", `status "dropped"`),
			problem("anchors", f.reportLine("(#limits)"), "#limits"),
		))
	})
})
