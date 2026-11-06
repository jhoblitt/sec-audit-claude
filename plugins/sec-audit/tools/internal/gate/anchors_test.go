package gate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("the anchors check", func() {
	var f *fixture

	BeforeEach(func(ctx SpecContext) {
		f = newFixture(ctx, "clean")
	})

	It("reports each internal link with no anchor of its own", Label("TestAnchorsBroken"), func(ctx SpecContext) {
		replace(&f.report, "[Limitations](#limitations)", "[Limitations](#limits)")
		replace(&f.report, "[Not reported](#not-reported)", "[Not reported](#)")
		Expect(f.check(ctx)).To(ConsistOf(
			problem("anchors", f.reportLine("(#limits)"), `link to #limits has no <a id="limits"> anchor`),
			problem("anchors", f.reportLine("[Not reported](#)"), `link to # has no <a id=""> anchor`),
		))
	})

	It("does not resolve a link to a heading's slug", func(ctx SpecContext) {
		replace(&f.report, "<a id=\"limitations\"></a>\n", "")
		Expect(f.check(ctx)).To(ConsistOf(
			problem("anchors", f.reportLine("(#limitations)"), "#limitations"),
		))
	})
})
