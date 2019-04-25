package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/pharmer/store/test/framework"
)

var _ = Describe("Credential", func() {
	var (
		f *framework.Invocation
	)
	BeforeEach(func() {
		f = root.Invoke()
		By("Receive storage " + f.Config.GetStoreType())
	})
	Describe("create credential", func() {
		var cred *cloudapi.Credential
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
