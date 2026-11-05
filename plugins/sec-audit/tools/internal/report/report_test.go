package report_test

import (
	"math"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/report"
)

const (
	vec31 = "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N"
	vec40 = "CVSS:4.0/AV:L/AC:L/AT:N/PR:L/UI:N/VC:H/VI:L/VA:N/SC:N/SI:N/SA:N"
	fence = "```"

	overviewHeader = "| ID | CVSS 3.1 | CVSS 4.0 | Severity | Finding | Affected (v1.2 / main) | Found by | Default exposure |"
	overviewDelim  = "|---|---|---|---|---|---|---|---|"
)

// cvssRow renders a section's CVSS row in the report-template form.
func cvssRow(version, vector, rating string) string {
	return "| **CVSS " + version + "** | `" + vector + "` — **" + rating + "** |"
}

var _ = Describe("Parse", func() {
	var dir string

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
	})

	parse := func(lines ...string) *report.Report {
		GinkgoHelper()
		path := filepath.Join(dir, "REPORT.md")
		Expect(os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600)).To(Succeed())
		r, err := report.Parse(path)
		Expect(err).NotTo(HaveOccurred())
		return r
	}

	It("reports a missing file", func() {
		_, err := report.Parse(filepath.Join(dir, "absent.md"))
		Expect(err).To(MatchError(os.ErrNotExist))
	})

	Describe("finding sections", func() {
		It("reads the ID, title, vectors, scores and verdicts", Label("TestParseFindingSection"), func() {
			r := parse(
				"## Findings",
				"",
				`<a id="f-01"></a>`,
				"",
				"### F-01 — Widget export follows symlinks",
				"",
				"| Field | Value |",
				"|---|---|",
				cvssRow("3.1", vec31, "7.1 High"),
				cvssRow("4.0", vec40, "6.9 Medium"),
				"| **Affected versions** | v1.2 **PRESENT**; main **PARTIAL** — main gates the export behind a flag |",
			)
			Expect(r.Findings).To(Equal([]report.Section{{
				ID:      "F-01",
				Title:   "Widget export follows symlinks",
				CVSS31:  vec31,
				Score31: "7.1",
				Sev31:   "High",
				CVSS40:  vec40,
				Score40: "6.9",
				Sev40:   "Medium",
				Rel:     map[string]string{"v1.2": "PRESENT", "main": "PARTIAL"},
				Anchor:  "f-01",
				Line:    5,
			}}))
		})

		It("reads a CVSS 4.0 row of — as no vector", func() {
			r := parse(
				`<a id="f-01"></a>`,
				"",
				"### F-01 — Widget export follows symlinks",
				cvssRow("3.1", vec31, "7.1 High"),
				"| **CVSS 4.0** | — |",
			)
			Expect(r.Findings).To(HaveLen(1))
			Expect(r.Findings[0].CVSS40).To(Equal("-"))
			Expect(r.Findings[0].Score40).To(Equal("-"))
			Expect(r.Findings[0].Sev40).To(Equal("-"))
		})

		It("keeps a CVSS cell that is in neither form verbatim, with no score", func() {
			r := parse(
				`<a id="f-01"></a>`,
				"",
				"### F-01 — Widget export follows symlinks",
				"| **CVSS 3.1** | 7.1 High |",
			)
			Expect(r.Findings).To(HaveLen(1))
			Expect(r.Findings[0].CVSS31).To(Equal("7.1 High"))
			Expect(r.Findings[0].Score31).To(BeEmpty())
			Expect(r.Findings[0].Sev31).To(BeEmpty())
		})

		It("keeps a verdict it cannot read under its raw text", func() {
			r := parse(
				`<a id="f-01"></a>`,
				"",
				"### F-01 — Widget export follows symlinks",
				"| **Affected versions** | v1.2 **PRESENT**; main is affected |",
			)
			Expect(r.Findings[0].Rel).To(Equal(map[string]string{"v1.2": "PRESENT", "main is affected": ""}))
		})

		It("does not end a section at a heading inside fenced code", func() {
			r := parse(
				`<a id="f-01"></a>`,
				"",
				"### F-01 — Widget export follows symlinks",
				fence+"sh",
				"# reproduce",
				"widget export --follow /tmp/x",
				fence,
				"| **Affected versions** | main **PRESENT** |",
			)
			Expect(r.Findings[0].Rel).To(Equal(map[string]string{"main": "PRESENT"}))
		})

		It("ends a section at the next heading of level 3 or above", func() {
			r := parse(
				`<a id="f-01"></a>`,
				"",
				"### F-01 — Widget export follows symlinks",
				"#### Exploit scenario",
				cvssRow("3.1", vec31, "7.1 High"),
				"## Limitations",
				"| **Affected versions** | main **PRESENT** |",
			)
			Expect(r.Findings[0].CVSS31).To(Equal(vec31))
			Expect(r.Findings[0].Rel).To(BeNil())
		})

		DescribeTable("reads an ID-shaped heading as a section whatever its anchor, recording the anchor before it",
			func(anchor string, lines ...string) {
				r := parse(lines...)
				Expect(r.Findings).To(HaveLen(1))
				Expect(r.Findings[0].ID).To(Equal("F-01"))
				Expect(r.Findings[0].Anchor).To(Equal(anchor))
			},
			Entry("no anchor", "",
				"### F-01 — Widget export follows symlinks"),
			Entry("an anchor for another ID", "f-02",
				`<a id="f-02"></a>`, "", "### F-01 — Widget export follows symlinks"),
			Entry("an anchor in another case", "F-01",
				`<a id="F-01"></a>`, "", "### F-01 — Widget export follows symlinks"),
			Entry("an anchor separated from the heading by prose", "",
				`<a id="f-01"></a>`, "Prose.", "### F-01 — Widget export follows symlinks"),
		)

		It("reads a split cluster's lettered ID as an ID", func() {
			r := parse("### F-03a — Widget import follows symlinks")
			Expect(r.Findings).To(HaveExactElements(HaveField("ID", "F-03a")))
		})

		DescribeTable("reads a heading that starts with an ID in any other form as a malformed section",
			Label("TestParseMalformedSectionHeading"),
			func(heading, title string) {
				r := parse(`<a id="f-04"></a>`, "", heading, cvssRow("3.1", vec31, "7.1 High"))
				Expect(r.Findings).To(HaveExactElements(And(
					HaveField("ID", "F-04"),
					HaveField("Title", title),
					HaveField("Malformed", true),
					HaveField("Anchor", "f-04"),
					HaveField("CVSS31", vec31),
				)))
			},
			Entry("a colon", "### F-04: Widget upload accepts unbounded bodies", "Widget upload accepts unbounded bodies"),
			Entry("a hyphen", "### F-04 - Widget upload accepts unbounded bodies", "Widget upload accepts unbounded bodies"),
			Entry("an en dash", "### F-04 – Widget upload accepts unbounded bodies", "Widget upload accepts unbounded bodies"),
			Entry("a bold ID", "### **F-04** — Widget upload accepts unbounded bodies", "Widget upload accepts unbounded bodies"),
			Entry("an underscored ID", "### __F-04__ — Widget upload accepts unbounded bodies", "Widget upload accepts unbounded bodies"),
			Entry("a code-span ID", "### `F-04` — Widget upload accepts unbounded bodies", "Widget upload accepts unbounded bodies"),
			Entry("a linked ID", "### [F-04](#f-04) — Widget upload accepts unbounded bodies", "Widget upload accepts unbounded bodies"),
			Entry("an inline anchor", `### <a id="f-04"></a>F-04 — Widget upload accepts unbounded bodies`, "Widget upload accepts unbounded bodies"),
			Entry("a bold heading", "### **F-04 — Widget upload accepts unbounded bodies**", "Widget upload accepts unbounded bodies"),
			Entry("a bare ID", "### F-04", ""),
		)

		It("reads a level-4 heading that starts with an ID in another form as a malformed hardening note", func() {
			r := parse(`<a id="h-01"></a>`, "", "#### H-01: Widget logs request headers")
			Expect(r.Hardening).To(Equal([]report.Section{{
				ID: "H-01", Title: "Widget logs request headers", Malformed: true, Anchor: "h-01", Line: 3,
			}}))
			Expect(r.Findings).To(BeEmpty())
		})

		DescribeTable("is not a finding section",
			func(lines ...string) {
				Expect(parse(lines...).Findings).To(BeEmpty())
			},
			Entry("a prose heading in the section form",
				"### Scope — what was read"),
			Entry("a prose heading carrying an anchor of its own",
				`<a id="scope"></a>`, "", "### Scope — what was read"),
			Entry("a heading with an ID-shaped token past its start",
				"### Fixed upstream as CVE-0000-0001"),
			Entry("a heading whose first word runs on past an ID",
				"### F-04bc notes"),
			Entry("a level-2 heading",
				`<a id="f-01"></a>`, "", "## F-01 — Widget export follows symlinks"),
			Entry("a level-2 heading in another form",
				`<a id="f-01"></a>`, "", "## F-01: Widget export follows symlinks"),
		)
	})

	Describe("the overview", func() {
		It("reads the releases from the header, then each row", Label("TestParseOverview"), func() {
			r := parse(
				"## Overview",
				"",
				overviewHeader,
				overviewDelim,
				"| [F-01](#f-01) | 7.1 | 6.9 | High | Widget export follows symlinks | yes / partly | claude, gpt | on |",
				"| [F-02](#f-02) | 5.5 | — | Medium | Widget cache keys collide | no / yes | claude | off |",
				"",
				"| [F-03](#f-03) | 1.6 | — | Low | A row of a later table | no / no | claude | off |",
			)
			Expect(r.OverviewReleases).To(Equal([]string{"v1.2", "main"}))
			Expect(r.Overview).To(Equal([]report.OverviewRow{
				{
					ID: "F-01", Score31: "7.1", Score40: "6.9", Sev: "High",
					Title: "Widget export follows symlinks", Rel: []string{"PRESENT", "PARTIAL"},
					FoundBy: "claude, gpt", Default: "on",
				},
				{
					ID: "F-02", Score31: "5.5", Score40: "-", Sev: "Medium",
					Title: "Widget cache keys collide", Rel: []string{"ABSENT", "PRESENT"},
					FoundBy: "claude", Default: "off",
				},
			}))
		})

		It("unescapes a pipe in a cell", Label("TestOverviewEscapedPipe"), func() {
			r := parse(overviewHeader, overviewDelim,
				"| [F-01](#f-01) | 7.1 | 6.9 | High | Widget `a \\| b` filter accepts naïve input | yes / yes | claude | on |")
			Expect(r.Overview).To(HaveLen(1))
			Expect(r.Overview[0].Title).To(Equal("Widget `a | b` filter accepts naïve input"))
			Expect(r.Overview[0].FoundBy).To(Equal("claude"))
		})

		DescribeTable("keeps a cell it cannot read verbatim",
			func(row string, check func(report.OverviewRow)) {
				r := parse(overviewHeader, overviewDelim, row)
				Expect(r.Overview).To(HaveLen(1))
				check(r.Overview[0])
			},
			Entry("an ID linked to another ID's anchor",
				"| [F-01](#f-02) | 7.1 | 6.9 | High | Widget export | yes / yes | claude | on |",
				func(o report.OverviewRow) { Expect(o.ID).To(Equal("[F-01](#f-02)")) }),
			Entry("a verdict that is not yes, partly or no",
				"| [F-01](#f-01) | 7.1 | 6.9 | High | Widget export | yes / maybe | claude | on |",
				func(o report.OverviewRow) { Expect(o.Rel).To(Equal([]string{"PRESENT", "maybe"})) }),
		)

		It("finds no overview under a header that differs", func() {
			r := parse(
				"| ID | CVSS 3.1 | Severity | Finding | Affected (main) | Found by | Default exposure |",
				"|---|---|---|---|---|---|---|",
				"| [F-01](#f-01) | 7.1 | High | Widget export | yes | claude | on |",
			)
			Expect(r.Overview).To(BeEmpty())
			Expect(r.OverviewReleases).To(BeNil())
		})
	})

	Describe("the summary counts", func() {
		It("reads the sentence forms", Label("TestParseSummaryCounts"), func() {
			r := parse(
				"The audit confirmed **7 scored findings**: 1 Critical, 2 High, 3 Medium and 1 Low,",
				"and **4 [hardening notes](#hardening-index)**.",
			)
			Expect(r.Summary).To(Equal(report.Counts{
				Scored: 7, Critical: 1, High: 2, Medium: 3, Low: 1, Hardening: 4, Found: true,
			}))
		})

		It("reads a sentence wrapped across lines", func() {
			r := parse(
				"**7 scored findings**: 1 Critical, 2 High, 3 Medium",
				"and 1 Low, and **4 [hardening notes](#hardening-index)**.",
			)
			Expect(r.Summary).To(Equal(report.Counts{
				Scored: 7, Critical: 1, High: 2, Medium: 3, Low: 1, Hardening: 4, Found: true,
			}))
		})

		DescribeTable("is not found",
			func(lines ...string) {
				Expect(parse(lines...).Summary.Found).To(BeFalse())
			},
			Entry("without the hardening sentence",
				"**7 scored findings**: 1 Critical, 2 High, 3 Medium and 1 Low."),
			Entry("without the scored sentence",
				"**4 [hardening notes](#hardening-index)**."),
			Entry("in another wording",
				"**1 scored finding**: 0 Critical, 1 High, 0 Medium and 0 Low, and **0 [hardening notes](#hardening-index)**."),
		)
	})

	It("reads the hardening index and each hardening note", Label("TestParseHardeningIndexAndNotes"), func() {
		r := parse(
			"## Hardening",
			"",
			`<a id="hardening-index"></a>`,
			"",
			"| ID | Note |",
			"|---|---|",
			"| [H-01](#h-01) | Widget logs request headers |",
			"| [H-02](#h-02) | Widget temp files are world-readable |",
			"",
			`<a id="h-01"></a>`,
			"",
			"#### H-01 — Widget logs request headers",
			"",
			"Prose.",
			"",
			`<a id="h-02"></a>`,
			"",
			"#### H-02 — Widget temp files are world-readable",
		)
		Expect(r.HardeningIndex).To(Equal([]string{"H-01", "H-02"}))
		Expect(r.Hardening).To(Equal([]report.Section{
			{ID: "H-01", Title: "Widget logs request headers", Anchor: "h-01", Line: 12},
			{ID: "H-02", Title: "Widget temp files are world-readable", Anchor: "h-02", Line: 18},
		}))
		Expect(r.Findings).To(BeEmpty())
	})

	It("reads the hardening index past a heading that follows its anchor", func() {
		r := parse(
			`<a id="hardening-index"></a>`,
			"",
			"## Hardening notes",
			"",
			"| ID | Note |",
			"|---|---|",
			"| [H-01](#h-01) | Widget logs request headers |",
		)
		Expect(r.HardeningIndex).To(Equal([]string{"H-01"}))
	})

	It("reads the hardening index past an anchor line stacked after its anchor", Label("TestParseHardeningIndexAnchorStack"), func() {
		r := parse(
			`<a id="hardening-index"></a>`,
			`<a id="hardening-notes"></a>`,
			"",
			"### Index",
			"",
			"| ID | Note |",
			"|---|---|",
			"| [H-01](#h-01) | Widget logs request headers |",
		)
		Expect(r.HardeningIndex).To(Equal([]string{"H-01"}))
	})

	It("finds no hardening index when prose follows its anchor before any table", func() {
		r := parse(
			`<a id="hardening-index"></a>`,
			"",
			"## Hardening notes",
			"",
			"None.",
			"",
			"## Not reported",
			"",
			"| ID | Status | Reason |",
			"|---|---|---|",
			"| F-04 | refuted | Not reachable. |",
		)
		Expect(r.HardeningIndex).To(BeEmpty())
	})

	It("finds every placeholder, inside fenced code too", Label("TestParsePlaceholders"), func() {
		r := parse(
			"# {{REPORT_TITLE}}",
			"",
			"Target: {{TARGET_REPO}} at {{TARGET_REF}}.",
			"Template syntax {{ .Values.name }} and {{lowercase}} are not placeholders.",
			fence,
			"{{IN_FENCE}}",
			fence,
		)
		Expect(r.Placeholders).To(Equal([]report.Pos{{Line: 1}, {Line: 3}, {Line: 3}, {Line: 6}}))
	})

	It("collects explicit anchors and internal links outside fenced code", func() {
		r := parse(
			`<a id="f-01"></a>`,
			"See [F-01](#f-01) and [the index](#hardening-index).",
			fence,
			"[ignored](#in-fence)",
			`<a id="in-fence"></a>`,
			fence,
		)
		Expect(r.Anchors).To(Equal(map[string]int{"f-01": 1}))
		Expect(r.InternalLinks).To(Equal([]report.Link{
			{Target: "f-01", Line: 2},
			{Target: "hardening-index", Line: 2},
		}))
	})

	Describe("code links", func() {
		DescribeTable("reads each form", Label("TestParseCodeLinkForms"),
			func(url string, want report.CodeLink) {
				r := parse("See [the code](" + url + ").")
				want.Pos = report.Pos{Line: 1}
				Expect(r.CodeLinks).To(Equal([]report.CodeLink{want}))
			},
			Entry("?plain=1 with a line range",
				"https://github.com/example/widget/blob/0123abc/cmd/widget/main.go?plain=1#L10-L20",
				report.CodeLink{
					Host: "github.com", Owner: "example", Repo: "widget", Ref: "0123abc",
					Path: "cmd/widget/main.go", L1: 10, L2: 20,
				}),
			Entry("a single line",
				"https://github.com/example/widget/blob/v1.2.0/go.mod#L7",
				report.CodeLink{
					Host: "github.com", Owner: "example", Repo: "widget", Ref: "v1.2.0",
					Path: "go.mod", L1: 7, L2: 7,
				}),
			Entry("a range",
				"https://github.com/example/widget/blob/0123abc/internal/export/export.go#L3-L9",
				report.CodeLink{
					Host: "github.com", Owner: "example", Repo: "widget", Ref: "0123abc",
					Path: "internal/export/export.go", L1: 3, L2: 9,
				}),
			Entry("no anchor",
				"https://github.com/example/widget/blob/0123abc/README.md",
				report.CodeLink{
					Host: "github.com", Owner: "example", Repo: "widget", Ref: "0123abc",
					Path: "README.md",
				}),
			Entry("a non-GitHub host",
				"https://gitlab.example.org/example/widget/-/blob/v1.2.0/src/app.go#L4-8",
				report.CodeLink{
					Host: "gitlab.example.org", Owner: "example", Repo: "widget", Ref: "v1.2.0",
					Path: "src/app.go", L1: 4, L2: 8,
				}),
			Entry("a range with column anchors",
				"https://github.com/example/widget/blob/0123abc/main.go#L5C3-L7C10",
				report.CodeLink{
					Host: "github.com", Owner: "example", Repo: "widget", Ref: "0123abc",
					Path: "main.go", L1: 5, L2: 7,
				}),
			Entry("a non-GitHub /src/ link",
				"https://gitea.example.net/example/widget/src/commit/0123abc/main.go#L5",
				report.CodeLink{
					Host: "gitea.example.net", Owner: "example", Repo: "widget", Ref: "0123abc",
					Path: "main.go", L1: 5, L2: 5,
				}),
			Entry("a raw.githubusercontent.com link",
				"https://raw.githubusercontent.com/example/widget/0123abc/src/main.go",
				report.CodeLink{
					Host: "raw.githubusercontent.com", Owner: "example", Repo: "widget", Ref: "0123abc",
					Path: "src/main.go",
				}),
			Entry("a github.com blame link",
				"https://github.com/example/widget/blame/v1.2.0/src/app.go#L4-L8",
				report.CodeLink{
					Host: "github.com", Owner: "example", Repo: "widget", Ref: "v1.2.0",
					Path: "src/app.go", L1: 4, L2: 8,
				}),
			Entry("a github.com raw link",
				"https://github.com/example/widget/raw/0123abc/src/main.go",
				report.CodeLink{
					Host: "github.com", Owner: "example", Repo: "widget", Ref: "0123abc",
					Path: "src/main.go",
				}),
			Entry("a non-GitHub /tree/ link",
				"https://gitlab.example.org/example/widget/-/tree/v1.2.0/src",
				report.CodeLink{
					Host: "gitlab.example.org", Owner: "example", Repo: "widget", Ref: "v1.2.0",
					Path: "src",
				}),
			Entry("a ref with a slash in it, cut at its first element",
				"https://github.com/example/widget-lib/blob/lib/v0.2.0/src/x.go#L3",
				report.CodeLink{
					Host: "github.com", Owner: "example", Repo: "widget-lib", Ref: "lib",
					Path: "v0.2.0/src/x.go", L1: 3, L2: 3,
				}),
		)

		DescribeTable("re-splits the ref and path at a ref the caller names", Label("TestCodeLinkPathAt"),
			func(url, ref, wantPath string, wantOK bool) {
				r := parse(url)
				Expect(r.CodeLinks).To(HaveLen(1))
				path, ok := r.CodeLinks[0].PathAt(ref)
				Expect([]any{path, ok}).To(Equal([]any{wantPath, wantOK}), "PathAt(%q)", ref)
			},
			Entry("a ref with a slash in it",
				"https://github.com/example/widget-lib/blob/lib/v0.2.0/src/x.go", "lib/v0.2.0", "src/x.go", true),
			Entry("the one element the parser read as the ref",
				"https://github.com/example/widget-lib/blob/lib/v0.2.0/src/x.go", "lib", "v0.2.0/src/x.go", true),
			Entry("a ref that ends inside a path element",
				"https://github.com/example/widget-lib/blob/lib/v0.2.0/src/x.go", "lib/v0.2", "", false),
			Entry("a ref the link does not start with",
				"https://github.com/example/widget-lib/blob/lib/v0.2.0/src/x.go", "v0.2.0", "", false),
			Entry("a link that is the ref alone",
				"https://github.com/example/widget-lib/blob/lib/v0.2.0", "lib/v0.2.0", "", true),
		)

		It("reads a line that opens with an inline code span as prose, not a fence", func() {
			r := parse(
				fence+"widget export"+fence+" follows https://github.com/example/widget/blob/0123abc/main.go#L5",
				"See https://github.com/example/widget/blob/0123abc/main.go#L6.",
			)
			Expect(r.CodeLinks).To(HaveLen(2))
			Expect(r.CodeLinks[0].L1).To(Equal(5))
			Expect(r.CodeLinks[1].L1).To(Equal(6))
		})

		It("finds a code link in link text, bare text and a table cell, once per line", func() {
			url := "https://github.com/example/widget/blob/0123abc/main.go#L5"
			r := parse(
				"Intro.",
				"["+url+"]("+url+")",
				"",
				"Bare: "+url+".",
				"| **Where** | `"+url+"` |",
			)
			Expect(r.CodeLinks).To(HaveLen(3))
			for i, line := range []int{2, 4, 5} {
				Expect(r.CodeLinks[i].Line).To(Equal(line), "code link %d", i)
				Expect(r.CodeLinks[i].Path).To(Equal("main.go"), "code link %d", i)
				Expect(r.CodeLinks[i].L1).To(Equal(5), "code link %d", i)
			}
		})

		It("reads a line number too large for an int as past the end of any file", func() {
			r := parse("https://github.com/example/widget/blob/0123abc/main.go#L99999999999999999999")
			Expect(r.CodeLinks).To(HaveLen(1))
			Expect(r.CodeLinks[0].L1).To(Equal(math.MaxInt))
		})

		It("ignores links that are not code links, and links in fenced code", func() {
			r := parse(
				"https://github.com/example/widget/issues/12",
				"https://github.com/example/widget/tree/main/src/widget",
				"https://example.com/docs/widget",
				fence,
				"https://github.com/example/widget/blob/0123abc/main.go#L5",
				fence,
			)
			Expect(r.CodeLinks).To(BeEmpty())
		})
	})
})
