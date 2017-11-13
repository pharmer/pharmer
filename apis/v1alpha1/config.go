package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LocalSpec struct {
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
}

type S3Spec struct {
	Endpoint string `json:"endpoint,omitempty" protobuf:"bytes,1,opt,name=endpoint"`
	Bucket   string `json:"bucket,omiempty" protobuf:"bytes,2,opt,name=bucket"`
	Prefix   string `json:"prefix,omitempty" protobuf:"bytes,3,opt,name=prefix"`
}

type GCSSpec struct {
	Bucket string `json:"bucket,omiempty" protobuf:"bytes,1,opt,name=bucket"`
	Prefix string `json:"prefix,omitempty" protobuf:"bytes,2,opt,name=prefix"`
}

type AzureStorageSpec struct {
	Container string `json:"container,omitempty" protobuf:"bytes,1,opt,name=container"`
	Prefix    string `json:"prefix,omitempty" protobuf:"bytes,2,opt,name=prefix"`
}

type SwiftSpec struct {
	Container string `json:"container,omitempty" protobuf:"bytes,1,opt,name=container"`
	Prefix    string `json:"prefix,omitempty" protobuf:"bytes,2,opt,name=prefix"`
}

type PostgresSpec struct {
	DbName   string `json:"dbName,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int64  `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

type StorageBackend struct {
	CredentialName string `json:"credentialName,omitempty" protobuf:"bytes,1,opt,name=credentialName"`

	Local    *LocalSpec        `json:"local,omitempty"`
	S3       *S3Spec           `json:"s3,omitempty"`
	GCS      *GCSSpec          `json:"gcs,omitempty"`
	Azure    *AzureStorageSpec `json:"azure,omitempty"`
	Swift    *SwiftSpec        `json:"swift,omitempty"`
	Postgres *PostgresSpec     `json:"postgres,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PharmerConfig struct {
	metav1.TypeMeta `json:",inline,omitempty,omitempty"`
	Context         string         `json:"context,omitempty"`
	Credentials     []Credential   `json:"credentials,omitempty"`
	Store           StorageBackend `json:"store,omitempty"`
}

func (pc PharmerConfig) GetStoreType() string {
	if pc.Store.Local != nil {
		return "Local"
	} else if pc.Store.S3 != nil {
		return "S3"
	} else if pc.Store.GCS != nil {
		return "GCS"
	} else if pc.Store.Azure != nil {
		return "Azure"
	} else if pc.Store.Swift != nil {
		return "OpenStack Swift"
	} else if pc.Store.Postgres != nil {
		return "Postgres"
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
