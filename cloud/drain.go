package cloud

import (
	"context"
	"io/ioutil"

	api "github.com/pharmer/pharmer/apis/v1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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
	out := api.Convert_KubeConfig_To_Config(c1)
	clientConfig := clientcmd.NewDefaultClientConfig(*out, &clientcmd.ConfigOverrides{})
	do.Factory = cmdutil.NewFactory(clientConfig)

	return NodeDrain{o: do, ctx: ctx, kc: kc}, nil
}

func (nd *NodeDrain) Apply() error {
	cmd := &cobra.Command{}
	// https://github.com/kubernetes/kubernetes/blob/7377c5911a5e2d5f18dfb15617316a891e661f22/pkg/kubectl/cmd/drain.go#L215
	cmd.Flags().String("selector", nd.o.Selector, "Selector (label query) to filter on")
	// https://github.com/kubernetes/kubernetes/blob/7377c5911a5e2d5f18dfb15617316a891e661f22/pkg/kubectl/cmd/drain.go#L227
	cmdutil.AddDryRunFlag(cmd)
	if err := nd.o.SetupDrain(cmd, []string{nd.Node}); err != nil {
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
