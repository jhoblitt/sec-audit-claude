package cvss_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/cvss"
)

const (
	vec31 = "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N"
	vec40 = "CVSS:4.0/AV:L/AC:L/AT:N/PR:L/UI:N/VC:H/VI:L/VA:N/SC:N/SI:N/SA:N"
)

var _ = Describe("Score31", func() {
	It("scores a known vector", Label("TestScore31Known"), func() {
		score, sev, err := cvss.Score31(vec31)
		Expect(err).NotTo(HaveOccurred())
		Expect(score).To(Equal(7.1))
		Expect(sev).To(Equal("High"))
	})

	DescribeTable("names each severity band in title case",
		func(vector string, wantScore float64, wantSev string) {
			score, sev, err := cvss.Score31(vector)
			Expect(err).NotTo(HaveOccurred())
			Expect(score).To(Equal(wantScore))
			Expect(sev).To(Equal(wantSev))
		},
		Entry("no impact", "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:N", 0.0, "None"),
		Entry("low", "CVSS:3.1/AV:P/AC:H/PR:H/UI:R/S:U/C:L/I:N/A:N", 1.6, "Low"),
		Entry("medium", "CVSS:3.1/AV:A/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", 5.5, "Medium"),
		Entry("critical", "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", 9.3, "Critical"),
	)
})

var _ = Describe("canonical form", Label("TestScoreCanonicalForm"), func() {
	DescribeTable("rejects a vector not in canonical form, naming that form",
		func(score func(string) (float64, string, error), vector, canon string) {
			_, sev, err := score(vector)
			Expect(err).To(MatchError(ContainSubstring("is not in canonical form %q", canon)))
			Expect(sev).To(BeEmpty())
		},
		Entry("3.1: metrics out of order", cvss.Score31, "CVSS:3.1/AC:L/AV:N/PR:L/UI:N/S:U/C:H/I:L/A:N", vec31),
		Entry("3.1: a metric set to X", cvss.Score31, vec31+"/E:X", vec31),
		Entry("4.0: a metric set to X", cvss.Score40, vec40+"/E:X", vec40),
	)

	DescribeTable("accepts optional metrics in canonical form",
		func(score func(string) (float64, string, error), vector string) {
			_, _, err := score(vector)
			Expect(err).NotTo(HaveOccurred())
		},
		Entry("3.1: temporal metrics", cvss.Score31, vec31+"/E:P/RL:O/RC:C"),
		Entry("4.0: a threat metric", cvss.Score40, vec40+"/E:A"),
	)
})

var _ = Describe("Score40", func() {
	It("scores a known vector", Label("TestScore40Known"), func() {
		score, sev, err := cvss.Score40(vec40)
		Expect(err).NotTo(HaveOccurred())
		Expect(score).To(Equal(6.9))
		Expect(sev).To(Equal("Medium"))
	})

	It("names a vector with no impact None", func() {
		score, sev, err := cvss.Score40("CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:N/VI:N/VA:N/SC:N/SI:N/SA:N")
		Expect(err).NotTo(HaveOccurred())
		Expect(score).To(Equal(0.0))
		Expect(sev).To(Equal("None"))
	})

	// The Python cvss package scores this vector 8.6. The gate is the
	// calculator of record, so its value is the audit's; a go-cvss upgrade
	// that moves it shows here.
	It("scores a vector other calculators put 0.1 higher at the gate's value", Label("TestScore40CalculatorOfRecord"), func() {
		score, sev, err := cvss.Score40("CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:P/VC:H/VI:L/VA:N/SC:H/SI:H/SA:H")
		Expect(err).NotTo(HaveOccurred())
		Expect(score).To(Equal(8.5))
		Expect(sev).To(Equal("High"))
	})
})

var _ = Describe("Score", func() {
	DescribeTable("scores a vector of the version its prefix names",
		func(vector string, wantScore float64, wantSev string) {
			score, sev, err := cvss.Score(vector)
			Expect(err).NotTo(HaveOccurred())
			Expect(score).To(Equal(wantScore))
			Expect(sev).To(Equal(wantSev))
		},
		Entry("3.1", vec31, 7.1, "High"),
		Entry("4.0", vec40, 6.9, "Medium"),
	)

	DescribeTable("rejects a vector it cannot score",
		func(vector, want string) {
			_, sev, err := cvss.Score(vector)
			Expect(err).To(MatchError(ContainSubstring(want)))
			Expect(sev).To(BeEmpty())
		},
		Entry("free text", "garbage", "is not a CVSS 3.1 or 4.0 vector"),
		Entry("empty", "", "is not a CVSS 3.1 or 4.0 vector"),
		Entry("CVSS 3.0", "CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N", "is not a CVSS 3.1 or 4.0 vector"),
		Entry("a 3.1 vector with a base metric missing", "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L", "parsing CVSS 3.1 vector"),
		Entry("a 4.0 vector with a base metric missing", "CVSS:4.0/AV:L/AC:L/AT:N/PR:L/UI:N/VC:H/VI:L/VA:N/SC:N/SI:N", "parsing CVSS 4.0 vector"),
	)
})

var _ = DescribeTable("Format writes a score with one decimal",
	func(score float64, want string) {
		Expect(cvss.Format(score)).To(Equal(want))
	},
	Entry("no impact", 0.0, "0.0"),
	Entry("a fractional score", 7.1, "7.1"),
	Entry("a whole score", 10.0, "10.0"),
)

var _ = DescribeTable("rejects a vector that is not one of its version",
	Label("TestScoreRejectsGarbage"),
	func(score func(string) (float64, string, error), vector, want string) {
		_, sev, err := score(vector)
		Expect(err).To(MatchError(ContainSubstring(want)))
		Expect(sev).To(BeEmpty())
	},
	Entry("3.1: free text", cvss.Score31, "garbage", "CVSS 3.1 vector"),
	Entry("3.1: empty", cvss.Score31, "", "CVSS 3.1 vector"),
	Entry("3.1: a 4.0 vector", cvss.Score31, vec40, "CVSS 3.1 vector"),
	Entry("3.1: a base metric missing", cvss.Score31, "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L", "CVSS 3.1 vector"),
	Entry("3.1: an unknown metric value", cvss.Score31, "CVSS:3.1/AV:X/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N", "CVSS 3.1 vector"),
	Entry("4.0: free text", cvss.Score40, "garbage", "CVSS 4.0 vector"),
	Entry("4.0: a 3.1 vector", cvss.Score40, vec31, "CVSS 4.0 vector"),
	Entry("4.0: a base metric missing", cvss.Score40, "CVSS:4.0/AV:L/AC:L/AT:N/PR:L/UI:N/VC:H/VI:L/VA:N/SC:N/SI:N", "CVSS 4.0 vector"),
)
