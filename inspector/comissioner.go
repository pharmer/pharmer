package inspector

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	appscodeSSH "github.com/appscode/api/ssh/v1beta1"
	"github.com/appscode/client/cli"
	"github.com/appscode/go/errors"
	vcs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/cenkalti/backoff"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Kube struct {
	Client        clientset.Interface
	Config        *rest.Config
	VoyagerClient vcs.VoyagerV1beta1Interface
}

type SSH struct {
	PublicKey  []byte
	PrivateKey []byte
}

type Cluster struct {
	Name           string
	Provider       string
	Namespace      string
	Credential     map[string]string
	CredentialPHID string
	*Kube
	*SSH
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

func New(kubeconfig, cluster string) (*Cluster, error) {
	c := &Cluster{
		Name: cluster,
	}
	kc, err := getKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	c.Kube = &Kube{
		Client:        clientset.NewForConfigOrDie(kc),
		Config:        kc,
		VoyagerClient: vcs.NewForConfigOrDie(kc),
	}

	return c, nil
}

func getKubeConfig(file string) (*rest.Config, error) {
	if file == "" {
		if _, err := os.Stat(clientcmd.RecommendedHomeFile); err == nil {
			file = clientcmd.RecommendedHomeFile
		} else {
			return nil, errors.New("No kube config file found").Err()
		}
	}
	return clientcmd.BuildConfigFromFlags("", file)
}

/*func (c *Cluster) LoadKubeClient() error {
	kc, err := getKubeConfig("")
	if err != nil {
		fmt.Println(err)
		log.Fatalln("Failed to load Kube Config")
		return errors.FromErr(err).Err()
	}

	c.Kube = &Kube{
		Client:        clientset.NewForConfigOrDie(kc),
		Config:        kc,
		VoyagerClient: vcs.NewForConfigOrDie(kc),
	}

	return nil
}*/

func (c *Cluster) LoadSSHKey(file string) error {
	if _, err := os.Stat(file); err != nil {
		return err
	}
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))
	c.SSH = &SSH{
		PrivateKey: bytes,
	}
	//block, _ := pem.Decode(bytes)

	return nil
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
