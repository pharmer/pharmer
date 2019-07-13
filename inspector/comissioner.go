package inspector

import (
	"github.com/appscode/go/term"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/cloud/utils/certificates"
	"pharmer.dev/pharmer/cloud/utils/kube"
	"pharmer.dev/pharmer/store"
)

type Inspector struct {
	storeProvider store.ResourceInterface
	client        kubernetes.Interface
	cluster       *api.Cluster
	config        *rest.Config
}

func New(storeProvider store.ResourceInterface, cluster *api.Cluster, caCert *certificates.CertKeyPair) (*Inspector, error) {
	restConfig, err := kube.NewRestConfig(caCert, cluster)
	if err != nil {
		return nil, err
	}
	kubeclient, err := kube.NewAdminClient(caCert, cluster)
	if err != nil {
		return nil, err
	}

	return &Inspector{
		storeProvider: storeProvider,
		client:        kubeclient,
		cluster:       cluster,
		config:        restConfig,
	}, nil
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
		_ = i.DeleteNginxService()
		_ = i.DeleteNginx()
	}()

	var pods []core.Pod
	if pods, err = i.InstallNginx(); err != nil {
		return err
	}

	term.Infoln("Checking Pod networks...")
	if err := i.runNodeExecutor(pods[0].Name, pods[1].Status.PodIP, pods[0].Spec.Containers[0].Name); err != nil {
		return errors.WithStack(err)
	}
	if err := i.runNodeExecutor(pods[1].Name, pods[0].Status.PodIP, pods[1].Spec.Containers[0].Name); err != nil {
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
	if err := i.runNodeExecutor(pods[0].Name, svcIP, pods[0].Spec.Containers[0].Name); err != nil {
		return errors.WithStack(err)
	}
	term.Infoln("Checking networks using service name...")
	if err := i.runNodeExecutor(pods[1].Name, svcIP, pods[1].Spec.Containers[0].Name); err != nil {
		return errors.WithStack(err)
	}

	if err := i.runMasterExecutor(masterNode, svcIP); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
