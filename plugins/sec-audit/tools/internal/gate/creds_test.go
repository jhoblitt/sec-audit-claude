package gate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/gate"
)

// Each synthetic key is assembled at run time, so that no credential-shaped
// literal sits in this file for a scanner, or the gate, to find.
var syntheticKeys = map[string]string{
	"aws-access-key": "AK" + "IA" + "ZZEXAMPLE" + "SYNTH12",
	"github-token":   "gh" + "p_" + strings.Repeat("Synth0", 6),
	"slack-token":    "xo" + "xb-" + "synthetic-" + "slacktoken",
	"private-key":    "-----BEGIN " + "EC PRIVATE" + " KEY-----",
	"google-api-key": "AI" + "za" + strings.Repeat("Synth", 7),
	"jwt":            "ey" + "JzeW50aGV0aWMiOnRydWV9" + "." + "ey" + "Jub3RyZWFsIjp0cnVlfQ" + "." + "c2ln",
	"keyring-aes":    "AQ" + strings.Repeat("Synthetic+", 3) + "Synthet1" + "==",
	"widget-key":     "wk" + "_" + "synthetic123",
}

// leaks returns each 6-byte window of key that appears in out.
func leaks(out, key string) []string {
	var found []string
	for i := 0; i+6 <= len(key); i++ {
		if strings.Contains(out, key[i:i+6]) {
			found = append(found, key[i:i+6])
		}
	}
	return found
}

var _ = Describe("the credentials check", func() {
	var f *fixture

	BeforeEach(func(ctx SpecContext) {
		f = newFixture(ctx, "clean")
		patterns := filepath.Join(f.dir, "patterns.tsv")
		Expect(os.WriteFile(patterns, []byte("# project patterns\n\nwidget-key\twk_[a-z0-9]{12}\n"), 0o600)).To(Succeed())
		var err error
		f.opts.Patterns, err = gate.LoadPatterns(patterns)
		Expect(err).NotTo(HaveOccurred())
	})

	// output renders problems as everything the gate can print about them.
	output := func(problems []gate.Problem) string {
		GinkgoHelper()
		data, err := json.Marshal(problems)
		Expect(err).NotTo(HaveOccurred())
		return string(data)
	}

	DescribeTable("names the pattern and position, never the match", Label("TestCredsNeverPrintsMatch"),
		func(ctx SpecContext, name string) {
			key := syntheticKeys[name]
			f.report += "\nThe fixture leaks `" + key + "` here.\n"
			problems := f.check(ctx)
			Expect(problems).To(ConsistOf(
				problem("credentials", f.reportLine("The fixture leaks"), "credential pattern "+name+" matches at column 20"),
			))
			Expect(leaks(output(problems), key)).To(BeEmpty())
		},
		Entry("an AWS access key", "aws-access-key"),
		Entry("a GitHub token", "github-token"),
		Entry("a Slack token", "slack-token"),
		Entry("a PEM private key", "private-key"),
		Entry("a Google API key", "google-api-key"),
		Entry("a JWT", "jwt"),
		Entry("a keyring secret", "keyring-aes"),
		Entry("a --patterns pattern", "widget-key"),
	)

	It("scans the findings table too, and redacts a credential another check would echo", Label("TestCredsNeverPrintsMatch"), func(ctx SpecContext) {
		key := syntheticKeys["aws-access-key"]
		replace(&f.table, "\tWidget cache keys collide across users\t", "\tWidget cache keys collide across users "+key+"\t")
		problems := f.check(ctx)
		Expect(problems).To(ContainElements(
			problem("credentials", f.opts.Table+":3", "credential pattern aws-access-key matches at column"),
			problem("consistency", f.reportLine("### F-02"), "title"),
			problem("consistency", f.opts.Report, "title"),
		))
		Expect(leaks(output(problems), key)).To(BeEmpty())
	})

	It("does not flag a credential's prefix written as prose", func(ctx SpecContext) {
		f.report += "\nGitHub classic tokens start with `ghp_` and AWS key IDs with `AKIA`.\n"
		Expect(f.check(ctx)).To(BeEmpty())
	})

	DescribeTable("rejects a malformed --patterns file, naming the line",
		func(body, message string) {
			path := filepath.Join(GinkgoT().TempDir(), "patterns.tsv")
			Expect(os.WriteFile(path, []byte(body), 0o600)).To(Succeed())
			_, err := gate.LoadPatterns(path)
			Expect(err).To(MatchError(ContainSubstring(message)))
		},
		Entry("a line with no tab", "ok\tx+\nno-tab-here\n", "patterns.tsv:2: want name<TAB>RE2"),
		Entry("an empty name", "\tx+\n", "patterns.tsv:1: want name<TAB>RE2"),
		Entry("an invalid expression", "bad\t(unclosed\n", "patterns.tsv:1: bad: "),
	)

	It("matches custom patterns as RE2", func() {
		path := filepath.Join(GinkgoT().TempDir(), "patterns.tsv")
		Expect(os.WriteFile(path, []byte("widget-key\twk_[a-z0-9]{12}\n"), 0o600)).To(Succeed())
		patterns, err := gate.LoadPatterns(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(patterns).To(HaveExactElements(And(
			HaveField("Name", "widget-key"),
			HaveField("RE", Equal(regexp.MustCompile(`wk_[a-z0-9]{12}`))),
		)))
	})
})
