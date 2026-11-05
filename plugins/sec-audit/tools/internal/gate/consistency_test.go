package gate_test

import (
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	f01Overview = "| [F-01](#f-01) | 7.1 | 6.9 | High | Widget export follows symlinks out of the export root | yes / yes | claude, gpt | default |"
	f02Overview = "| [F-02](#f-02) | 5.5 | 5.1 | Medium | Widget cache keys collide across users | partly / yes | gpt | default |"
	f01CVSS31   = "C:H/I:L/A:N` — **7.1 High** |"
	f02Heading  = "### F-02 — Widget cache keys collide across users"
	f04Section  = "<a id=\"f-04\"></a>\n\n### F-04 — Widget upload accepts unbounded bodies\n\nFixed before this audit.\n\n"
)

var _ = Describe("the consistency check", func() {
	var f *fixture

	BeforeEach(func(ctx SpecContext) {
		f = newFixture(ctx, "clean")
	})

	// swap exchanges the two report lines that start with a and b.
	swap := func(a, b string) {
		GinkgoHelper()
		lines := strings.Split(f.report, "\n")
		i := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, a) })
		j := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, b) })
		Expect([]int{i, j}).NotTo(ContainElement(-1), "lines starting %q and %q", a, b)
		lines[i], lines[j] = lines[j], lines[i]
		f.report = strings.Join(lines, "\n")
	}

	DescribeTable("recomputes every score and compares every copy of it", Label("TestConsistencyScoreMismatch"),
		func(ctx SpecContext, edit func(), where func() string, message string) {
			edit()
			Expect(f.check(ctx)).To(ConsistOf(problem("consistency", where(), message)))
		},
		Entry("a score off by 0.1 everywhere, against its vector",
			func() {
				replace(&f.table, "\t7.1\tHigh\tCVSS:4.0/AV:L/AC:L/AT:N", "\t7.2\tHigh\tCVSS:4.0/AV:L/AC:L/AT:N")
				replace(&f.report, f01CVSS31, "C:H/I:L/A:N` — **7.2 High** |")
				replace(&f.report, "| [F-01](#f-01) | 7.1 |", "| [F-01](#f-01) | 7.2 |")
			},
			func() string { return f.opts.Table },
			"F-01: score31 is 7.2 but its vector scores 7.1"),
		Entry("a CVSS 4.0 score off by 0.1 everywhere, against its vector",
			func() {
				replace(&f.table, "\t5.1\tMedium\t", "\t5.2\tMedium\t")
				replace(&f.report, "SC:N/SI:N/SA:N` — **5.1 Medium** |", "SC:N/SI:N/SA:N` — **5.2 Medium** |")
				replace(&f.report, "| 5.5 | 5.1 |", "| 5.5 | 5.2 |")
			},
			func() string { return f.opts.Table },
			"F-02: score40 is 5.2 but its vector scores 5.1"),
		Entry("a section's score off by 0.1",
			func() { replace(&f.report, f01CVSS31, "C:H/I:L/A:N` — **7.0 High** |") },
			func() string { return f.reportLine("### F-01") },
			`F-01: section CVSS 3.1 score is "7.0", table has "7.1"`),
		Entry("a section's severity",
			func() { replace(&f.report, "— **5.5 Medium** |", "— **5.5 High** |") },
			func() string { return f.reportLine(f02Heading) },
			`F-02: section CVSS 3.1 severity is "High", table has "Medium"`),
		Entry("an overview score off by 0.1",
			func() { replace(&f.report, "| [F-01](#f-01) | 7.1 | 6.9 |", "| [F-01](#f-01) | 7.1 | 7.0 |") },
			func() string { return f.opts.Report },
			`F-01: overview CVSS 4.0 is "7.0", table has "6.9"`),
		Entry("a vector that does not parse, copied faithfully",
			func() {
				replace(&f.table, "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N", "CVSS:3.1/AV:X/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N")
				replace(&f.report, "`CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N`", "`CVSS:3.1/AV:X/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N`")
			},
			func() string { return f.opts.Table },
			"F-01: cvss31: parsing CVSS 3.1 vector"),
	)

	DescribeTable("orders the overview by CVSS 3.1, then CVSS 4.0, descending, then ID", Label("TestConsistencyOverviewOrder"),
		func(ctx SpecContext, a, b, message string) {
			swap(a, b)
			Expect(f.check(ctx)).To(ConsistOf(problem("consistency", f.opts.Report, message)))
		},
		Entry("a lower CVSS 3.1 score first", "| [F-03](#f-03) |", "| [F-02](#f-02) |", "overview lists F-02 before F-03"),
		Entry("a CVSS 3.1 tie broken the wrong way", "| [F-01](#f-01) |", "| [F-03](#f-03) |", "overview lists F-03 before F-01"),
	)

	It("breaks a tie by ID, comparing the numbers in IDs of different widths", func(ctx SpecContext) {
		for old, repl := range map[string]string{"F-01": "F-9", "f-01": "f-9", "F-03": "F-10", "f-03": "f-10"} {
			f.report = strings.ReplaceAll(f.report, old, repl)
			f.table = strings.ReplaceAll(f.table, old, repl)
		}
		replace(&f.table, "CVSS:4.0/AV:L/AC:L/AT:P/PR:L/UI:N/VC:L/VI:H/VA:N/SC:N/SI:N/SA:N\t5.8", "CVSS:4.0/AV:L/AC:L/AT:N/PR:L/UI:N/VC:H/VI:L/VA:N/SC:N/SI:N/SA:N\t6.9")
		replace(&f.report, "`CVSS:4.0/AV:L/AC:L/AT:P/PR:L/UI:N/VC:L/VI:H/VA:N/SC:N/SI:N/SA:N` — **5.8 Medium**", "`CVSS:4.0/AV:L/AC:L/AT:N/PR:L/UI:N/VC:H/VI:L/VA:N/SC:N/SI:N/SA:N` — **6.9 Medium**")
		replace(&f.report, "| 7.1 | 5.8 |", "| 7.1 | 6.9 |")
		Expect(f.check(ctx)).To(BeEmpty())

		swap("| [F-9](#f-9) |", "| [F-10](#f-10) |")
		Expect(f.check(ctx)).To(ConsistOf(problem("consistency", f.opts.Report, "overview lists F-10 before F-9")))
	})

	DescribeTable("reports a retired or refuted ID wherever it reappears", Label("TestConsistencyRetiredIDReappears"),
		func(ctx SpecContext, edit func(), where func() string, message string) {
			edit()
			Expect(f.check(ctx)).To(ConsistOf(problem("consistency", where(), message)))
		},
		Entry("as a finding section",
			func() { replace(&f.report, "<a id=\"hardening\"></a>", f04Section+"<a id=\"hardening\"></a>") },
			func() string { return f.reportLine("### F-04") },
			"F-04: retired, but the report has a finding section for it"),
		Entry("as an overview row",
			func() {
				replace(&f.report, f02Overview, f02Overview+"\n| F-04 | — | — | — | Widget upload accepts unbounded bodies | no / no | claude | default |")
			},
			func() string { return f.opts.Report },
			"F-04: retired, but the report has an overview row for it"),
		Entry("in the hardening index",
			func() {
				replace(&f.report, "| [H-01](#h-01) | Widget server logs request headers at debug level |",
					"| [H-01](#h-01) | Widget server logs request headers at debug level |\n| F-04 | Widget upload accepts unbounded bodies |")
			},
			func() string { return f.opts.Report },
			"F-04: retired, but the hardening index lists it"),
		Entry("refuted, as a finding section",
			func() {
				replace(&f.table, "\tretired\t", "\trefuted\t")
				replace(&f.report, "| F-04 | retired |", "| F-04 | refuted |")
				replace(&f.report, "<a id=\"hardening\"></a>", f04Section+"<a id=\"hardening\"></a>")
			},
			func() string { return f.reportLine("### F-04") },
			"F-04: refuted, but the report has a finding section for it"),
	)

	// insertSection adds a section under heading before the hardening part,
	// after an anchor line for anchor unless anchor is "".
	insertSection := func(anchor, heading string) {
		GinkgoHelper()
		section := heading + "\n\nFixed before this audit.\n\n"
		if anchor != "" {
			section = "<a id=\"" + anchor + "\"></a>\n\n" + section
		}
		replace(&f.report, "<a id=\"hardening\"></a>", section+"<a id=\"hardening\"></a>")
	}
	malformed := func(heading, id, form string) any {
		GinkgoHelper()
		return problem("consistency", f.reportLine(heading), id+`: finding section heading is not in the form "`+form+`"`)
	}

	DescribeTable("reports a retired ID's section in a heading of any other form", Label("TestConsistencyMalformedHeading"),
		func(ctx SpecContext, anchor, heading string, also func(heading string) []any) {
			insertSection(anchor, heading)
			want := append([]any{
				malformed(heading, "F-04", "### F-04 — <title>"),
				problem("consistency", f.reportLine(heading), "F-04: retired, but the report has a finding section for it"),
			}, also(heading)...)
			Expect(f.check(ctx)).To(ConsistOf(want...))
		},
		Entry("a colon", "f-04", "### F-04: Widget upload accepts unbounded bodies", func(string) []any { return nil }),
		Entry("a hyphen", "f-04", "### F-04 - Widget upload accepts unbounded bodies", func(string) []any { return nil }),
		Entry("an en dash", "f-04", "### F-04 – Widget upload accepts unbounded bodies", func(string) []any { return nil }),
		Entry("a bold ID", "f-04", "### **F-04** — Widget upload accepts unbounded bodies", func(string) []any { return nil }),
		Entry("a bare ID", "f-04", "### F-04", func(string) []any { return nil }),
		Entry("an inline anchor and no anchor line", "", `### <a id="f-04"></a>F-04 — Widget upload accepts unbounded bodies`,
			func(heading string) []any {
				return []any{problem("consistency", f.reportLine(heading),
					`finding section heading for F-04 has no <a id="f-04"> anchor line before it`)}
			}),
	)

	DescribeTable("checks a heading of another form against its row as it does one in the form",
		func(ctx SpecContext, edit func(), want func() []any) {
			edit()
			Expect(f.check(ctx)).To(ConsistOf(want()...))
		},
		Entry("a refuted ID",
			func() {
				replace(&f.table, "\tretired\t", "\trefuted\t")
				replace(&f.report, "| F-04 | retired |", "| F-04 | refuted |")
				insertSection("f-04", "### F-04: Widget upload accepts unbounded bodies")
			},
			func() []any {
				return []any{
					malformed("### F-04:", "F-04", "### F-04 — <title>"),
					problem("consistency", f.reportLine("### F-04:"), "F-04: refuted, but the report has a finding section for it"),
				}
			}),
		Entry("an ID with no row",
			func() { insertSection("f-09", "### F-09: Widget extra") },
			func() []any {
				return []any{
					malformed("### F-09:", "F-09", "### F-09 — <title>"),
					problem("consistency", f.reportLine("### F-09:"), "F-09: the report has a finding section for it, but the table has no row"),
				}
			}),
		Entry("a second section for a scored ID",
			func() { insertSection("f-02", "### F-02: Widget cache keys collide across users") },
			func() []any {
				return []any{
					malformed("### F-02:", "F-02", "### F-02 — <title>"),
					problem("consistency", f.reportLine("### F-02:"), "F-02: second finding section, after the one at line "),
				}
			}),
		Entry("a scored ID's only section, which it counts",
			func() {
				replace(&f.report, "### F-01 — Widget export follows symlinks out of the export root",
					"### F-01: Widget export follows symlinks out of the export root")
			},
			func() []any { return []any{malformed("### F-01:", "F-01", "### F-01 — <title>")} }),
		Entry("a hardening note",
			func() {
				replace(&f.report, "#### H-01 — Widget server logs request headers at debug level",
					"#### **H-01** — Widget server logs request headers at debug level")
			},
			func() []any {
				return []any{problem("consistency", f.reportLine("#### **H-01**"),
					`H-01: hardening note heading is not in the form "#### H-01 — <title>"`)}
			}),
	)

	DescribeTable("compares the summary's counts with the table's", Label("TestConsistencySummaryCounts"),
		func(ctx SpecContext, old, repl, message string) {
			replace(&f.report, old, repl)
			Expect(f.check(ctx)).To(ConsistOf(problem("consistency", f.opts.Report, message)))
		},
		Entry("a severity count", "0 Critical, 2 High", "0 Critical, 1 High", "summary says 1 High, table has 2"),
		Entry("the scored count", "**3 scored findings**", "**4 scored findings**", "summary says 4 scored findings, table has 3"),
		Entry("the hardening count", "**1 [hardening notes]", "**2 [hardening notes]", "summary says 2 hardening notes, table has 1"),
		Entry("a missing sentence", "**1 [hardening notes](#hardening-index)**", "one hardening note",
			`summary count sentences not found; want "**3 scored findings**: 0 Critical, 2 High, 1 Medium and 0 Low" and "**1 [hardening notes](#hardening-index)**"`),
	)

	Describe("sections", func() {
		It("reports a scored finding with no section", func(ctx SpecContext) {
			replace(&f.report, f02Heading+"\n", "")
			Expect(f.check(ctx)).To(ConsistOf(
				problem("consistency", f.opts.Report, "F-02: scored, but the report has no finding section for it"),
			))
		})

		It("reports a second section for one ID", func(ctx SpecContext) {
			replace(&f.report, "<a id=\"hardening\"></a>", "<a id=\"f-02\"></a>\n\n"+f02Heading+"\n\nAgain.\n\n<a id=\"hardening\"></a>")
			Expect(f.check(ctx)).To(ConsistOf(
				problem("consistency", f.reportLines(f02Heading)[1], "F-02: second finding section, after the one at line "),
			))
		})

		DescribeTable("reports a section heading with no anchor of its own, and still checks it against its row",
			func(ctx SpecContext, edit func(), heading string, also func() []any) {
				edit()
				id := strings.Fields(strings.TrimPrefix(heading, "### "))[0]
				want := append([]any{
					problem("consistency", f.reportLines(heading)[len(f.reportLines(heading))-1],
						"finding section heading for "+id+` has no <a id="`+strings.ToLower(id)+`"> anchor`),
				}, also()...)
				Expect(f.check(ctx)).To(ConsistOf(want...))
			},
			Entry("a retired ID",
				func() {
					replace(&f.report, "<a id=\"hardening\"></a>",
						"### F-04 — Widget upload accepts unbounded bodies\n\n"+
							"| **CVSS 3.1** | `CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N` — **7.1 High** |\n\n<a id=\"hardening\"></a>")
				},
				"### F-04",
				func() []any {
					return []any{problem("consistency", f.reportLine("### F-04"), "F-04: retired, but the report has a finding section for it")}
				}),
			Entry("an ID with no row",
				func() {
					replace(&f.report, "<a id=\"hardening\"></a>", "### F-09 — Widget extra\n\n<a id=\"hardening\"></a>")
				},
				"### F-09",
				func() []any {
					return []any{problem("consistency", f.reportLine("### F-09"), "F-09: the report has a finding section for it, but the table has no row")}
				}),
			Entry("a scored ID",
				func() { replace(&f.report, "<a id=\"f-02\"></a>\n\n", "") },
				f02Heading,
				func() []any {
					return []any{problem("anchors", f.reportLine(f02Overview), "link to #f-02 has no")}
				}),
			Entry("a scored ID whose anchor is in another case",
				func() { replace(&f.report, "<a id=\"f-02\"></a>", "<a id=\"F-02\"></a>") },
				f02Heading,
				func() []any {
					return []any{problem("anchors", f.reportLine(f02Overview), "link to #f-02 has no")}
				}),
			Entry("a second section for a scored ID",
				func() {
					replace(&f.report, "<a id=\"hardening\"></a>", f02Heading+"\n\nAgain.\n\n<a id=\"hardening\"></a>")
				},
				f02Heading,
				func() []any {
					return []any{problem("consistency", f.reportLines(f02Heading)[1], "F-02: second finding section, after the one at line ")}
				}),
		)

		It("reports a section with no row in the table", func(ctx SpecContext) {
			replace(&f.report, "<a id=\"hardening\"></a>", "<a id=\"f-09\"></a>\n\n### F-09 — Widget extra\n\n<a id=\"hardening\"></a>")
			Expect(f.check(ctx)).To(ConsistOf(
				problem("consistency", f.reportLine("### F-09"), "F-09: the report has a finding section for it, but the table has no row"),
			))
		})

		It("reports a hardening note written as a finding section", func(ctx SpecContext) {
			replace(&f.report, "#### H-01", "### H-01")
			Expect(f.check(ctx)).To(ConsistOf(
				problem("consistency", f.reportLine("### H-01"), "H-01: hardening, but the report has a finding section for it"),
				problem("consistency", f.opts.Report, "H-01: hardening, but the report has no hardening note for it"),
			))
		})

		It("compares a section's title and verdicts with the table's", func(ctx SpecContext) {
			replace(&f.report, "### F-03 — Widget server trusts a client-supplied role header", "### F-03 — Widget server trusts a role header")
			replace(&f.report, "v0.9 **PRESENT**; v1.0 **PRESENT**", "v0.9 **ABSENT**; v1.0 **PRESENT**; v1.1 **PRESENT**")
			replace(&f.report, "v0.9 **PARTIAL**; v1.0 **PRESENT**", "v0.9 maybe; v1.0 **PRESENT**")
			Expect(f.check(ctx)).To(ConsistOf(
				problem("consistency", f.reportLine("### F-03"), `F-03: section title is "Widget server trusts a role header", table has "Widget server trusts a client-supplied role header"`),
				problem("consistency", f.reportLine("### F-01"), `F-01: section verdict for v0.9 is "ABSENT", table has "PRESENT"`),
				problem("consistency", f.reportLine("### F-01"), `F-01: section names release "v1.1", which the table does not`),
				problem("consistency", f.reportLine(f02Heading), `F-02: section has no verdict for v0.9`),
				problem("consistency", f.reportLine(f02Heading), `F-02: section's **Affected versions** part "v0.9 maybe" is not in the form <rel> **<VERDICT>**`),
			))
		})

		It("reports a section with no CVSS 4.0 or Affected versions row", func(ctx SpecContext) {
			replace(&f.report, "| **CVSS 4.0** | `CVSS:4.0/AV:A/AC:L/AT:N/PR:N/UI:P/VC:L/VI:L/VA:L/SC:N/SI:N/SA:N` — **5.1 Medium** |\n", "")
			replace(&f.report, "| **Affected versions** | v0.9 **PARTIAL**; v1.0 **PRESENT** — v0.9 keys include the user |\n", "")
			Expect(f.check(ctx)).To(ConsistOf(
				problem("consistency", f.reportLine(f02Heading), "F-02: section has no **CVSS 4.0** row"),
				problem("consistency", f.reportLine(f02Heading), "F-02: section has no **Affected versions** row"),
			))
		})

		It("reports a CVSS cell it cannot read", func(ctx SpecContext) {
			replace(&f.report, "`CVSS:3.1/AV:A/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L` — **5.5 Medium**", "5.5 Medium")
			Expect(f.check(ctx)).To(ConsistOf(
				problem("consistency", f.reportLine(f02Heading), "F-02: section's **CVSS 3.1** cell is not in the form"),
			))
		})
	})

	Describe("the overview", func() {
		It("reports a scored finding with no overview row", func(ctx SpecContext) {
			replace(&f.report, f02Overview+"\n", "")
			Expect(f.check(ctx)).To(ConsistOf(
				problem("consistency", f.opts.Report, "F-02: scored, but the overview has no row for it"),
			))
		})

		It("reports a second overview row for one ID", func(ctx SpecContext) {
			replace(&f.report, f02Overview, f02Overview+"\n"+f02Overview)
			Expect(f.check(ctx)).To(ConsistOf(
				problem("consistency", f.opts.Report, "F-02: 2 overview rows"),
			))
		})

		It("compares each cell with the table", func(ctx SpecContext) {
			replace(&f.report, f01Overview,
				"| [F-01](#f-01) | 7.1 | 6.9 | Medium | Widget export follows links | no / yes | claude | opt-in |")
			Expect(f.check(ctx)).To(ConsistOf(
				problem("consistency", f.opts.Report, `F-01: overview severity is "Medium", table has "High"`),
				problem("consistency", f.opts.Report, `F-01: overview title is "Widget export follows links", table has "Widget export follows symlinks out of the export root"`),
				problem("consistency", f.opts.Report, `F-01: overview verdict for v0.9 is "ABSENT", table has "PRESENT"`),
				problem("consistency", f.opts.Report, `F-01: overview found by is "claude", table has "claude, gpt"`),
				problem("consistency", f.opts.Report, `F-01: overview default exposure is "opt-in", table has "default"`),
			))
		})

		It("compares the overview's releases with the table's", func(ctx SpecContext) {
			replace(&f.report, "Affected (v0.9 / v1.0)", "Affected (v0.8 / v1.0)")
			Expect(f.check(ctx)).To(ConsistOf(
				problem("consistency", f.opts.Report, "overview names releases v0.8 / v1.0, table has v0.9 / v1.0"),
			))
		})

		It("reports a report with no overview", func(ctx SpecContext) {
			replace(&f.report, "| Found by | Default exposure |", "| Found by | Exposure |")
			Expect(f.check(ctx)).To(ConsistOf(
				problem("consistency", f.opts.Report, "no finding overview"),
			))
		})
	})

	Describe("the hardening index", func() {
		It("reports a hardening note missing from it, and an entry that is no hardening note", func(ctx SpecContext) {
			replace(&f.report, "| [H-01](#h-01) | Widget server logs request headers at debug level |", "| [F-01](#f-01) | Widget export |")
			Expect(f.check(ctx)).To(ConsistOf(
				problem("consistency", f.opts.Report, "F-01: scored, but the hardening index lists it"),
				problem("consistency", f.opts.Report, "H-01: hardening, but the hardening index does not list it"),
			))
		})
	})

	It("reports an ID on more than one row once", func(ctx SpecContext) {
		lines := strings.Split(f.table, "\n")
		f.table += lines[5] + "\n"
		Expect(f.check(ctx)).To(ConsistOf(
			problem("consistency", f.opts.Table, "H-01: id appears in 2 rows"),
		))
	})
})
