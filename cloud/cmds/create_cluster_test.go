package cmds_test

import (
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pharmer/pharmer/test/e2e/util"
)

var _ = Describe("Testing with Ginkgo", func() {
})

var _ = BeforeSuite(func() {
	util.BuildPharmer()
})

var _ = AfterSuite(func() {
	util.DeleteCredential()
	gexec.CleanupBuildArtifacts()
})

var _ = Describe("CreateCluster", func() {
	It("should create a cluster", func() {
		util.SetClusterName()
		err := util.CreateCluster("gce", "1.14.0")
		Expect(err).NotTo(HaveOccurred())
	})
})
