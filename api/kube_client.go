package api

import (
	"sync"

	"github.com/appscode/errors"
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

func NewKubeClient(config *rest.Config) (*kubeClient, error) {
	if fakeKube.useFakeServer { //for fake kube client
		return fakeKube.fakeClient, nil
	}

	// Initiate real kube client
	client, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	voyagerClient, err := vcs.NewForConfig(config)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	searchlightClient, err := scs.NewForConfig(config)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	kubedbClient, err := k8sdb.NewForConfig(config)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	stashClient, err := rc.NewForConfig(config)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	return &kubeClient{
		Client:            client,
		VoyagerClient:     voyagerClient,
		SearchlightClient: searchlightClient,
		KubeDBClient:      kubedbClient,
		StashClient:       stashClient,
		config:            config,
	}, nil
}

func (c *kubeClient) Config() *rest.Config {
	cfg := *c.config // copy data
	return &cfg
}

func CheckFake() bool {
	return fakeKube.useFakeServer
}
