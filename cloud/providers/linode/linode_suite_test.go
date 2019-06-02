package linode_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLinode(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Linode Suite")
}
