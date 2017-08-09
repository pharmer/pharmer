package data

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/appscode/data/files"
)

func LoadLatestPkgData() (pkg PkgData, err error) {
	bytes, err := files.Asset("files/pkg.latest.json")
	if err != nil {
		return
	}

	// pkg := PkgData{}
	err = json.Unmarshal(bytes, &pkg)
	return
}

func LoadKubernetesVersion(provider, env, version string) (*CloudKubernetesVersion, error) {
	bytes, err := files.Asset("files/cloud_provider.json")
	if err != nil {
		return nil, err
	}
	var providers ClusterProviders
	err = json.Unmarshal(bytes, &providers)
	if err != nil {
		return nil, err
	}

	p, found := providers.Provider[provider]
	if !found {
		return nil, fmt.Errorf("Can't find ClusterProduct provider %v", provider)
	}
	versions, found := p.Kubernetes.VersionsByEnv[env]
	if !found {
		return nil, fmt.Errorf("Can't find ClusterProduct for provider %v in env %v", provider, env)
	}
	for _, v := range versions {
		if v.Version == version {
			return v, nil
		}
	}
	return nil, fmt.Errorf("Can't find Kubernetes version %v for %v in %v", version, provider, env)
}

func ClusterMachineType(cloud, externalSku string) (*InstanceType, error) {
	bytes, err := files.Asset("files/cloud_provider.json")
	if err != nil {
		return nil, err
	}

	var providers ClusterProviders
	err = json.Unmarshal(bytes, &providers)
	if err != nil {
		return nil, err
	}
	for _, instance := range providers.Provider[cloud].InstanceTypes {
		if instance.ExternalSku == externalSku {
			return instance, nil
		}
	}
	return nil, fmt.Errorf("No data found for instace %v for cloud provider %v.", externalSku, cloud)
}

func KubernetesPricingMetadata(cloud, externalSku string) (map[string]string, error) {
	machineType, err := ClusterMachineType(cloud, externalSku)
	if err != nil {
		return nil, err
	}
	metadata := make(map[string]string)
	switch machineType.RAM.(type) {
	case int, int32, int64:
		metadata["ram"] = strconv.Itoa(machineType.RAM.(int))
	case float64, float32:
		metadata["ram"] = strconv.FormatFloat(machineType.RAM.(float64), 'f', 2, 64)
	default:
		return nil, fmt.Errorf("Failed to parse memory metadata for instace %v for cloud provider %v.", externalSku, cloud)
	}
	metadata["cpu"] = strconv.Itoa(machineType.CPU)
	return metadata, nil
}

func LoadDNSProviderData() (dns []DNSProviders, err error) {
	bytes, err := files.Asset("files/dns_provider.json")
	if err != nil {
		return
	}
	// pkg := PkgData{}
	err = json.Unmarshal(bytes, &dns)
	return
}
