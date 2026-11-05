package table_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/table"
)

const (
	vec31 = "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N"
	vec40 = "CVSS:4.0/AV:L/AC:L/AT:N/PR:L/UI:N/VC:H/VI:L/VA:N/SC:N/SI:N/SA:N"
)

var fixedColumns = []string{
	"id", "cluster", "status", "cvss31", "score31", "sev31", "cvss40", "score40", "sev40",
	"title", "principal", "default_exposure", "found_by", "confirmed_by",
}

// stdHeader is the header of a table whose releases are v1.2 and main.
var stdHeader = header(slices.Concat(fixedColumns, []string{"rel:v1.2", "rel:main"})...)

func header(cols ...string) string {
	return strings.Join(cols, "\t")
}

func row(fields ...string) string {
	return strings.Join(fields, "\t")
}

// scoredRow is a well-formed scored row for a table whose releases are
// v1.2 and main.
func scoredRow(id string) []string {
	return []string{
		id, "C-001", "scored", vec31, "7.1", "High", vec40, "6.9", "Medium",
		"Widget export follows symlinks", "authenticated user", "on", "claude", "gpt",
		"PRESENT", "PARTIAL",
	}
}

func hardeningRow(id string) []string {
	return []string{
		id, "-", "hardening", "-", "-", "-", "-", "-", "-",
		"Widget logs request headers", "-", "-", "claude", "-",
		"-", "-",
	}
}

var _ = Describe("Load", func() {
	var dir string

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
	})

	write := func(lines ...string) string {
		path := filepath.Join(dir, "findings.tsv")
		Expect(os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600)).To(Succeed())
		return path
	}

	DescribeTable("rejects a header with a missing or reordered column, naming it",
		Label("TestTableLoadHeaderOrder"),
		func(cols []string, want string) {
			_, err := table.Load(write(header(cols...)))
			Expect(err).To(MatchError(ContainSubstring(want)))
		},
		Entry("a missing column",
			[]string{
				"id", "cluster", "status", "cvss31", "score31", "cvss40", "score40", "sev40",
				"title", "principal", "default_exposure", "found_by", "confirmed_by", "rel:main",
			},
			`missing column "sev31"`),
		Entry("two swapped columns",
			[]string{
				"id", "cluster", "status", "score31", "cvss31", "sev31", "cvss40", "score40", "sev40",
				"title", "principal", "default_exposure", "found_by", "confirmed_by", "rel:main",
			},
			`column "cvss31" out of order`),
		Entry("a header that ends early",
			[]string{
				"id", "cluster", "status", "cvss31", "score31", "sev31", "cvss40", "score40", "sev40",
				"title", "principal", "default_exposure", "found_by",
			},
			`missing column "confirmed_by"`),
		Entry("a trailing column that is not a release",
			slices.Concat(fixedColumns, []string{"notes"}),
			`column 15 is "notes", want rel:<name>`),
		Entry("a release named twice",
			slices.Concat(fixedColumns, []string{"rel:main", "rel:main"}),
			`release "main" named twice`),
	)

	It("accepts a scored row without a CVSS 4.0 vector", Label("TestTableAcceptsMissing40"), func() {
		f := scoredRow("F-01")
		f[6], f[7], f[8] = "-", "-", "-"
		tbl, err := table.Load(write(stdHeader, row(f...)))
		Expect(err).NotTo(HaveOccurred())
		Expect(tbl.Rows).To(HaveLen(1))
		Expect(tbl.Rows[0].CVSS40).To(Equal("-"))
		Expect(tbl.Rows[0].Score40).To(Equal("-"))
		Expect(tbl.Rows[0].Sev40).To(Equal("-"))
		Expect(tbl.Rows[0].CVSS31).To(Equal(vec31))
	})

	It("rejects an unknown status, naming the row and the value", Label("TestTableRejectsBadStatus"), func() {
		f := scoredRow("F-01")
		f[2] = "open"
		_, err := table.Load(write(stdHeader, row(f...)))
		Expect(err).To(MatchError(And(ContainSubstring("findings.tsv:2"), ContainSubstring(`status "open"`))))
	})

	It("reads the rel:<name> columns as releases, in header order", Label("TestTableRelColumns"), func() {
		tbl, err := table.Load(write(stdHeader, row(scoredRow("F-01")...)))
		Expect(err).NotTo(HaveOccurred())
		Expect(tbl.Releases).To(Equal([]string{"v1.2", "main"}))
		Expect(tbl.Rows[0].Rel).To(Equal(map[string]string{"v1.2": "PRESENT", "main": "PARTIAL"}))
	})

	It("loads every column of every row and indexes the rows by ID", func() {
		tbl, err := table.Load(write(stdHeader, row(scoredRow("F-01")...), row(hardeningRow("H-01")...)))
		Expect(err).NotTo(HaveOccurred())
		Expect(tbl.Rows).To(Equal([]table.Row{
			{
				ID: "F-01", Cluster: "C-001", Status: "scored",
				CVSS31: vec31, Score31: "7.1", Sev31: "High",
				CVSS40: vec40, Score40: "6.9", Sev40: "Medium",
				Title: "Widget export follows symlinks", Principal: "authenticated user", Default: "on",
				FoundBy: "claude", ConfirmedBy: "gpt",
				Rel: map[string]string{"v1.2": "PRESENT", "main": "PARTIAL"},
			},
			{
				ID: "H-01", Cluster: "-", Status: "hardening",
				CVSS31: "-", Score31: "-", Sev31: "-",
				CVSS40: "-", Score40: "-", Sev40: "-",
				Title: "Widget logs request headers", Principal: "-", Default: "-",
				FoundBy: "claude", ConfirmedBy: "-",
				Rel: map[string]string{"v1.2": "-", "main": "-"},
			},
		}))
		Expect(tbl.ByID).To(HaveLen(2))
		Expect(tbl.ByID["H-01"]).To(BeIdenticalTo(&tbl.Rows[1]))
	})

	It("keeps every row of a duplicated ID and indexes the first", func() {
		first, second := scoredRow("F-01"), scoredRow("F-01")
		second[9] = "A second title"
		tbl, err := table.Load(write(stdHeader, row(first...), row(second...)))
		Expect(err).NotTo(HaveOccurred())
		Expect(tbl.Rows).To(HaveLen(2))
		Expect(tbl.ByID["F-01"].Title).To(Equal("Widget export follows symlinks"))
	})

	It("tolerates CRLF line endings", func() {
		path := filepath.Join(dir, "findings.tsv")
		data := stdHeader + "\r\n" + row(scoredRow("F-01")...) + "\r\n"
		Expect(os.WriteFile(path, []byte(data), 0o600)).To(Succeed())
		tbl, err := table.Load(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(tbl.Rows[0].Rel).To(HaveKeyWithValue("main", "PARTIAL"))
	})

	It("reports a missing file", func() {
		_, err := table.Load(filepath.Join(dir, "absent.tsv"))
		Expect(err).To(MatchError(os.ErrNotExist))
	})

	DescribeTable("rejects a malformed row, naming what is wrong",
		func(edit func([]string) []string, want string) {
			_, err := table.Load(write(stdHeader, row(edit(scoredRow("F-01"))...)))
			Expect(err).To(MatchError(And(ContainSubstring("findings.tsv:2"), ContainSubstring(want))))
		},
		Entry("too few fields",
			func(f []string) []string { return f[:len(f)-1] },
			"15 fields, want 16"),
		Entry("an empty cell",
			func(f []string) []string { f[3] = ""; return f },
			`empty cvss31; use "-"`),
		Entry("an ID of -",
			func(f []string) []string { f[0] = "-"; return f },
			"row has no id"),
		Entry("a scored row without a CVSS 3.1 vector",
			func(f []string) []string { f[3], f[4], f[5] = "-", "-", "-"; return f },
			"scored row has no cvss31"),
		Entry("a score with two decimals",
			func(f []string) []string { f[4] = "7.10"; return f },
			`score31 "7.10" is not a score with one decimal`),
		Entry("a score above 10",
			func(f []string) []string { f[7] = "10.5"; return f },
			`score40 "10.5" is not a score with one decimal`),
		Entry("an upper-case severity",
			func(f []string) []string { f[5] = "HIGH"; return f },
			`sev31 "HIGH" is not one of None, Low, Medium, High, Critical`),
		Entry("a 4.0 score without a 4.0 vector",
			func(f []string) []string { f[6] = "-"; return f },
			"cvss40 is - but score40 or sev40 is set"),
		Entry("a release verdict of - on a scored row",
			func(f []string) []string { f[15] = "-"; return f },
			`rel:main "-" is not one of PRESENT, ABSENT, PARTIAL`),
		Entry("a lower-case release verdict",
			func(f []string) []string { f[14] = "present"; return f },
			`rel:v1.2 "present" is not one of PRESENT, ABSENT, PARTIAL`),
	)

	It("rejects a non-scored row that carries scoring", func() {
		f := scoredRow("H-01")
		f[2] = "hardening"
		_, err := table.Load(write(stdHeader, row(f...)))
		Expect(err).To(MatchError(ContainSubstring("hardening row carries cvss31")))
	})
})
