package e2e

import (
	"strings"
	"testing"

	. "github.com/pharmer/pharmer/test/e2e/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

var _ = BeforeSuite(func() {
	BuildPharmer()
})

var _ = AfterSuite(func() {
	DeleteCredential()
	gexec.CleanupBuildArtifacts()
})

var _ = Describe("E2E Tests", func() {
	for _, provider := range strings.Split(Providers, ",") {
		Describe("full e2e tests for provider "+provider, func() {
			Context("for kubernetes version"+CurrentVersion, func() {
				fullE2ETest(provider, CurrentVersion)
			})
			Context("for kubernetes version"+UpdateToVersion, func() {
				fullE2ETest(provider, UpdateToVersion)
			})
		})
	}
})

var fullE2ETest = func(provider, version string) {
	It("should setup cluster-name and create credentials", func() {
		SetClusterName()
		CreateCredential()
	})

	It("should create a cluster", func() {
		CreateCluster(provider, version)
	})

	It("should apply the cluster", func() {
		ApplyCluster()
	})

	It("should get kubeconfig for the cluster", func() {
		UseCluster()
	})

	Context("wait for nodes to be ready", func() {
		It("should wait for masters to be ready", func() {
			WaitForNodeReady("master", Masters)
		})

		It("should wait for nodes to be ready", func() {
			WaitForNodeReady("node", 1)
		})
	})

	Context("scale the cluster", func() {
		It("it should scale the cluster to two nodes", func() {
			ScaleCluster(2)
		})

		It("should wait for two nodes to be ready", func() {
			WaitForNodeReady("node", 2)
		})
	})

	//It("upgrades cluster", func() {
	//	UpgradeCluster()
	//})
	//
	//It("applies the changes", func() {
	//	ApplyCluster()
	//})
	//
	//It("waits for clusters to be updated", func() {
	//	WaitForUpdates()
	//})

	if !SkipDeleteCluster {
		It("should delete cluster", func() {
			DeleteCluster()
			ApplyCluster()
		})
	}
}
