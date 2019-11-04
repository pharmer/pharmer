/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cloud

import (
	"errors"
	"sync"

	"pharmer.dev/pharmer/cloud/utils/certificates"

	"k8s.io/klog"
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
		klog.Fatalf("Cloud provider %q was registered twice", name)
	}
	klog.V(1).Infof("Registered cloud provider %q", name)
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
