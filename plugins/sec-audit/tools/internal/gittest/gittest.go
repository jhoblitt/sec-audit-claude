// Package gittest builds throwaway git repositories for specs.
//
// Every git it runs reads neither the user's nor the system's git
// configuration and carries an identity of its own, so a commit-signing or
// hook setting on the machine cannot reach a spec.
package gittest

import (
	"context"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Git runs git in dir and returns its standard output, trimmed. A failure
// fails the running spec.
func Git(ctx context.Context, dir string, args ...string) string {
	GinkgoHelper()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...) //nolint:gosec // G204: specs run git with the arguments they choose
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=gittest",
		"GIT_AUTHOR_EMAIL=gittest@example.com",
		"GIT_COMMITTER_NAME=gittest",
		"GIT_COMMITTER_EMAIL=gittest@example.com",
	)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	Expect(err).NotTo(HaveOccurred(), "git %s: %s", strings.Join(args, " "), stderr.String())
	return strings.TrimSpace(string(out))
}

// Repo copies the tree at src into a new repository at dst, commits it, tags
// the commit tag, and returns the commit's full ID.
func Repo(ctx context.Context, src, dst, tag string) string {
	GinkgoHelper()
	Expect(os.CopyFS(dst, os.DirFS(src))).To(Succeed())
	Git(ctx, dst, "init", "--quiet", "--initial-branch=main")
	Git(ctx, dst, "add", "--all")
	Git(ctx, dst, "commit", "--quiet", "--message=fixture")
	Git(ctx, dst, "tag", tag)
	return Git(ctx, dst, "rev-parse", "HEAD")
}
