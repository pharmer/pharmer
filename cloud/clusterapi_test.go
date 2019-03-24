package cloud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_env "github.com/appscode/go/env"

	//"github.com/kubernetes-incubator/apiserver-builder/pkg/controller"
	"github.com/pharmer/pharmer/config"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/util/homedir"
)

func TestCreateApiserver(t *testing.T) {

	cfgFile := filepath.Join(homedir.HomeDir(), ".pharmer", "config.d", "default")
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Println(err)
	}
	ctx := NewContext(context.Background(), cfg, _env.Dev)

	cluster, err := Store(ctx).Owner(owner).Clusters().Get("doc6")
	fmt.Println(err)

	if ctx, err = LoadCACertificates(ctx, cluster); err != nil {
		fmt.Println(err, "----")
	}
	if ctx, err = LoadSSHKey(ctx, cluster); err != nil {
		fmt.Println(err)
	}
	/*if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return err
	}*/

	fmt.Println(cluster.Spec.Masters[0].ClusterName)
	os.Exit(1)

	kc, err := NewAdminClient(ctx, cluster)
	if err != nil {
		fmt.Println(err)
	}

	ca, err := NewClusterApi(ctx, cluster, kc)
	if err != nil {
		fmt.Println(err)
	}
	c, err := ca.client.Clusters(core.NamespaceDefault).Create(ca.cluster.Spec.ClusterAPI)
	fmt.Println(c)
	if err != nil {
		fmt.Println(err)
	}

	/*if err := ca.Apply(); err != nil {
		fmt.Println(err)
	}*/

}

func TestExists(t *testing.T) {
	/*machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"cluster.pharmer.io/cluster": "doco9",
				"cluster.pharmer.io/mg":      "2gb",
			},
			Name: "2gb-pool-phg2q",
		},
		Spec: clusterv1.MachineSpec{},
	}

	conf, err := controller.GetConfig("/home/sanjid/.kube/config")
	if err != nil {
		fmt.Println(err)
	}
	client, err := clientset.NewForConfig(conf)
	fmt.Println(client, err)

	cfgFile := filepath.Join(homedir.HomeDir(), ".pharmer", "config.d", "default")
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Println(err)
	}
	ctx := NewContext(context.Background(), cfg, _env.Dev)

	cm, err := GetCloudManager("digitalocean", ctx)
	fmt.Println(cm, err)
	err = cm.InitializeActuator(client.ClusterV1alpha1().Machines(core.NamespaceDefault))
	fmt.Println(err)
	fmt.Println(cm.Exists(machine))*/

}
