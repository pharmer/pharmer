package cloud

import (
	"encoding/base64"
	"sync"

	"github.com/appscode/errors"
	_env "github.com/appscode/go/env"
	"github.com/appscode/pharmer/api"
	_ "github.com/appscode/searchlight/api/install"
	scs "github.com/appscode/searchlight/client/clientset"
	_ "github.com/appscode/stash/api/install"
	rc "github.com/appscode/stash/client/clientset"
	_ "github.com/appscode/voyager/api/install"
	vcs "github.com/appscode/voyager/client/clientset"
	k8sdb "github.com/k8sdb/apimachinery/client/clientset"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type kubeClient struct {
	Client            clientset.Interface
	VoyagerClient     vcs.ExtensionInterface
	SearchlightClient scs.ExtensionInterface
	KubeDBClient      k8sdb.ExtensionInterface
	StashClient       rc.ExtensionInterface
	config            *rest.Config
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
	voyagerClient, err := vcs.NewForConfig(kubeconfig)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	searchlightClient, err := scs.NewForConfig(kubeconfig)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	kubedbClient, err := k8sdb.NewForConfig(kubeconfig)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	stashClient, err := rc.NewForConfig(kubeconfig)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	return &kubeClient{
		Client:            client,
		VoyagerClient:     voyagerClient,
		SearchlightClient: searchlightClient,
		KubeDBClient:      kubedbClient,
		StashClient:       stashClient,
		config:            kubeconfig,
	}, nil
}

func (c *kubeClient) Config() *rest.Config {
	cfg := *c.config // copy data
	return &cfg
}

func CheckFake() bool {
	return fakeKube.useFakeServer
}
