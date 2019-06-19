package cloud

import (
	"errors"
	"sync"

	"github.com/golang/glog"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
)

// Factory is a function that returns a cloud.ClusterManager.
// The config parameter provides an io.Reader handler to the factory in
// order to load specific configurations. If no configuration is provided
// the parameter is nil.
type Factory func(scope *Scope) Interface

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

func GetCloudManager(s *Scope) (Interface, error) {
	if s.Certs == nil {
		certs, err := certificates.GetPharmerCerts(s.StoreProvider, s.Cluster.Name)
		if err != nil {
			return nil, err
		}
		s.Certs = certs
	}

	providersMutex.Lock()
	defer providersMutex.Unlock()
	f, found := providers[s.Cluster.Spec.Config.Cloud.CloudProvider]
	if !found {
		return nil, errors.New("cloud provider not registerd")
	}
	return f(s), nil
}
