package e2e

import (
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/store"
	"github.com/appscode/pharmer/store/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Node Group", func() {
	var (
		f       *framework.Invocation
		storage store.Interface
		err     error
	)
	BeforeEach(func() {
		f = root.Invoke()
		storage = f.Storage
		By("Receive storage " + f.Config.GetStoreType())
	})
	Describe("create node group", func() {
		var ng *api.NodeGroup
		BeforeEach(func() {
			ng, err = f.NG.GetSkeleton()
			Expect(err).NotTo(HaveOccurred())
		})
		By("using node group object ")
		It("should create", func() {
			err := f.NG.Create(ng)
			Expect(err).NotTo(HaveOccurred())

		})

		It("should not create", func() {
			err := f.NG.Create(ng)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("retrieve node group", func() {
		var name string
		BeforeEach(func() {
			name = f.NG.GetName()
		})
		By("checking with existing node group name")
		It("should find", func() {
			_, err := f.Storage.NodeGroups(f.ClusterName).Get(name)
			Expect(err).NotTo(HaveOccurred())
		})

		By("checking without existing node group name")
		It("should not find", func() {
			_, err := f.Storage.NodeGroups(f.ClusterName).Get("nog")
			Expect(err).To(HaveOccurred())
		})

		By("checking without existing cluster name")
		It("should not find", func() {
			_, err := f.Storage.NodeGroups("noc").Get(name)
			Expect(err).To(HaveOccurred())
		})

		By("checking for all node group list")
		It("should find", func() {
			err = f.NG.List()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("update node group", func() {
		var (
			ng   *api.NodeGroup
			name string
		)
		BeforeEach(func() {
			name = f.NG.GetName()
			ng, err = f.Storage.NodeGroups(f.ClusterName).Get(name)
			Expect(err).NotTo(HaveOccurred())
		})
		By("checking with existing node group")
		It("should update", func() {
			err = f.NG.Update(ng)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update status", func() {
			err = f.NG.UpdateStatus(ng)
			Expect(err).NotTo(HaveOccurred())
		})
	})

})
