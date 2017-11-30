package cloud

import (
	"context"
	"io/ioutil"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	clientcmd_api "k8s.io/client-go/tools/clientcmd/api"
	clientcmd_v1 "k8s.io/client-go/tools/clientcmd/api/v1"
	drain "k8s.io/kubernetes/pkg/kubectl/cmd"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type NodeDrain struct {
	o    drain.DrainOptions
	Node string
	ctx  context.Context
	kc   kubernetes.Interface
}

func NewNodeDrain(ctx context.Context, kc kubernetes.Interface, cluster *api.Cluster) (NodeDrain, error) {
	do := drain.DrainOptions{
		Force:              true,
		IgnoreDaemonsets:   true,
		DeleteLocalData:    true,
		GracePeriodSeconds: -1,
		Timeout:            0,
		Out:                ioutil.Discard,
		ErrOut:             ioutil.Discard,
	}

	c1, err := GetAdminConfig(ctx, cluster)
	if err != nil {
		return NodeDrain{}, err
	}
	err = clientcmd_v1.AddToScheme(scheme.Scheme)
	if err != nil {
		return NodeDrain{}, err
	}
	out := &clientcmd_api.Config{}
	err = scheme.Scheme.Convert(c1, out, nil)
	if err != nil {
		return NodeDrain{}, err
	}
	clientConfig := clientcmd.NewDefaultClientConfig(*out, &clientcmd.ConfigOverrides{})
	do.Factory = cmdutil.NewFactory(clientConfig)

	return NodeDrain{o: do, ctx: ctx, kc: kc}, nil
}

func (nd *NodeDrain) Apply() error {
	if err := nd.o.SetupDrain(&cobra.Command{}, []string{nd.Node}); err != nil {
		return err
	}
	if err := nd.o.RunDrain(); err != nil {
		return err
	}
	return nil
}

func (nd *NodeDrain) DeleteNode() error {
	if nd.kc != nil {
		return nd.kc.CoreV1().Nodes().Delete(nd.Node, &metav1.DeleteOptions{})
	}
	return nil
}
