package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing/iotest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/cli"
)

const (
	vec31 = "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N"
	vec40 = "CVSS:4.0/AV:L/AC:L/AT:N/PR:L/UI:N/VC:H/VI:L/VA:N/SC:N/SI:N/SA:N"
	// vec40Python is a vector the Python cvss package scores 8.6.
	vec40Python = "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:P/VC:H/VI:L/VA:N/SC:H/SI:H/SA:H"
)

var _ = Describe("score", Label("TestScoreCommand"), func() {
	var stdout, stderr *bytes.Buffer

	BeforeEach(func() {
		stdout, stderr = new(bytes.Buffer), new(bytes.Buffer)
	})

	score := func(ctx context.Context, stdin io.Reader, args ...string) error {
		return cli.Run(ctx, append([]string{"score"}, args...), stdin, stdout, stderr)
	}
	lines := func(s ...string) io.Reader { return strings.NewReader(strings.Join(s, "\n")) }

	It("prints one line per input line, in order, and exits 0", func(ctx SpecContext) {
		err := score(ctx, lines(vec31, vec40, "-", vec40Python, ""))
		Expect(cli.ExitCode(err)).To(Equal(0), "error: %v", err)
		Expect(stdout.String()).To(Equal("7.1 High\n6.9 Medium\n-\n8.5 High\n"))
	})

	It("ignores whitespace around a line, a CRLF line end included", func(ctx SpecContext) {
		err := score(ctx, strings.NewReader("  "+vec31+"\t\r\n -\r\n"+vec40))
		Expect(cli.ExitCode(err)).To(Equal(0), "error: %v", err)
		Expect(stdout.String()).To(Equal("7.1 High\n-\n6.9 Medium\n"))
	})

	It("prints nothing for no input, and exits 0", func(ctx SpecContext) {
		Expect(cli.ExitCode(score(ctx, strings.NewReader("")))).To(Equal(0))
		Expect(stdout.String()).To(BeEmpty())
	})

	It("marks each line it cannot score ERR, scores the rest, and exits 1", func(ctx SpecContext) {
		vec30 := strings.Replace(vec31, "3.1", "3.0", 1)
		err := score(ctx, lines(vec31, "garbage", "", vec30, "CVSS:3.1/AV:X/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N", vec40, ""))
		Expect(cli.ExitCode(err)).To(Equal(1))
		Expect(err).To(MatchError(ContainSubstring("4 of 6 lines are ERR")))
		Expect(strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")).To(HaveExactElements(
			"7.1 High",
			`ERR "garbage" is not a CVSS 3.1 or 4.0 vector`,
			"ERR blank line",
			`ERR "`+vec30+`" is not a CVSS 3.1 or 4.0 vector`,
			HavePrefix(`ERR parsing CVSS 3.1 vector "CVSS:3.1/AV:X/`),
			"6.9 Medium",
		))
	})

	DescribeTable("exits 2 on a usage or I/O error, printing no score",
		func(ctx SpecContext, stdin io.Reader, args []string, message string) {
			err := score(ctx, stdin, args...)
			Expect(cli.ExitCode(err)).To(Equal(2))
			Expect(err).To(MatchError(ContainSubstring(message)))
			Expect(stdout.String()).To(BeEmpty())
		},
		Entry("a vector given as an argument", strings.NewReader(""), []string{vec31}, `unknown command "`+vec31+`"`),
		Entry("an unknown flag", strings.NewReader(""), []string{"--frob"}, "unknown flag: --frob"),
		Entry("standard input that fails", iotest.ErrReader(errors.New("stdin closed")), nil, "reading vectors: stdin closed"),
	)

	It("exits 0 on a request for help, describing its output", func(ctx SpecContext) {
		Expect(cli.ExitCode(score(ctx, strings.NewReader(""), "--help"))).To(Equal(0))
		Expect(stdout.String()).To(ContainSubstring("Usage:"))
		Expect(stdout.String()).To(ContainSubstring("ERR <reason>"))
	})
})
