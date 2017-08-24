package api

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

type AzureSpec struct {
	Container string `json:"container,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
}

type SwiftSpec struct {
	Container string `json:"container,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
}

type StorageBackend struct {
	CredentialName string `json:"credentialName,omitempty"`

	Local *LocalSpec `json:"local,omitempty"`
	S3    *S3Spec    `json:"s3,omitempty"`
	GCS   *GCSSpec   `json:"gcs,omitempty"`
	Azure *AzureSpec `json:"azure,omitempty"`
	Swift *SwiftSpec `json:"swift,omitempty"`
}

type DNSProvider struct {
	CredentialName string `json:"credentialName,omitempty"`
}

type PharmerConfig struct {
	TypeMeta    `json:",inline,omitempty"`
	Context     string         `json:context,omitempty`
	Credentials []Credential   `json:"credentials,omitempty"`
	Store       StorageBackend `json:"store,omitempty"`
	DNS         *DNSProvider   `json:"dns,omitempty"`
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
