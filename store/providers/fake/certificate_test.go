package fake_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
)

var _ = Describe("Certificate", func() {
	It("test test certificates", func() {
		opts := options.NewClusterCreateConfig()
		cluster := opts.Cluster
		cluster.Name = "test"
		ctx := cloud.NewContext(context.Background(), &v1beta1.PharmerConfig{}, "")
		var err error

		By("Create cluster")
		c, err := cloud.Store(ctx).Clusters().Create(cluster)
		Expect(err).NotTo(HaveOccurred())
		Expect(c).Should(Equal(cluster))

		By("Create credential")
		ctx, err = cloud.CreateCACertificates(ctx, cluster, "")
		Expect(err).NotTo(HaveOccurred())
	})
})
