package fake_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
)

var _ = Describe("Cluster", func() {
	It("test cluster", func() {
		opts := options.NewClusterCreateConfig()
		cluster := opts.Cluster
		cluster.Name = "test"
		ctx := cloud.NewContext(context.Background(), &v1beta1.PharmerConfig{}, "")

		By("Create cluster")
		c, err := cloud.Store(ctx).Clusters().Create(cluster)
		Expect(err).NotTo(HaveOccurred())
		Expect(c).Should(Equal(cluster))

		By("Get cluster")
		c, err = cloud.Store(ctx).Clusters().Get("test")
		Expect(err).NotTo(HaveOccurred())
		Expect(c).Should(Equal(cluster))

		By("Update cluster")
		cluster.Namespace = "testns"
		c, err = cloud.Store(ctx).Clusters().Update(cluster)
		Expect(err).NotTo(HaveOccurred())
		Expect(c).Should(Equal(cluster))

		c, err = cloud.Store(ctx).Clusters().Get("test")
		Expect(err).NotTo(HaveOccurred())
		Expect(c).Should(Equal(cluster))
	})
})
