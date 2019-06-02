package digitalocean_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDigitalocean(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Digitalocean Suite")
}
