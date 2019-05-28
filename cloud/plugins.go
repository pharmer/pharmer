package cloud

import (
	"errors"
	"sync"

	"github.com/golang/glog"
	api "github.com/pharmer/pharmer/apis/v1beta1"
)

// Factory is a function that returns a cloud.ClusterManager.
// The config parameter provides an io.Reader handler to the factory in
// order to load specific configurations. If no configuration is provided
// the parameter is nil.
type Factory func(cluster *api.Cluster, certs *api.PharmerCertificates) Interface

// All registered cloud providers.
var (
	providersMutex sync.Mutex
	providers      = make(map[string]Factory)
)

// RegisterCloudManager registers a cloud.Factory by name.  This
// is expected to happen during app startup.
func RegisterCloudManager(name string, cloud Factory) {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	if _, found := providers[name]; found {
		glog.Fatalf("Cloud provider %q was registered twice", name)
	}
	glog.V(1).Infof("Registered cloud provider %q", name)
	providers[name] = cloud
}

// IsCloudManager returns true if name corresponds to an already registered
// cloud provider.
func IsCloudManager(name string) bool {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	_, found := providers[name]
	return found
}

// CloudManagers returns the name of all registered cloud providers in a
// string slice
func CloudManagers() []string {
	names := []string{}
	providersMutex.Lock()
	defer providersMutex.Unlock()
	for name := range providers {
		names = append(names, name)
	}
	return names
}

func GetCloudManager(cluster *api.Cluster) (Interface, error) {
	certs, err := getPharmerCerts(cluster.Name)
	if err != nil {
		return nil, err
	}
	return GetCloudManagerWithCerts(cluster, certs)
}

// GetCloudManager creates an instance of the named cloud provider, or nil if
// the name is not known.  The error return is only used if the named provider
// was known but failed to initialize. The config parameter specifies the
// io.Reader handler of the configuration file for the cloud provider, or nil
// for no configuation.
func GetCloudManagerWithCerts(cluster *api.Cluster, certs *api.PharmerCertificates) (Interface, error) {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	f, found := providers[cluster.Spec.Config.Cloud.CloudProvider]
	if !found {
		return nil, errors.New("not registerd")
	}
	return f(cluster, certs), nil
}
