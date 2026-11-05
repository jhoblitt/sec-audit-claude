package gate_test

import (
	"fmt"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/gate"
	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/gate/gatefakes"
	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/gittest"
)

const (
	exportLink = "https://github.com/example/widget/blob/v1.0.0/src/widget/export.py#L13-L18"
	libLink    = "https://github.com/example/widget-lib/blob/v0.3.0/widget_lib/paths.py#L6-L9"
)

var _ = Describe("the links check", func() {
	var f *fixture

	BeforeEach(func(ctx SpecContext) {
		f = newFixture(ctx, "clean")
	})

	// appendLines adds a paragraph of lines at the end of the report.
	appendLines := func(lines ...string) {
		f.report += "\n" + strings.Join(lines, "\n") + "\n"
	}

	DescribeTable("checks the cited lines against the file's length", Label("TestCheckLinePastEOF"),
		func(ctx SpecContext, link, message string) {
			replace(&f.report, exportLink, link)
			problems := f.check(ctx)
			if message == "" {
				Expect(problems).To(BeEmpty())
				return
			}
			Expect(problems).To(ConsistOf(problem("links", f.reportLine(link), message)))
		},
		Entry("a range ending on the last line",
			"https://github.com/example/widget/blob/v1.0.0/src/widget/export.py#L30-L39", ""),
		Entry("a range ending one line past the end",
			"https://github.com/example/widget/blob/v1.0.0/src/widget/export.py#L30-L40",
			"line 40 is past the end of src/widget/export.py (39 lines)"),
		Entry("a single line past the end",
			"https://github.com/example/widget/blob/v1.0.0/src/widget/export.py#L99",
			"line 99 is past the end of src/widget/export.py (39 lines)"),
		Entry("a range given in reverse",
			"https://github.com/example/widget/blob/v1.0.0/src/widget/export.py#L45-L13",
			"line 45 is past the end"),
	)

	DescribeTable("checks a github.com blame or raw link as it does a blob link", Label("TestCheckBlameAndRawLinks"),
		func(ctx SpecContext, link, message string) {
			appendLines("Also " + link + ".")
			problems := f.check(ctx)
			if message == "" {
				Expect(problems).To(BeEmpty())
				return
			}
			Expect(problems).To(ConsistOf(problem("links", f.reportLine(link), message)))
		},
		Entry("a blame link within the file",
			"https://github.com/example/widget/blame/v1.0.0/src/widget/export.py#L30-L39", ""),
		Entry("a blame link past the end of the file",
			"https://github.com/example/widget/blame/v1.0.0/src/widget/export.py#L40",
			"line 40 is past the end of src/widget/export.py (39 lines)"),
		Entry("a raw link to a file the ref lacks",
			"https://github.com/example/widget/raw/v1.0.0/src/widget/import.py",
			"no file src/widget/import.py at v1.0.0 in example/widget"),
		Entry("a blame link at a ref that is not mapped",
			"https://github.com/example/widget/blame/v0.9.0/src/widget/export.py#L13",
			`ref mismatch: the link cites example/widget at "v0.9.0", mapped at "v1.0.0"`),
	)

	It("reports a file that does not exist at the ref", func(ctx SpecContext) {
		link := "https://github.com/example/widget/blob/v1.0.0/src/widget/import.py#L3"
		replace(&f.report, exportLink, link)
		Expect(f.check(ctx)).To(ConsistOf(
			problem("links", f.reportLine(link), "no file src/widget/import.py at v1.0.0 in example/widget"),
		))
	})

	It("reports a link to a directory as no file", func(ctx SpecContext) {
		link := "https://github.com/example/widget/blob/v1.0.0/src/widget"
		replace(&f.report, exportLink, link)
		Expect(f.check(ctx)).To(ConsistOf(
			problem("links", f.reportLine(link), "no file src/widget at v1.0.0"),
		))
	})

	It("reports a link that names no file", func(ctx SpecContext) {
		appendLines("See https://github.com/example/widget/blob/v1.0.0/ for the tree.")
		Expect(f.check(ctx)).To(ConsistOf(
			problem("links", f.reportLine("for the tree"), "names no file"),
		))
	})

	It("reports a repository that is neither the target nor a dependency", Label("TestCheckUnmappedRepo"), func(ctx SpecContext) {
		f.opts.Deps = nil
		other := "https://github.com/example/gadget/blob/v2.0.0/main.py#L1"
		appendLines("Compare " + other + ".")
		Expect(f.check(ctx)).To(ConsistOf(
			problem("links", f.reportLine(libLink), "github.com/example/widget-lib is neither the --target nor a --dep"),
			problem("links", f.reportLine(other), "github.com/example/gadget is neither the --target nor a --dep"),
		))
	})

	It("maps a repository whatever the case of its owner and name", func(ctx SpecContext) {
		replace(&f.report, exportLink, strings.Replace(exportLink, "example/widget", "Example/Widget", 1))
		Expect(f.check(ctx)).To(BeEmpty())
	})

	It("reports one problem per link whose ref the clone lacks, naming the ref", Label("TestCheckMissingRef"), func(ctx SpecContext) {
		f.opts.Deps[0].Ref = "v0.4.0"
		missing := strings.Replace(libLink, "v0.3.0", "v0.4.0", 1)
		replace(&f.report, libLink, missing)
		second := "https://github.com/example/widget-lib/blob/v0.4.0/widget_lib/paths.py#L12"
		appendLines("Also " + second + ".")
		Expect(f.check(ctx)).To(ConsistOf(
			problem("links", f.reportLine(missing), `ref "v0.4.0" not found in `+f.opts.Deps[0].Dir),
			problem("links", f.reportLine(second), `ref "v0.4.0" not found in `+f.opts.Deps[0].Dir),
		))
	})

	It("reports a ref that differs from the mapped ref", func(ctx SpecContext) {
		link := strings.Replace(exportLink, "v1.0.0", "v1.0", 1)
		replace(&f.report, exportLink, link)
		Expect(f.check(ctx)).To(ConsistOf(
			problem("links", f.reportLine(link), `ref mismatch: the link cites example/widget at "v1.0", mapped at "v1.0.0"`),
		))
	})

	Describe("with a dependency mapped at a tag with a slash in it", Label("TestCheckSlashRefDependency"), func() {
		const slashLink = "https://github.com/example/widget-lib/blob/lib/v0.2.0/widget_lib/paths.py#L6-L9"

		BeforeEach(func(ctx SpecContext) {
			gittest.Git(ctx, f.opts.Deps[0].Dir, "tag", "lib/v0.2.0")
			f.opts.Deps[0].Ref = "lib/v0.2.0"
			replace(&f.report, libLink, slashLink)
		})

		It("accepts a link within the file", func(ctx SpecContext) {
			Expect(f.check(ctx)).To(BeEmpty())
		})

		It("reports a line past the end of the file", func(ctx SpecContext) {
			link := strings.Replace(slashLink, "#L6-L9", "#L16", 1)
			replace(&f.report, slashLink, link)
			Expect(f.check(ctx)).To(ConsistOf(
				problem("links", f.reportLine(link), "line 16 is past the end of widget_lib/paths.py (15 lines)"),
			))
		})

		It("reports a link that cites another such tag, naming that tag", func(ctx SpecContext) {
			link := strings.Replace(slashLink, "lib/v0.2.0", "lib/v0.3.0", 1)
			replace(&f.report, slashLink, link)
			Expect(f.check(ctx)).To(ConsistOf(
				problem("links", f.reportLine(link), `ref mismatch: the link cites example/widget-lib at "lib/v0.3.0", mapped at "lib/v0.2.0"`),
			))
		})
	})

	Describe("with the target mapped at a full commit ID", func() {
		var commit string

		BeforeEach(func(ctx SpecContext) {
			commit = gittest.Git(ctx, f.opts.Target.Dir, "rev-parse", "HEAD")
			f.opts.Target.Ref = commit
			f.report = strings.ReplaceAll(f.report, "example/widget/blob/v1.0.0/", "example/widget/blob/"+commit+"/")
		})

		It("accepts links that cite it in full", func(ctx SpecContext) {
			Expect(f.check(ctx)).To(BeEmpty())
		})

		It("accepts a link that cites an unambiguous prefix of it", func(ctx SpecContext) {
			link := strings.Replace(exportLink, "v1.0.0", commit[:7], 1)
			appendLines("Short: " + link + ".")
			Expect(f.check(ctx)).To(BeEmpty())
		})

		It("reports a link that cites some other commit", func(ctx SpecContext) {
			other := strings.Repeat("0", 40)
			link := strings.Replace(exportLink, "v1.0.0", other, 1)
			appendLines("Other: " + link + ".")
			Expect(f.check(ctx)).To(ConsistOf(
				problem("links", f.reportLine(link), "ref mismatch"),
			))
		})
	})

	It("lists a code link on another host as unverifiable", Label("TestCheckFlagsNonGitHubCodeLink"), func(ctx SpecContext) {
		gitlab := "https://gitlab.example.org/example/widget/-/blob/v1.0.0/src/widget/export.py#L13"
		gitea := "https://gitea.example.net/example/widget/src/tag/v1.0.0/src/widget/cache.py#L8"
		raw := "https://raw.githubusercontent.com/example/widget/v1.0.0/src/widget/server.py"
		appendLines("GitLab: "+gitlab+".", "", "Gitea: "+gitea+".", "", "Raw: "+raw+".")
		Expect(f.check(ctx)).To(ConsistOf(
			problem("links", f.reportLine(gitlab), "unverifiable code link on gitlab.example.org"),
			problem("links", f.reportLine(gitea), "unverifiable code link on gitea.example.net"),
			problem("links", f.reportLine(raw), "unverifiable code link on raw.githubusercontent.com"),
		))

		f.opts.AllowHosts = []string{"gitlab.example.org", "Gitea.Example.Net", "raw.githubusercontent.com"}
		Expect(f.check(ctx)).To(BeEmpty())
	})

	It("looks each file up once, however many links cite it", Label("TestLinkCacheOneLookupPerFile"), func(ctx SpecContext) {
		const lookup = "v1.0.0:src/widget/export.py"
		base := f.report
		run := func(links int) (calls, lookups int) {
			lines := make([]string, links)
			for i := range lines {
				lines[i] = fmt.Sprintf("- https://github.com/example/widget/blob/v1.0.0/src/widget/export.py#L%d", i%39+1)
			}
			f.report = base
			appendLines(lines...)
			runner := &gatefakes.FakeRunner{}
			runner.GitStub = gate.ExecRunner{}.Git
			f.opts.Git = runner
			Expect(f.check(ctx)).To(BeEmpty())
			for i := range runner.GitCallCount() {
				if _, _, args := runner.GitArgsForCall(i); slices.Contains(args, lookup) {
					lookups++
				}
			}
			return runner.GitCallCount(), lookups
		}

		calls1, _ := run(1)
		calls50, lookups50 := run(50)
		Expect(lookups50).To(Equal(1), "git lookups of %s for 51 links", lookup)
		Expect(calls50).To(Equal(calls1), "git calls for 51 links against 2")
	})
})
