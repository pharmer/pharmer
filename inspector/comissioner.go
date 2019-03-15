package inspector

import (
	"context"

	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Inspector struct {
	ctx     context.Context
	client  kubernetes.Interface
	cluster *api.Cluster
	config  *rest.Config

	owner string
}

func New(ctx context.Context, cluster *api.Cluster, owner string) (*Inspector, error) {
	if cluster.ClusterConfig().Cloud.CloudProvider == "" {
		return nil, errors.Errorf("cluster %v has no provider", cluster.Name)
	}
	var err error
	if ctx, err = LoadCACertificates(ctx, cluster, owner); err != nil {
		return nil, err
	}
	if ctx, err = LoadSSHKey(ctx, cluster, owner); err != nil {
		return nil, err
	}
	kc, err := NewAdminClient(ctx, cluster)
	if err != nil {
		return nil, err
	}

	adminConfig, err := GetAdminConfig(ctx, cluster, owner)
	if err != nil {
		return nil, err
	}
	out := api.Convert_KubeConfig_To_Config(adminConfig)
	clientConfig := clientcmd.NewDefaultClientConfig(*out, &clientcmd.ConfigOverrides{})
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return &Inspector{ctx: ctx, client: kc, cluster: cluster, config: restConfig, owner: owner}, nil
}

func (i *Inspector) NativeCheck() error {
	if err := i.CheckHelthStatus(); err != nil {
		return errors.WithStack(err)
	}
	if err := i.checkRBAC(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (i *Inspector) NetworkCheck() error {
	if err := i.CheckDNSPod(); err != nil {
		return err
	}

	nodes, err := i.getNodes()
	if err != nil {
		return errors.WithStack(err)
	}

	var nodeonly = make([]string, 0)
	var masterNode core.Node
	for _, n := range nodes.Items {
		if _, ok := n.ObjectMeta.Labels[api.RoleMasterKey]; !ok {
			nodeonly = append(nodeonly, n.Name)
		} else {
			masterNode = n
		}
	}
	if len(nodeonly) <= 1 {
		term.Fatalln("Need at least 2 nodes to check")
		return nil
	}

	defer func() {
		i.DeleteNginxService()
		i.DeleteNginx()
	}()

	var pods []core.Pod
	if pods, err = i.InstallNginx(); err != nil {
		return err
	}

	term.Infoln("Checking Pod networks...")
	if err := i.runNodeExecutor(pods[0].Name, pods[1].Status.PodIP, defaultNamespace, pods[0].Spec.Containers[0].Name); err != nil {
		return errors.WithStack(err)
	}
	if err := i.runNodeExecutor(pods[1].Name, pods[0].Status.PodIP, defaultNamespace, pods[1].Spec.Containers[0].Name); err != nil {
		return errors.WithStack(err)
	}

	term.Infoln("Checking from master")
	if err := i.runMasterExecutor(masterNode, pods[0].Status.PodIP); err != nil {
		return errors.WithStack(err)
	}

	if err := i.runMasterExecutor(masterNode, pods[1].Status.PodIP); err != nil {
		return errors.WithStack(err)
	}

	svcIP, err := i.InstallNginxService()
	if err != nil {
		return errors.WithStack(err)
	}

	term.Infoln("Checking networks usinng service ip...", svcIP)
	if err := i.runNodeExecutor(pods[0].Name, svcIP, defaultNamespace, pods[0].Spec.Containers[0].Name); err != nil {
		return errors.WithStack(err)
	}
	term.Infoln("Checking networks using service name...")
	if err := i.runNodeExecutor(pods[1].Name, svcIP, defaultNamespace, pods[1].Spec.Containers[0].Name); err != nil {
		return errors.WithStack(err)
	}

	if err := i.runMasterExecutor(masterNode, svcIP); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
