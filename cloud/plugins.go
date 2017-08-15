package cloud

import (
	"fmt"
	"sync"

	"github.com/appscode/pharmer/config"
	"github.com/golang/glog"
)

// Factory is a function that returns a cloud.Provider.
// The config parameter provides an config.PharmerConfig handler to the factory in
// order to load specific configurations. If no configuration is provided
// the parameter is nil.
type Factory func(cfg *config.PharmerConfig) (Provider, error)

// All registered cloud providers.
var (
	providersMutex sync.Mutex
	providers      = make(map[string]Factory)
)

// RegisterProvider registers a cloud.Factory by name.  This
// is expected to happen during app startup.
func RegisterProvider(name string, cloud Factory) {
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
// config.PharmerConfig handler of the configuration file for the cloud provider, or nil
// for no configuation.
func GetProvider(name string, cfg *config.PharmerConfig) (Provider, error) {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	f, found := providers[name]
	if !found {
		return nil, nil
	}
	return f(cfg)
}

// InitProvider creates an instance of the named cloud provider.
func InitProvider(name string, configFilePath string) (Provider, error) {
	var cloud Provider
	var err error

	if name == "" {
		glog.Info("No cloud provider specified.")
		return nil, nil
	}

	if configFilePath != "" {
		cfg, err := config.LoadConfig(configFilePath)
		if err != nil {
			glog.Fatalf("Couldn't open cloud provider configuration %s: %#v",
				configFilePath, err)
		}
		cloud, err = GetProvider(name, cfg)
	} else {
		// Pass explicit nil so plugins can actually check for nil. See
		// "Why is my nil error value not equal to nil?" in golang.org/doc/faq.
		cloud, err = GetProvider(name, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("could not init cloud provider %q: %v", name, err)
	}
	if cloud == nil {
		return nil, fmt.Errorf("unknown cloud provider %q", name)
	}

	return cloud, nil
}
