package e2e

import (
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/store"
	"github.com/appscode/pharmer/store/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cluster", func() {
	var (
		f       *framework.Invocation
		storage store.Interface
	)
	BeforeEach(func() {
		f = root.Invoke()
		storage = f.Storage

		By("Receive storage " + f.Config.GetStoreType())
	})
	Describe("create cluster", func() {
		var (
			cluster *api.Cluster
			err     error
		)
		BeforeEach(func() {
			cluster, err = f.Cluster.GetSkeleton()
			Expect(err).NotTo(HaveOccurred())
		})
		By("using cluster object ")
		It("should create", func() {
			_, err := f.Storage.Clusters().Create(cluster)
			Expect(err).NotTo(HaveOccurred())

		})

		It("should not create", func() {
			_, err := f.Storage.Clusters().Create(cluster)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("retrieve cluster", func() {
		var name string
		BeforeEach(func() {
			name = f.Cluster.GetName()
		})
		By("checking with existing cluster name")
		It("should find", func() {
			_, err := f.Storage.Clusters().Get(name)
			Expect(err).NotTo(HaveOccurred())
		})

		By("checking without existing cluster name")
		It("should not find", func() {
			_, err := f.Storage.Credentials().Get("noc")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("update cluster", func() {
		var (
			cluster *api.Cluster
			err     error
		)
		BeforeEach(func() {
			name := f.Cluster.GetName()
			cluster, err = f.Storage.Clusters().Get(name)
			Expect(err).NotTo(HaveOccurred())
		})
		By("using existing cluster")
		It("should update", func() {
			err := f.Cluster.Update(cluster)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("update cluster status", func() {
		var (
			cluster *api.Cluster
			err     error
		)
		BeforeEach(func() {
			name := f.Cluster.GetName()
			cluster, err = f.Storage.Clusters().Get(name)
			Expect(err).NotTo(HaveOccurred())
		})
		By("using existing cluster")
		It("should update status", func() {
			err := f.Cluster.UpdateStatus(cluster)
			Expect(err).NotTo(HaveOccurred())
		})
	})

})
