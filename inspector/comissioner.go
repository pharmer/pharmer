package inspector

import (
	"fmt"
	"github.com/appscode/go/errors"
	"k8s.io/client-go/kubernetes"

	"context"
	"github.com/appscode/go-term"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmd_api "k8s.io/client-go/tools/clientcmd/api"
	clientcmd_v1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

type Inspector struct {
	ctx     context.Context
	client  kubernetes.Interface
	cluster *api.Cluster
	config  *rest.Config
}

func New(ctx context.Context, cluster *api.Cluster) (*Inspector, error) {
	if cluster.Spec.Cloud.CloudProvider == "" {
		return nil, fmt.Errorf("cluster %v has no provider", cluster.Name)
	}
	var err error
	if ctx, err = LoadCACertificates(ctx, cluster); err != nil {
		return nil, err
	}
	if ctx, err = LoadSSHKey(ctx, cluster); err != nil {
		return nil, err
	}
	kc, err := NewAdminClient(ctx, cluster)
	if err != nil {
		return nil, err
	}

	adminConfig, err := GetAdminConfig(ctx, cluster)
	if err != nil {
		return nil, err
	}
	err = clientcmd_v1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}
	out := &clientcmd_api.Config{}
	err = scheme.Scheme.Convert(adminConfig, out, nil)
	if err != nil {
		return nil, err
	}
	clientConfig := clientcmd.NewDefaultClientConfig(*out, &clientcmd.ConfigOverrides{})
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return &Inspector{ctx: ctx, client: kc, cluster: cluster, config: restConfig}, nil
}

func (i *Inspector) NativeCheck() error {
	if err := i.CheckHelthStatus(); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := i.checkRBAC(); err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func (i *Inspector) NetworkCheck() error {
	if err := i.CheckDNSPod(); err != nil {
		return err
	}

	nodes, err := i.getNodes()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	var nodeonly = make([]string, 0)
	var masterNode apiv1.Node
	for _, n := range nodes.Items {
		if _, ok := n.ObjectMeta.Labels["node-role.kubernetes.io/master"]; !ok {
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

	var pods []apiv1.Pod
	if pods, err = i.InstallNginx(); err != nil {
		return err
	}

	term.Infoln("Checking Pod networks...")
	if err := i.runNodeExecutor(pods[0].Name, pods[1].Status.PodIP, defaultNamespace, pods[0].Spec.Containers[0].Name); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := i.runNodeExecutor(pods[1].Name, pods[0].Status.PodIP, defaultNamespace, pods[1].Spec.Containers[0].Name); err != nil {
		return errors.FromErr(err).Err()
	}

	term.Infoln("Checking from master")
	if err := i.runMasterExecutor(masterNode, pods[0].Status.PodIP); err != nil {
		return errors.FromErr(err).Err()
	}

	if err := i.runMasterExecutor(masterNode, pods[1].Status.PodIP); err != nil {
		return errors.FromErr(err).Err()
	}

	svcIP, err := i.InstallNginxService()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	term.Infoln("Checking networks usinng service ip...", svcIP)
	if err := i.runNodeExecutor(pods[0].Name, svcIP, defaultNamespace, pods[0].Spec.Containers[0].Name); err != nil {
		return errors.FromErr(err).Err()
	}
	term.Infoln("Checking networks using service name...")
	if err := i.runNodeExecutor(pods[1].Name, svcIP, defaultNamespace, pods[1].Spec.Containers[0].Name); err != nil {
		return errors.FromErr(err).Err()
	}

	if err := i.runMasterExecutor(masterNode, svcIP); err != nil {
		return errors.FromErr(err).Err()
	}

	return nil
}
