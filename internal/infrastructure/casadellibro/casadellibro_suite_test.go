package casadellibro_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCasadellibro(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Casadellibro Adapter Suite")
}
