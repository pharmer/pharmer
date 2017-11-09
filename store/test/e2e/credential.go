package e2e

import (
	"fmt"

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
		By("using credential ")
		It("should create", func() {
			_, err := f.Storage.Credentials().Create(cred)
			Expect(err).NotTo(HaveOccurred())

		})

		It("should not create", func() {
			_, err := f.Storage.Credentials().Create(cred)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("retrieve credential", func() {
		var name string
		BeforeEach(func() {
			name = f.Credential.GetSkeleton().Name
		})
		By("checking with existing credential name")
		It("should find", func() {
			_, err := f.Storage.Credentials().Get(name)
			Expect(err).NotTo(HaveOccurred())
		})

		By("checking without existing credential name")
		It("should not find", func() {
			_, err := f.Storage.Credentials().Get("nof")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("update credential", func() {
		var (
			cred *api.Credential
			err  error
		)
		BeforeEach(func() {
			name := f.Credential.GetSkeleton().Name
			cred, err = f.Storage.Credentials().Get(name)
			Expect(err).NotTo(HaveOccurred())
		})
		By("using existing credential")
		It("should update", func() {
			err := f.Credential.Update(cred)
			Expect(err).NotTo(HaveOccurred())
		})

	})

})
