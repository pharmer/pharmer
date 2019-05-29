package gce

import (
	"testing"
)

func Test_cloudConnector_renderStartupScript(t *testing.T) {

	//ctx := cloud.NewContext(context.Background(), &v1beta1.PharmerConfig{}, "")
	//
	//cluster := getCluster()
	//credential := getCredential()
	//
	//_, err := store.StoreProvider.Credentials().Create(credential)
	//if err != nil {
	//	t.Fatalf("failed to create credential: %v", err)
	//}
	//
	//_, _, err = cloud.Create(cluster)
	//if err != nil {
	//	t.Fatalf("failed to create cluster :%v", err)
	//}
	//
	//cmInterface, err := cloud.GetCloudManager("gce", ctx)
	//if err != nil {
	//	t.Fatalf("failed to get cloud manager :%v", err)
	//}
	//
	//cm, ok := cmInterface.(*ClusterManager)
	//if !ok {
	//	t.Fatalf("failed to get cluster manager: %v", err)
	//}
	//
	//cm.cluster = cluster
	//cm.namer = namer{cluster: cluster}
	//
	//// error: failed to create cluster :Credential test does not have necessary authorization. Reason: Credential missing required authorization.
	//// TODO: add mock gce client
	////err = GetCloudConnector(cm)
	////if err != nil {
	////	t.Fatalf("failed to create cluster :%v", err)
	////}
	////
	////machine := getMachine()
	////
	////script, err := cm.conn.renderStartupScript(cm.conn.cluster, machine, "token")
	////if err != nil {
	////	t.Fatalf("failed to generate startupscript: %v", err)
	////}
	////
	////fmt.Println(script)
}
