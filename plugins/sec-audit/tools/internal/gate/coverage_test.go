package gate_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jhoblitt/sec-audit-claude/plugins/sec-audit/tools/internal/gate"
)

var _ = Describe("Coverage", func() {
	var dir string

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
	})

	write := func(name, body string) string {
		GinkgoHelper()
		path := filepath.Join(dir, name)
		Expect(os.WriteFile(path, []byte(body), 0o600)).To(Succeed())
		return path
	}

	It("finds no problem in the testdata scope", func() {
		problems, err := gate.Coverage([]gate.Scope{{
			Name: "S01", List: "testdata/scopes/S01.txt", Manifest: "testdata/scopes/S01.notes.md",
		}})
		Expect(err).NotTo(HaveOccurred())
		Expect(problems).To(BeEmpty())
	})

	It("reports listed files with no READ line, and READ files not listed", Label("TestCoverageMissingAndExtra"), func() {
		list := write("S02.txt", "src/a.py\nsrc/b.py\n\nsrc/c.py\n")
		manifest := write("S02.notes.md", "# S02 notes\n\n"+
			"READ src/a.py 10 — clean\n"+
			"- READ src/d.py 4 — clean\n"+
			"  READ src/a.py 10 — read twice\n"+
			"CLEAN src/b.py — no READ line, so not read\n"+
			"Prose that mentions READ src/c.py is not a READ line.\n")
		other := write("S03.txt", "src/e.py\n")
		otherManifest := write("S03.notes.md", "READ src/e.py 3 — clean\n")

		problems, err := gate.Coverage([]gate.Scope{
			{Name: "S02", List: list, Manifest: manifest},
			{Name: "S03", List: other, Manifest: otherManifest},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(problems).To(Equal([]gate.Problem{
			{Check: "coverage", Where: "S02", Message: "missing: src/b.py is listed in " + list + " but has no READ line in " + manifest},
			{Check: "coverage", Where: "S02", Message: "missing: src/c.py is listed in " + list + " but has no READ line in " + manifest},
			{Check: "coverage", Where: "S02", Message: "extra: src/d.py has a READ line in " + manifest + " but is not listed in " + list},
		}))
	})

	It("reports a scope file it cannot read as an error", func() {
		list := write("S04.txt", "src/a.py\n")
		_, err := gate.Coverage([]gate.Scope{{Name: "S04", List: list, Manifest: filepath.Join(dir, "absent.md")}})
		Expect(err).To(MatchError(os.ErrNotExist))
	})
})
