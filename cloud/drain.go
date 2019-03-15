package cloud

import (
	"context"
	"io/ioutil"
	"os"
	"time"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubernetes/pkg/kubectl/cmd/drain"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util" //"k8s.io/kubernetes/pkg/kubectl/scheme"
)

type NodeDrain struct {
	o    *drain.DrainOptions
	Node string
	ctx  context.Context
	kc   kubernetes.Interface
	f    cmdutil.Factory
}

type restClientGetter struct {
	config *clientcmdapi.Config
}

func (r *restClientGetter) ToRESTConfig() (*rest.Config, error) {
	return r.ToRawKubeConfigLoader().ClientConfig()
}

func (r *restClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := r.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return discovery.NewCachedDiscoveryClientForConfig(config, os.TempDir(), "", 10*time.Minute)
}

func (r *restClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	client, err := r.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(client), nil
}

func (r *restClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return clientcmd.NewDefaultClientConfig(*r.config, &clientcmd.ConfigOverrides{})
}

func NewNodeDrain(ctx context.Context, kc kubernetes.Interface, cluster *api.Cluster, owner string) (NodeDrain, error) {
	do := drain.NewDrainOptions(nil, genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    ioutil.Discard,
		ErrOut: ioutil.Discard,
	})

	do.Force = true
	do.IgnoreDaemonsets = true
	do.DeleteLocalData = true

	do.Timeout = 0

	c1, err := GetAdminConfig(ctx, cluster, owner)
	if err != nil {
		return NodeDrain{}, err
	}
	out := api.Convert_KubeConfig_To_Config(c1)
	// clientConfig := clientcmd.NewDefaultClientConfig(*out, &clientcmd.ConfigOverrides{})
	//	do.Factory = cmdutil.NewFactory(clientConfig)

	factory := cmdutil.NewFactory(&restClientGetter{out})

	return NodeDrain{o: do, ctx: ctx, kc: kc, f: factory}, nil
}

func (nd *NodeDrain) Apply() error {
	cmd := &cobra.Command{}
	// https://github.com/kubernetes/kubernetes/blob/7377c5911a5e2d5f18dfb15617316a891e661f22/pkg/kubectl/cmd/drain.go#L215
	cmd.Flags().String("selector", nd.o.Selector, "Selector (label query) to filter on")
	// https://github.com/kubernetes/kubernetes/blob/7377c5911a5e2d5f18dfb15617316a891e661f22/pkg/kubectl/cmd/drain.go#L227
	cmdutil.AddDryRunFlag(cmd)
	if err := nd.o.Complete(nd.f, cmd, []string{nd.Node}); err != nil {
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
