package packet

import (
	"context"
	"fmt"
	"testing"

	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/credential"
	"github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func getCluster() *v1beta1.Cluster {
	return &v1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: v1beta1.PharmerClusterSpec{
			ClusterAPI: &clusterapi.Cluster{},
			Config: &v1beta1.ClusterConfig{
				Cloud: v1beta1.CloudSpec{
					NetworkProvider: v1beta1.PodNetworkCalico,
					CloudProvider:   "linode",
					Zone:            "us-central-1f",
				},
				CredentialName:    "test",
				KubernetesVersion: "v1.14.0",
			},
		},
	}
}

func getCredential() *cloudapi.Credential {
	return &cloudapi.Credential{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: cloudapi.CredentialSpec{
			Provider: "azure",
			Data: map[string]string{
				credential.PacketProjectID: "a",
				credential.PacketAPIKey:    "a",
			},
		},
	}
}

func getMachine() *clusterapi.Machine {
	return &clusterapi.Machine{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: clusterapi.MachineSpec{
			ObjectMeta:   metav1.ObjectMeta{},
			Taints:       nil,
			ProviderSpec: clusterapi.ProviderSpec{},
			Versions: clusterapi.MachineVersionInfo{
				Kubelet: "v1.14.0",
			},
			ConfigSource: nil,
			ProviderID:   nil,
		},
		Status: clusterapi.MachineStatus{},
	}
}

func Test_cloudConnector_renderStartupScript(t *testing.T) {
	ctx := cloud.NewContext(context.Background(), &v1beta1.PharmerConfig{}, "")

	cluster := getCluster()
	credential := getCredential()

	_, err := cloud.Store(ctx).Credentials().Create(credential)
	if err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	_, err = cloud.Create(ctx, cluster, "")
	if err != nil {
		t.Fatalf("failed to create cluster :%v", err)
	}

	cmInterface, err := cloud.GetCloudManager("linode", ctx)
	if err != nil {
		t.Fatalf("failed to get cloud manager :%v", err)
	}

	cm, ok := cmInterface.(*ClusterManager)
	if !ok {
		t.Fatalf("failed to get cluster manager: %v", err)
	}

	cm.cluster = cluster
	cm.namer = namer{cluster: cluster}

	cm.conn, err = PrepareCloud(cm.ctx, cm.cluster.Name, cm.owner)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}

	machine := getMachine()

	script, err := cm.conn.renderStartupScript(cm.conn.cluster, machine, "token", cm.owner)
	if err != nil {
		t.Fatalf("failed to generate startupscript: %v", err)
	}

	fmt.Println(script)
}
