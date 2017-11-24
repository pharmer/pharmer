package e2e

import (
	"github.com/appscode/go/crypto/ssh"
	"github.com/appscode/pharmer/store/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SSH", func() {
	var (
		f   *framework.Invocation
		err error
	)
	BeforeEach(func() {
		f = root.Invoke()
		By("Receive storage " + f.Config.GetStoreType())
	})
	Describe("create ssh", func() {
		var (
			ssh  *ssh.SSHKey
			name string
		)
		BeforeEach(func() {
			ssh, err = f.SSH.GetSkeleton()
			Expect(err).NotTo(HaveOccurred())
			name = f.SSH.GetName()
		})
		By("using ssh object ")
		It("should check ", func() {
			By("should not find")
			_, _, err := f.Storage.SSHKeys(f.ClusterName).Get(name)
			Expect(err).To(HaveOccurred())

			By("should create")
			err = f.SSH.Create(ssh)
			Expect(err).NotTo(HaveOccurred())

			By("should find")
			_, _, err = f.Storage.SSHKeys(f.ClusterName).Get(name)
			Expect(err).NotTo(HaveOccurred())

			By("should delete")
			err = f.Storage.SSHKeys(f.ClusterName).Delete(name)
			Expect(err).NotTo(HaveOccurred())

			By("should not find")
			_, _, err = f.Storage.SSHKeys(f.ClusterName).Get(name)
			Expect(err).To(HaveOccurred())

		})

	})

})
