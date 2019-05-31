package cloud

import (
	"testing"
	//"github.com/kubernetes-incubator/apiserver-builder/pkg/controller"
)

//func TestCreateApiserver(t *testing.T) {
//
//	cfgFile := filepath.Join(homedir.HomeDir(), ".pharmer", "config.d", "default")
//	cfg, err := config.LoadConfig(cfgFile)
//	if err != nil {
//		fmt.Println(err)
//	}
//	ctx := NewContext(context.Background(), cfg, _env.Dev)
//
//	Cluster, err := store.StoreProvider.Clusters().Get("doc6")
//	fmt.Println(err)
//
//	if ctx, err = LoadCACertificates(ctx, Cluster); err != nil {
//		fmt.Println(err, "----")
//	}
//	if ctx, err = LoadSSHKey(ctx, Cluster); err != nil {
//		fmt.Println(err)
//	}
//	/*if cm.conn, err = NewConnector(cm.ctx, cm.Cluster); err != nil {
//		return err
//	}*/
//
//	fmt.Println(Cluster.Spec.Masters[0].ClusterName)
//	os.Exit(1)
//
//	kubeClient, err := NewAdminClient(ctx, Cluster)
//	if err != nil {
//		fmt.Println(err)
//	}
//
//	ca, err := NewClusterApi(ctx, Cluster, kubeClient)
//	if err != nil {
//		fmt.Println(err)
//	}
//	c, err := ca.client.Clusters(core.NamespaceDefault).Create(ca.Cluster.Spec.ClusterAPI)
//	fmt.Println(c)
//	if err != nil {
//		fmt.Println(err)
//	}
//
//	/*if err := ca.Apply(); err != nil {
//		fmt.Println(err)
//	}*/
//
//}

func TestExists(t *testing.T) {
	/*machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"Cluster.pharmer.io/Cluster": "doco9",
				"Cluster.pharmer.io/mg":      "2gb",
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
