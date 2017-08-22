package cloud

import (
	"encoding/base64"
	"sync"

	"github.com/appscode/errors"
	_env "github.com/appscode/go/env"
	"github.com/appscode/pharmer/api"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type kubeClient struct {
	Client clientset.Interface
	config *rest.Config
}

type fakeKubeClient struct {
	useFakeServer        bool
	fakeDirectoryAddress string
	fakeClient           *kubeClient
	once                 sync.Once
}

var fakeKube = fakeKubeClient{useFakeServer: false}

// WARNING:
// Returned KubeClient uses admin bearer token. This should only be used for cluster provisioning operations.
// For other cluster operations initiated by users, use KubeAddon context.
func NewAdminClient(cluster *api.Cluster) (*kubeClient, error) {
	kubeconfig := &rest.Config{
		Host:        cluster.Spec.ApiServerUrl,
		BearerToken: cluster.Spec.KubeBearerToken,
	}
	if _env.FromHost().DevMode() {
		kubeconfig.Insecure = true
	} else {
		caCert, err := base64.StdEncoding.DecodeString(cluster.Spec.CaCert)
		if err != nil {
			return nil, err
		}
		kubeconfig.TLSClientConfig = rest.TLSClientConfig{
			CAData: caCert,
		}
	}

	if fakeKube.useFakeServer { //for fake kube client
		return fakeKube.fakeClient, nil
	}

	// Initiate real kube client
	client, err := clientset.NewForConfig(kubeconfig)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	return &kubeClient{
		Client: client,
		config: kubeconfig,
	}, nil
}

func (c *kubeClient) Config() *rest.Config {
	cfg := *c.config // copy data
	return &cfg
}

func CheckFake() bool {
	return fakeKube.useFakeServer
}
