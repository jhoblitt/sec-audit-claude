package gate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("the placeholders check", func() {
	It("reports every placeholder left in the report", func(ctx SpecContext) {
		f := newFixture(ctx, "clean")
		replace(&f.report, "- **Date:** 2026-11-05", "- **Date:** {{AUDIT_DATE}} ({{AUDIT_TZ}})")
		f.report += "\n```text\n{{REPRO_STEPS}}\n```\n"
		Expect(f.check(ctx)).To(ConsistOf(
			problem("placeholders", f.reportLine("{{AUDIT_DATE}}"), "placeholder"),
			problem("placeholders", f.reportLine("{{AUDIT_DATE}}"), "placeholder"),
			problem("placeholders", f.reportLine("{{REPRO_STEPS}}"), "placeholder"),
		))
	})
})
