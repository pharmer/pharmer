package v1alpha1

import (
	"fmt"

	"github.com/appscode/mergo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type LocalSpec struct {
	Path string `json:"path,omitempty"`
}

type S3Spec struct {
	Endpoint string `json:"endpoint,omitempty"`
	Bucket   string `json:"bucket,omiempty"`
	Prefix   string `json:"prefix,omitempty"`
}

type GCSSpec struct {
	Bucket string `json:"bucket,omiempty"`
	Prefix string `json:"prefix,omitempty"`
}

type AzureStorageSpec struct {
	Container string `json:"container,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
}

type SwiftSpec struct {
	Container string `json:"container,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
}

type StorageBackend struct {
	CredentialName string `json:"credentialName,omitempty"`

	Local *LocalSpec        `json:"local,omitempty"`
	S3    *S3Spec           `json:"s3,omitempty"`
	GCS   *GCSSpec          `json:"gcs,omitempty"`
	Azure *AzureStorageSpec `json:"azure,omitempty"`
	Swift *SwiftSpec        `json:"swift,omitempty"`
}

type DNSProvider struct {
	CredentialName string `json:"credentialName,omitempty"`
}

type PharmerConfig struct {
	metav1.TypeMeta `json:",inline,omitempty,omitempty"`
	Context         string         `json:"context,omitempty"`
	Credentials     []Credential   `json:"credentials,omitempty"`
	Store           StorageBackend `json:"store,omitempty"`
	DNS             *DNSProvider   `json:"dns,omitempty"`
}

var _ runtime.Object = &PharmerConfig{}

func (pc *PharmerConfig) DeepCopyObject() runtime.Object {
	if pc == nil {
		return pc
	}
	out := new(PharmerConfig)
	mergo.MergeWithOverwrite(out, pc)
	return out
}

func (pc PharmerConfig) GetStoreType() string {
	if pc.Store.Local != nil {
		return "Local"
	} else if pc.Store.S3 != nil {
		return "S3"
	} else if pc.Store.S3 != nil {
		return "S3"
	} else if pc.Store.GCS != nil {
		return "GCS"
	} else if pc.Store.Azure != nil {
		return "Azure"
	} else if pc.Store.Swift != nil {
		return "OpenStack Swift"
	}
	return "<Unknown>"
}

func (pc PharmerConfig) GetDNSProviderType() string {
	if pc.DNS == nil {
		return "-"
	}
	if pc.DNS.CredentialName == "" {
		return "-"
	}
	for _, c := range pc.Credentials {
		if c.Name == pc.DNS.CredentialName {
			return c.Spec.Provider
		}
	}
	return "<Unknown>"
}

func (pc PharmerConfig) GetCredential(name string) (*Credential, error) {
	for _, c := range pc.Credentials {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("Missing credential %s", name)
}
