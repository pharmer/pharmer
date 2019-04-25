package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store/test/framework"
)

var _ = Describe("Cluster", func() {
	var (
		f *framework.Invocation
	)
	BeforeEach(func() {
		f = root.Invoke()

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
		It("should check", func() {
			By("should not find" + cluster.Name)
			_, err := f.Storage.Credentials().Get(cluster.Name)
			Expect(err).To(HaveOccurred())

			By("should create")
			_, err = f.Storage.Clusters().Create(cluster)
			Expect(err).NotTo(HaveOccurred())

			By("should find")
			_, err = f.Storage.Clusters().Get(cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			By("should not create")
			_, err = f.Storage.Clusters().Create(cluster)
			Expect(err).To(HaveOccurred())

			By("should update")
			err = f.Cluster.Update(cluster)
			Expect(err).NotTo(HaveOccurred())

			By("should check updated")
			err = f.Cluster.CheckUpdate(cluster)
			Expect(err).NotTo(HaveOccurred())

			By("should update status")
			err = f.Cluster.UpdateStatus(cluster)
			Expect(err).NotTo(HaveOccurred())

			By("should check status updated")
			err = f.Cluster.CheckUpdateStatus(cluster)
			Expect(err).NotTo(HaveOccurred())

			By("should list")
			err = f.Cluster.List()
			Expect(err).NotTo(HaveOccurred())

			By("should delete")
			err = f.Storage.Clusters().Delete(cluster.Name)
			Expect(err).NotTo(HaveOccurred())

			By("should not find")
			_, err = f.Storage.Credentials().Get(cluster.Name)
			Expect(err).To(HaveOccurred())

		})

	})

})
