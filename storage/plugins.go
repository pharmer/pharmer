package storage

import (
	"fmt"
	"sync"

	"github.com/appscode/pharmer/config"
	"github.com/golang/glog"
)

// Factory is a function that returns a cloud.Store.
// The config parameter provides an config.PharmerConfig handler to the factory in
// order to load specific configurations. If no configuration is provided
// the parameter is nil.
type Factory func(cfg *config.PharmerConfig) (Store, error)

// All registered storage providers.
var (
	providersMutex sync.Mutex
	providers      = make(map[string]Factory)
)

// RegisterStore registers a cloud.Factory by name.  This
// is expected to happen during app startup.
func RegisterStore(name string, cloud Factory) {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	if _, found := providers[name]; found {
		glog.Fatalf("Cloud provider %q was registered twice", name)
	}
	glog.V(1).Infof("Registered storage provider %q", name)
	providers[name] = cloud
}

// IsStore returns true if name corresponds to an already registered
// storage provider.
func IsStore(name string) bool {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	_, found := providers[name]
	return found
}

// Stores returns the name of all registered storage providers in a
// string slice
func Stores() []string {
	names := []string{}
	providersMutex.Lock()
	defer providersMutex.Unlock()
	for name := range providers {
		names = append(names, name)
	}
	return names
}

// GetStore creates an instance of the named storage provider, or nil if
// the name is not known.  The error return is only used if the named provider
// was known but failed to initialize. The config parameter specifies the
// config.PharmerConfig handler of the configuration file for the storage provider, or nil
// for no configuation.
func GetStore(name string, cfg *config.PharmerConfig) (Store, error) {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	f, found := providers[name]
	if !found {
		return nil, nil
	}
	return f(cfg)
}

// InitStore creates an instance of the named storage provider.
func InitStore(name string, configFilePath string) (Store, error) {
	var cloud Store
	var err error

	if name == "" {
		glog.Info("No storage provider specified.")
		return nil, nil
	}

	if configFilePath != "" {
		cfg, err := config.LoadConfig(configFilePath)
		if err != nil {
			glog.Fatalf("Couldn't open storage provider configuration %s: %#v",
				configFilePath, err)
		}
		cloud, err = GetStore(name, cfg)
	} else {
		// Pass explicit nil so plugins can actually check for nil. See
		// "Why is my nil error value not equal to nil?" in golang.org/doc/faq.
		cloud, err = GetStore(name, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("could not init storage provider %q: %v", name, err)
	}
	if cloud == nil {
		return nil, fmt.Errorf("unknown storage provider %q", name)
	}

	return cloud, nil
}
