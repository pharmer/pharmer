package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store/test/framework"
)

var _ = Describe("Node Group", func() {
	var (
		f   *framework.Invocation
		err error
	)
	BeforeEach(func() {
		f = root.Invoke()
		By("Receive storage " + f.Config.GetStoreType())
	})
	Describe("create node group", func() {
		var ng *api.NodeGroup
		BeforeEach(func() {
			ng, err = f.NG.GetSkeleton()
			Expect(err).NotTo(HaveOccurred())
		})
		By("using node group object ")
		It("should check", func() {
			By("should not find")
			_, err := f.Storage.NodeGroups(f.ClusterName).Get(ng.Name)
			Expect(err).To(HaveOccurred())

			By("should create")
			err = f.NG.Create(ng)
			Expect(err).NotTo(HaveOccurred())

			By("should find")
			_, err = f.Storage.NodeGroups(f.ClusterName).Get(ng.Name)
			Expect(err).NotTo(HaveOccurred())

			By("should update")
			err = f.NG.Update(ng)
			Expect(err).NotTo(HaveOccurred())

			By("should check update")
			err = f.NG.CheckUpdate(ng)
			Expect(err).NotTo(HaveOccurred())

			By("should update status")
			err = f.NG.UpdateStatus(ng)
			Expect(err).NotTo(HaveOccurred())

			By("should check status updated")
			err = f.NG.CheckUpdateStatus(ng)
			Expect(err).NotTo(HaveOccurred())

			By("should list")
			err = f.NG.List()
			Expect(err).NotTo(HaveOccurred())

			By("should delete")
			err = f.Storage.NodeGroups(f.ClusterName).Delete(ng.Name)
			Expect(err).NotTo(HaveOccurred())

			By("should not find")
			_, err = f.Storage.NodeGroups(f.ClusterName).Get(ng.Name)
			Expect(err).To(HaveOccurred())

		})

	})

})
