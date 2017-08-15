package cloud

import (
	"sync"

	"github.com/golang/glog"
)

// All registered cloud providers.
var (
	providersMutex sync.Mutex
	providers      = make(map[string]Provider)
)

// RegisterProvider registers a cloudprovider.Factory by name.  This
// is expected to happen during app startup.
func RegisterProvider(name string, cloud Provider) {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	if _, found := providers[name]; found {
		glog.Fatalf("Cloud provider %q was registered twice", name)
	}
	glog.V(1).Infof("Registered cloud provider %q", name)
	providers[name] = cloud
}

// IsProvider returns true if name corresponds to an already registered
// cloud provider.
func IsProvider(name string) bool {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	_, found := providers[name]
	return found
}

// Providers returns the name of all registered cloud providers in a
// string slice
func Providers() []string {
	names := []string{}
	providersMutex.Lock()
	defer providersMutex.Unlock()
	for name := range providers {
		names = append(names, name)
	}
	return names
}

// GetProvider creates an instance of the named cloud provider, or nil if
// the name is not known.  The error return is only used if the named provider
// was known but failed to initialize. The config parameter specifies the
// api.PharmerConfig handler of the configuration file for the cloud provider, or nil
// for no configuation.
func GetProvider(name string) (Provider, error) {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	f, found := providers[name]
	if !found {
		return nil, nil
	}
	return f, nil
}
