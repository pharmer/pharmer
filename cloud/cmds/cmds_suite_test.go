package cmds_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing with Ginkgo", func() {
	It("cmds", func() {

		RegisterFailHandler(Fail)
		RunSpecs(GinkgoT(), "Cmds Suite")
	})
})
