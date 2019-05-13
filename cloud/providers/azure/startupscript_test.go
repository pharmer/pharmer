package azure

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
)

func Test_cloudConnector_renderStartupScript(t *testing.T) {
	RegisterFailHandler(Fail)

	ctx := cloud.NewContext(context.Background(), &v1beta1.PharmerConfig{}, "")

	cluster := getCluster()
	credential := getCredential()

	_, err := cloud.Store(ctx).Credentials().Create(credential)
	Expect(err).NotTo(HaveOccurred())

	_, err = cloud.Create(ctx, cluster, "")
	Expect(err).NotTo(HaveOccurred())

	cmInterface, err := cloud.GetCloudManager("azure", ctx)
	Expect(err).NotTo(HaveOccurred())

	cm, ok := cmInterface.(*ClusterManager)
	Expect(ok).Should(Equal(true))

	cm.cluster = cluster
	cm.namer = namer{cluster: cluster}

	err = PrepareCloud(cm)
	if err != nil {
		panic(err)
	}

	machine := getMachine()

	script, err := cm.conn.renderStartupScript(cm.conn.cluster, machine, cm.conn.owner, "token")
	Expect(err).NotTo(HaveOccurred())

	fmt.Println(script)
}
