package store

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/klog"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/config"
)

// Factory is a function that returns a storage.Interface.
// The config parameter provides an io.Reader handler to the factory in
// order to load specific configurations. If no configuration is provided
// the parameter is nil.
type Factory func(cfg *api.PharmerConfig) (Interface, error)

// All registered cloud providers.
var (
	providersMutex sync.Mutex
	providers      = make(map[string]Factory)
)

// RegisterProvider registers a storage.Factory by name.  This
// is expected to happen during app startup.
func RegisterProvider(name string, cloud Factory) {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	if _, found := providers[name]; found {
		klog.Fatalf("Cloud provider %q was registered twice", name)
	}
	klog.V(1).Infof("Registered cloud provider %q", name)
	providers[name] = cloud
}

// GetProvider creates an node of the named cloud provider, or nil if
// the name is not known.  The error return is only used if the named provider
// was known but failed to initialize. The config parameter specifies the
// io.Reader handler of the configuration file for the cloud provider, or nil
// for no configuation.
func getProvider(name string, cfg *api.PharmerConfig) (Interface, error) {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	f, found := providers[name]
	if !found {
		return nil, errors.Errorf("provider %s not registered", name)
	}
	return f(cfg)
}

func GetStoreProvider(cmd *cobra.Command) (ResourceInterface, error) {
	cfgFile, _ := config.GetConfigFile(cmd.Flags())
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return nil, err
	}
	return NewStoreProvider(cfg)
}

func NewStoreInterface(cfg *api.PharmerConfig) (Interface, error) {
	var storeType string
	if cfg == nil {
		storeType = fakeUID
	} else if cfg.Store.Postgres != nil {
		storeType = xormUID
	} else {
		storeType = vfsUID
	}

	store, err := getProvider(storeType, cfg)
	if err != nil {
		return nil, err
	}
	return store, nil
}

func NewStoreProvider(cfg *api.PharmerConfig) (ResourceInterface, error) {
	store, err := NewStoreInterface(cfg)
	if err != nil {
		return nil, err
	}
	return store, nil
}
