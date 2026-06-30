package cli_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"app/internal/delivery/cli"
)

var _ = Describe("Root command", func() {
	It("rejects an unknown transport", func() {
		root := cli.NewRootCmd()
		root.SetArgs([]string{"serve", "-t", "bogus"})
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})

		err := root.Execute()
		Expect(err).To(MatchError(ContainSubstring("unknown transport")))
	})

	It("prints the version", func() {
		var out bytes.Buffer
		root := cli.NewRootCmd()
		root.SetArgs([]string{"version"})
		root.SetOut(&out)
		root.SetErr(&out)

		Expect(root.Execute()).To(Succeed())
		Expect(out.String()).To(ContainSubstring("casadellibro-mcp"))
	})
})
