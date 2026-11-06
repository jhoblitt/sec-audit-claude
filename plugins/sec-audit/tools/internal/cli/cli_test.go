package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/cli"
	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/gittest"
)

// testdata is the gate's, so the CLI runs over the same clean fixture.
const testdata = "../gate/testdata"

var _ = Describe("Run", func() {
	var (
		dir            string
		report, table  string
		target, dep    string
		stdout, stderr *bytes.Buffer
	)

	BeforeEach(func(ctx SpecContext) {
		dir = GinkgoT().TempDir()
		widget := filepath.Join(dir, "widget")
		lib := filepath.Join(dir, "widget-lib")
		gittest.Repo(ctx, filepath.Join(testdata, "repos", "widget"), widget, "v1.0.0")
		gittest.Repo(ctx, filepath.Join(testdata, "repos", "widget-lib"), lib, "v0.3.0")
		report = copyFile(filepath.Join(testdata, "clean", "REPORT.md"), dir)
		table = copyFile(filepath.Join(testdata, "clean", "findings.tsv"), dir)
		target = "example/widget=" + widget + "@v1.0.0"
		dep = "example/widget-lib=" + lib + "@v0.3.0"
		stdout, stderr = new(bytes.Buffer), new(bytes.Buffer)
	})

	run := func(ctx context.Context, args ...string) int {
		return cli.ExitCode(cli.Run(ctx, args, strings.NewReader(""), stdout, stderr))
	}
	check := func(ctx context.Context, extra ...string) int {
		return run(ctx, append([]string{"check", "--report", report, "--table", table, "--target", target, "--dep", dep}, extra...)...)
	}
	leavePlaceholder := func() {
		GinkgoHelper()
		f, err := os.OpenFile(report, os.O_APPEND|os.O_WRONLY, 0)
		Expect(err).NotTo(HaveOccurred())
		_, err = f.WriteString("\nOwner: {{OWNER_NAME}}\n")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())
	}
	scope := func() string {
		return "S01=" + filepath.Join(testdata, "scopes", "S01.txt") + ":" + filepath.Join(testdata, "scopes", "S01.notes.md")
	}

	Describe("exit codes", Label("TestCLIExitCodes"), func() {
		It("exits 0 on a clean report, printing the summary", func(ctx SpecContext) {
			Expect(check(ctx)).To(Equal(0), "stderr: %s", stderr)
			Expect(stdout.String()).To(Equal("0 problems\n"))
		})

		It("exits 1 on a report with problems, one line per problem and a summary", func(ctx SpecContext) {
			leavePlaceholder()
			Expect(check(ctx)).To(Equal(1))
			Expect(strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")).To(HaveExactElements(
				MatchRegexp(`^placeholders: \Q`+report+`\E:\d+: .*placeholder`),
				"1 problem (placeholders 1)",
			))
		})

		It("exits 0 on a clean coverage scope", func(ctx SpecContext) {
			Expect(run(ctx, "coverage", "--scope", scope())).To(Equal(0), "stderr: %s", stderr)
		})

		It("exits 1 on a coverage scope with a gap", func(ctx SpecContext) {
			list := filepath.Join(dir, "S02.txt")
			Expect(os.WriteFile(list, []byte("src/widget/extra.py\n"), 0o600)).To(Succeed())
			Expect(run(ctx, "coverage", "--scope", "S02="+list+":"+filepath.Join(testdata, "scopes", "S01.notes.md"))).To(Equal(1))
			Expect(stdout.String()).To(ContainSubstring("coverage: S02: missing: src/widget/extra.py"))
			Expect(stdout.String()).To(HaveSuffix("4 problems (coverage 4)\n"))
		})

		DescribeTable("exits 0 on a request for help",
			func(ctx SpecContext, args ...string) {
				Expect(run(ctx, args...)).To(Equal(0))
				Expect(stdout.String()).To(ContainSubstring("Usage:"))
			},
			Entry("audit-gate", "--help"),
			Entry("audit-gate check", "check", "--help"),
			Entry("audit-gate coverage", "coverage", "--help"),
		)

		DescribeTable("exits 2 on a usage error, naming it",
			func(ctx SpecContext, args func() []string, message string) {
				err := cli.Run(ctx, args(), strings.NewReader(""), stdout, stderr)
				Expect(cli.ExitCode(err)).To(Equal(2))
				Expect(err).To(MatchError(ContainSubstring(message)))
			},
			Entry("an unknown subcommand", func() []string { return []string{"render"} }, `unknown command "render"`),
			Entry("an unknown flag", func() []string { return []string{"check", "--frob"} }, "unknown flag: --frob"),
			Entry("no --report", func() []string { return []string{"check", "--table", table, "--target", target} }, "--report is required"),
			Entry("no --table", func() []string { return []string{"check", "--report", report, "--target", target} }, "--table is required"),
			Entry("no --target", func() []string { return []string{"check", "--report", report, "--table", table} }, "--target is required"),
			Entry("a --target with no ref",
				func() []string {
					return []string{"check", "--report", report, "--table", table, "--target", "example/widget=" + dir}
				},
				`--target "example/widget=`),
			Entry("a --dep with no owner",
				func() []string {
					return []string{"check", "--report", report, "--table", table, "--target", target, "--dep", "widget-lib=" + dir + "@v1"}
				},
				"want OWNER/REPO=DIR@REF"),
			Entry("no --scope", func() []string { return []string{"coverage"} }, "--scope is required"),
			Entry("a --scope with no manifest", func() []string { return []string{"coverage", "--scope", "S01=list.txt"} }, "want NAME=LISTFILE:MANIFESTFILE"),
		)

		DescribeTable("exits 2 on an I/O error",
			func(ctx SpecContext, args func() []string) {
				Expect(run(ctx, args()...)).To(Equal(2))
			},
			Entry("a report that does not exist", func() []string {
				return []string{"check", "--report", filepath.Join(dir, "absent.md"), "--table", table, "--target", target, "--dep", dep}
			}),
			Entry("a --patterns file that does not exist", func() []string {
				return []string{"check", "--report", report, "--table", table, "--target", target, "--dep", dep, "--patterns", filepath.Join(dir, "absent.tsv")}
			}),
			Entry("a --target directory that is not a repository", func() []string {
				return []string{"check", "--report", report, "--table", table, "--target", "example/widget=" + dir + "@v1.0.0", "--dep", dep}
			}),
			Entry("a scope list that does not exist", func() []string {
				return []string{"coverage", "--scope", "S01=" + filepath.Join(dir, "absent.txt") + ":" + filepath.Join(testdata, "scopes", "S01.notes.md")}
			}),
		)
	})

	Describe("--json", Label("TestCLIJSONShape"), func() {
		decode := func() (problems []map[string]any, summary map[string]any) {
			GinkgoHelper()
			var out map[string]json.RawMessage
			Expect(json.Unmarshal(stdout.Bytes(), &out)).To(Succeed(), "stdout: %s", stdout)
			Expect(out).To(HaveLen(2))
			Expect(json.Unmarshal(out["problems"], &problems)).To(Succeed())
			Expect(json.Unmarshal(out["summary"], &summary)).To(Succeed())
			return problems, summary
		}

		It("emits an empty problem list and a zero summary on a clean report", func(ctx SpecContext) {
			Expect(check(ctx, "--json")).To(Equal(0))
			Expect(stdout.String()).To(ContainSubstring(`"problems":[]`))
			problems, summary := decode()
			Expect(problems).To(BeEmpty())
			Expect(summary).To(Equal(map[string]any{"problems": 0.0, "by_check": map[string]any{}}))
		})

		It("emits each problem as check, where and message, and counts them by check", func(ctx SpecContext) {
			leavePlaceholder()
			Expect(check(ctx, "--json")).To(Equal(1))
			problems, summary := decode()
			Expect(problems).To(HaveExactElements(And(
				HaveLen(3),
				HaveKeyWithValue("check", "placeholders"),
				HaveKeyWithValue("where", MatchRegexp(`^\Q`+report+`\E:\d+$`)),
				HaveKeyWithValue("message", ContainSubstring("placeholder")),
			)))
			Expect(summary).To(Equal(map[string]any{"problems": 1.0, "by_check": map[string]any{"placeholders": 1.0}}))
		})

		It("is set by AUDIT_GATE_JSON as well as the flag", func(ctx SpecContext) {
			GinkgoT().Setenv("AUDIT_GATE_JSON", "true")
			Expect(run(ctx, "coverage", "--scope", scope())).To(Equal(0))
			_, summary := decode()
			Expect(summary).To(HaveKeyWithValue("problems", 0.0))
		})
	})
})

func copyFile(src, dir string) string {
	GinkgoHelper()
	data, err := os.ReadFile(src)
	Expect(err).NotTo(HaveOccurred())
	dst := filepath.Join(dir, filepath.Base(src))
	Expect(os.WriteFile(dst, data, 0o600)).To(Succeed())
	return dst
}
