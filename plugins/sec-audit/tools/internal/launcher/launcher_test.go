// Package launcher_test tests tools/run.sh, the build-on-first-use launcher
// every skill invokes audit-gate through; it has no Go source of its own.
package launcher_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	runScript = "../../run.sh"
	manifest  = "../../../.claude-plugin/plugin.json"
	dieLine   = "sec-audit tools: cannot create "
)

// pluginName is the name the harness builds CLAUDE_PLUGIN_DATA's basename
// from, and the constant run.sh matches it against.
func pluginName() string {
	GinkgoHelper()
	raw, err := os.ReadFile(manifest)
	Expect(err).NotTo(HaveOccurred())
	var m struct {
		Name string `json:"name"`
	}
	Expect(json.Unmarshal(raw, &m)).To(Succeed())
	Expect(m.Name).NotTo(BeEmpty(), "%s has no name", manifest)
	return m.Name
}

// goEnv reads one of the go command's settings, so a launched build can
// share this process's build and module caches.
func goEnv(ctx context.Context, key string) string {
	GinkgoHelper()
	out, err := exec.CommandContext(ctx, "go", "env", key).Output()
	Expect(err).NotTo(HaveOccurred())
	return strings.TrimSpace(string(out))
}

// launch runs run.sh for tool under a sandbox root: HOME, the XDG cache and
// CLAUDE_PLUGIN_ROOT all point inside it, and CLAUDE_PLUGIN_DATA is
// pluginData. env, when given, is appended last and so wins.
func launch(ctx context.Context, root, pluginData, tool string, env ...string) (string, error) {
	GinkgoHelper()
	cmd := exec.CommandContext(ctx, "bash", runScript, tool)
	cmd.Env = append(os.Environ(),
		"HOME="+root,
		"XDG_CACHE_HOME="+filepath.Join(root, "xdg"),
		"CLAUDE_PLUGIN_ROOT="+filepath.Join(root, "root"),
		"CLAUDE_PLUGIN_DATA="+pluginData,
		"GOCACHE="+goEnv(ctx, "GOCACHE"),
	)
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

var _ = Describe("run.sh's cache directory", func() {
	// chosenCacheDir reports which directory run.sh picked, by making the
	// pick uncreatable: a regular file where the directory would go turns
	// "mkdir -p" into the launcher's own "cannot create <dir>" failure,
	// before it builds anything. It returns the sandbox root the answer is
	// relative to.
	chosenCacheDir := func(ctx context.Context, dataBasename string) (root, chosen string) {
		GinkgoHelper()
		root = GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(root, "root", "tools", "cmd", "audit-gate"), 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(root, "xdg"), nil, 0o600)).To(Succeed())
		pluginData := ""
		if dataBasename != "" {
			pluginData = filepath.Join(root, dataBasename)
			Expect(os.WriteFile(pluginData, nil, 0o600)).To(Succeed())
		}
		out, err := launch(ctx, root, pluginData, "audit-gate")
		Expect(err).To(HaveOccurred(), "run.sh succeeded with every cache directory blocked:\n%s", out)
		i := strings.LastIndex(out, dieLine)
		Expect(i).NotTo(Equal(-1), "run.sh failed without naming a cache directory:\n%s", out)
		return root, strings.TrimSpace(out[i+len(dieLine):])
	}

	// The observed failure this guards: another plugin leaks its
	// CLAUDE_PLUGIN_DATA into the shell the model runs run.sh from, and
	// every tool dies on a read-only path under that plugin.
	DescribeTable("ignores another plugin's CLAUDE_PLUGIN_DATA",
		func(ctx SpecContext, basename func(name string) string) {
			root, got := chosenCacheDir(ctx, basename(pluginName()))
			Expect(got).To(Equal(filepath.Join(root, "xdg", "sec-audit-claude", "tools")))
		},
		Entry("unset", func(string) string { return "" }),
		Entry("another plugin's", func(string) string { return "codex-openai-codex" }),
		Entry("a bare data directory", func(string) string { return "data" }),
		Entry("a name this plugin's is a prefix of", func(name string) string { return name + "2" }),
	)

	DescribeTable("honors this plugin's CLAUDE_PLUGIN_DATA",
		func(ctx SpecContext, basename func(name string) string) {
			base := basename(pluginName())
			root, got := chosenCacheDir(ctx, base)
			Expect(got).To(Equal(filepath.Join(root, base, "tools")))
		},
		Entry("the plugin's name", func(name string) string { return name }),
		Entry("the plugin's name and its marketplace", func(name string) string { return name + "-sec-audit-claude" }),
	)
})

// A two-tool module whose commands share nothing: alpha imports internal/a
// and beta imports internal/b, so an edit under one is invisible to the
// other's package set and visible to its own.
var fakeTree = map[string]string{
	"go.mod":            "module example.test\n\ngo 1.24\n",
	"cmd/alpha/main.go": "package main\n\nimport \"example.test/internal/a\"\n\nvar _ = a.S\n\nfunc main() {}\n",
	"cmd/beta/main.go":  "package main\n\nimport \"example.test/internal/b\"\n\nvar _ = b.S\n\nfunc main() {}\n",
	"internal/a/a.go":   "package a\n\nconst S = \"a1\"\n",
	"internal/b/b.go":   "package b\n\nconst S = \"b1\"\n",
}

var _ = Describe("run.sh's build", func() {
	var root string

	write := func(rel, body string) {
		GinkgoHelper()
		p := filepath.Join(root, "root", "tools", filepath.FromSlash(rel))
		Expect(os.MkdirAll(filepath.Dir(p), 0o755)).To(Succeed())
		Expect(os.WriteFile(p, []byte(body), 0o600)).To(Succeed())
	}

	// built launches tool, which must succeed, and returns the cached
	// binary; inode identity tells a rebuild, which lands a fresh file
	// through mv, from a reuse.
	built := func(ctx context.Context, tool string, env ...string) os.FileInfo {
		GinkgoHelper()
		out, err := launch(ctx, root, "", tool, env...)
		Expect(err).NotTo(HaveOccurred(), "run.sh %s:\n%s", tool, out)
		fi, err := os.Stat(filepath.Join(root, "xdg", "sec-audit-claude", "tools", tool))
		Expect(err).NotTo(HaveOccurred())
		return fi
	}

	BeforeEach(func() {
		root = GinkgoT().TempDir()
		for rel, body := range fakeTree {
			write(rel, body)
		}
	})

	It("rebuilds a tool when its own packages change, and only then", func(ctx SpecContext) {
		alpha := built(ctx, "alpha")
		built(ctx, "beta")

		write("internal/b/b.go", "package b\n\nconst S = \"b2\"\n")
		Expect(os.SameFile(alpha, built(ctx, "alpha"))).To(BeTrue(), "editing beta's package rebuilt alpha")

		write("internal/a/a.go", "package a\n\nconst S = \"a2\"\n")
		Expect(os.SameFile(alpha, built(ctx, "alpha"))).To(BeFalse(), "editing alpha's own package did not rebuild it")
	})

	// The per-tool fingerprint needs go list, so without a toolchain
	// staleness falls back to the mtime test; a cached binary must still run
	// there, and a tool with nothing cached must still fail loudly.
	It("runs a cached binary without a toolchain, and fails loudly with nothing cached", func(ctx SpecContext) {
		alpha := built(ctx, "alpha")

		path := GinkgoT().TempDir()
		for _, name := range []string{"find", "sort", "xargs", "cat", "cksum", "mkdir", "mv", "rm", "basename", "flock"} {
			if bin, err := exec.LookPath(name); err == nil {
				Expect(os.Symlink(bin, filepath.Join(path, name))).To(Succeed())
			}
		}
		Expect(os.SameFile(alpha, built(ctx, "alpha", "PATH="+path))).To(BeTrue(),
			"the cached binary was replaced by a run that had no toolchain")

		out, err := launch(ctx, root, "", "beta", "PATH="+path)
		Expect(err).To(HaveOccurred(), "run.sh built beta without a toolchain:\n%s", out)
		Expect(out).To(ContainSubstring("go toolchain not found"))
	})

	DescribeTable("fails loudly when the build fails",
		func(ctx SpecContext, rel, body string) {
			write(rel, body)
			out, err := launch(ctx, root, "", "broken", "GOPROXY=off", "GOFLAGS=-mod=mod")
			Expect(err).To(HaveOccurred(), "run.sh succeeded:\n%s", out)
			Expect(out).To(ContainSubstring("sec-audit tools: build failed for 'broken'"))
		},
		Entry("on code that does not compile", "cmd/broken/main.go", "package main\n\nfunc main() {\n"),
		Entry("on a module it cannot download", "cmd/broken/main.go",
			"package main\n\nimport _ \"example.invalid/absent\"\n\nfunc main() {}\n"),
	)

	It("names an unknown tool", func(ctx SpecContext) {
		out, err := launch(ctx, root, "", "gamma")
		Expect(err).To(HaveOccurred())
		Expect(out).To(ContainSubstring("unknown tool 'gamma'"))
	})
})

var _ = Describe("run.sh audit-gate", func() {
	// The launch runs from a directory holding no tools, so a fallback that
	// read the working directory rather than run.sh's own location would
	// find no audit-gate there.
	DescribeTable("builds the shipped tool and prints its help",
		func(ctx SpecContext, pluginRoot func(plugin string) []string) {
			root := GinkgoT().TempDir()
			plugin, err := filepath.Abs("../../..")
			Expect(err).NotTo(HaveOccurred())
			script, err := filepath.Abs(runScript)
			Expect(err).NotTo(HaveOccurred())
			cmd := exec.CommandContext(ctx, "bash", script, "audit-gate", "check", "--help")
			cmd.Dir = root
			cmd.Env = slices.DeleteFunc(os.Environ(), func(kv string) bool { return strings.HasPrefix(kv, "CLAUDE_PLUGIN_ROOT=") })
			cmd.Env = append(cmd.Env,
				"HOME="+root,
				"XDG_CACHE_HOME="+filepath.Join(root, "xdg"),
				"CLAUDE_PLUGIN_DATA=",
				"GOCACHE="+goEnv(ctx, "GOCACHE"),
				"GOMODCACHE="+goEnv(ctx, "GOMODCACHE"),
			)
			cmd.Env = append(cmd.Env, pluginRoot(plugin)...)
			out, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "run.sh audit-gate check --help:\n%s", out)
			Expect(string(out)).To(ContainSubstring("audit-gate check --report FILE"))
		},
		Entry("with CLAUDE_PLUGIN_ROOT naming the plugin", func(plugin string) []string {
			return []string{"CLAUDE_PLUGIN_ROOT=" + plugin}
		}),
		Entry("with CLAUDE_PLUGIN_ROOT unset, from its own location", func(string) []string { return nil }),
		Entry("with CLAUDE_PLUGIN_ROOT empty, from its own location", func(string) []string {
			return []string{"CLAUDE_PLUGIN_ROOT="}
		}),
	)
})
