package commissioner

import (
	"fmt"
	"os"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	appscodeSSH "github.com/appscode/api/ssh/v1beta1"
	"github.com/appscode/client/cli"
	"github.com/appscode/errors"
	vcs "github.com/appscode/voyager/client/clientset"
	"github.com/cenkalti/backoff"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Kube struct {
	Client        clientset.Interface
	Config        *rest.Config
	VoyagerClient vcs.ExtensionInterface
}

type Cluster struct {
	Name           string
	Provider       string
	Namespace      string
	Credential     map[string]string
	CredentialPHID string
	*Kube
}

// Default values for ExponentialBackOff.
const (
	DefaultInitialInterval     = 50 * time.Millisecond
	DefaultRandomizationFactor = 0.5
	DefaultMultiplier          = 1.5
	DefaultMaxInterval         = 5 * time.Second
	DefaultMaxElapsedTime      = 5 * time.Minute
)

// NewExponentialBackOff creates an instance of ExponentialBackOff using default values.
func NewExponentialBackOff() *backoff.ExponentialBackOff {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     DefaultInitialInterval,
		RandomizationFactor: DefaultRandomizationFactor,
		Multiplier:          DefaultMultiplier,
		MaxInterval:         DefaultMaxInterval,
		MaxElapsedTime:      DefaultMaxElapsedTime,
		Clock:               backoff.SystemClock,
	}
	if b.RandomizationFactor < 0 {
		b.RandomizationFactor = 0
	} else if b.RandomizationFactor > 1 {
		b.RandomizationFactor = 1
	}
	b.Reset()
	return b
}

func NewComissionar(provider, cluster string) (*Cluster, error) {
	apprc, err := cli.LoadApprc()
	if err != nil {
		return &Cluster{}, errors.FromErr(err).Err()
	}
	namespace := apprc.GetAuth().TeamId
	return &Cluster{Provider: provider, Name: cluster, Namespace: namespace}, nil
}

func (c *Cluster) getKubeConfig() (*rest.Config, error) {
	req := proto.ClusterClientConfigRequest{
		Name: c.Name,
	}
	masterURL := ""
	if resp, err := c.callClusterConfigApi(req); err != nil {
		return nil, errors.FromErr(err).Err()
	} else {
		masterURL = resp.ApiServerUrl
	}
	var kubeConfig string = ""
	if _, err := os.Stat(clientcmd.RecommendedHomeFile); err == nil {
		kubeConfig = clientcmd.RecommendedHomeFile
	} else {
		return nil, errors.New("No kube config file found").Err()
	}
	fmt.Println("Master url = ", masterURL, kubeConfig)
	return clientcmd.BuildConfigFromFlags(masterURL, kubeConfig)

}

func (c *Cluster) callClusterConfigApi(req proto.ClusterClientConfigRequest) (*proto.ClusterClientConfigResponse, error) {
	var resp *proto.ClusterClientConfigResponse
	err := backoff.Retry(func() error {
		client, err := cli.Client("")
		if err != nil {
			return err
		}
		defer client.Close()
		resp, err = client.Kubernetes().V1beta1().Cluster().ClientConfig(client.Context(), &req)
		return err
	}, NewExponentialBackOff())
	return resp, err
}

func (c *Cluster) callClusterSSHApi(req appscodeSSH.SSHGetRequest) (*appscodeSSH.SSHGetResponse, error) {
	var resp *appscodeSSH.SSHGetResponse
	err := backoff.Retry(func() error {
		client, err := cli.Client("")
		if err != nil {
			return err
		}
		defer client.Close()
		resp, err = client.SSH().Get(client.Context(), &req)
		return err
	}, NewExponentialBackOff())
	return resp, err
}
