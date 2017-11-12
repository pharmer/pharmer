package e2e

import (
	"github.com/appscode/go/crypto/ssh"
	"github.com/appscode/pharmer/store"
	"github.com/appscode/pharmer/store/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SSH", func() {
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
	Describe("create ssh", func() {
		var ssh *ssh.SSHKey
		BeforeEach(func() {
			ssh, err = f.SSH.GetSkeleton()
			Expect(err).NotTo(HaveOccurred())
		})
		By("using ssh object ")
		It("should create", func() {
			err := f.SSH.Create(ssh)
			Expect(err).NotTo(HaveOccurred())

		})

		It("should not create", func() {
			err := f.SSH.Create(ssh)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("retrieve key", func() {
		var name string
		BeforeEach(func() {
			name = f.SSH.GetName()
		})
		By("checking with existing ssh key name")
		It("should find", func() {
			_, _, err := f.Storage.SSHKeys(f.ClusterName).Get(name)
			Expect(err).NotTo(HaveOccurred())
		})

		By("checking without existing ssh key name")
		It("should not find", func() {
			_, _, err := f.Storage.SSHKeys(f.ClusterName).Get("nos")
			Expect(err).To(HaveOccurred())
		})

		By("checking without existing cluster name")
		It("should not find", func() {
			_, _, err := f.Storage.SSHKeys("noc").Get(name)
			Expect(err).To(HaveOccurred())
		})
	})

})
