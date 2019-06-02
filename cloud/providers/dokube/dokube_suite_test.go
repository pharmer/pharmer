package dokube_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDokube(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dokube Suite")
}
