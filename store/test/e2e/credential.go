package e2e

import (
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/store"
	"github.com/appscode/pharmer/store/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Credential", func() {
	var (
		f       *framework.Invocation
		storage store.Interface
	)
	BeforeEach(func() {
		f = root.Invoke()
		storage = f.Storage

		By("Receive storage " + f.Config.GetStoreType())
	})
	Describe("create credential", func() {
		var cred *api.Credential
		BeforeEach(func() {
			cred = f.Credential.GetSkeleton()
		})
		By("using credential object ")
		It("should check", func() {
			By("should not find" + cred.Name)
			_, err := f.Storage.Credentials().Get(cred.Name)
			Expect(err).To(HaveOccurred())

			By("should create")
			_, err = f.Storage.Credentials().Create(cred)
			Expect(err).NotTo(HaveOccurred())

			By("should find")
			_, err = f.Storage.Credentials().Get(cred.Name)
			Expect(err).NotTo(HaveOccurred())

			By("should not create")
			_, err = f.Storage.Credentials().Create(cred)
			Expect(err).To(HaveOccurred())

			By("should update")
			err = f.Credential.Update(cred)
			Expect(err).NotTo(HaveOccurred())

			By("should check update")
			err = f.Credential.CheckUpdate(cred)
			Expect(err).NotTo(HaveOccurred())

			By("should list")
			err = f.Credential.List()
			Expect(err).NotTo(HaveOccurred())

			By("delete")
			err = f.Storage.Credentials().Delete(cred.Name)
			Expect(err).NotTo(HaveOccurred())

			By("should not find")
			_, err = f.Storage.Credentials().Get(cred.Name)
			Expect(err).To(HaveOccurred())

		})
	})
})
