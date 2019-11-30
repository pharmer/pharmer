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
package v1alpha1

import (
	cloudapi "pharmer.dev/cloud/apis/cloud/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LocalSpec struct {
	Path string `json:"path,omitempty"`
}

type S3Spec struct {
	Endpoint string `json:"endpoint,omitempty"`
	Bucket   string `json:"bucket,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
}

type GCSSpec struct {
	Bucket string `json:"bucket,omitempty"`
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

type PostgresSpec struct {
	DbName       string `json:"database,omitempty"`
	Host         string `json:"host,omitempty"`
	Port         int64  `json:"port,omitempty"`
	User         string `json:"user,omitempty"`
	Password     string `json:"password,omitempty"`
	MasterKeyURL string `json:"masterKeyURL,omitempty"`
}

type StorageBackend struct {
	CredentialName string `json:"credentialName,omitempty"`

	Local    *LocalSpec        `json:"local,omitempty"`
	S3       *S3Spec           `json:"s3,omitempty"`
	GCS      *GCSSpec          `json:"gcs,omitempty"`
	Azure    *AzureStorageSpec `json:"azure,omitempty"`
	Swift    *SwiftSpec        `json:"swift,omitempty"`
	Postgres *PostgresSpec     `json:"postgres,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PharmerConfig struct {
	metav1.TypeMeta `json:",inline"`
	Context         string                `json:"context,omitempty"`
	Credentials     []cloudapi.Credential `json:"credentials,omitempty"`
	Store           StorageBackend        `json:"store,omitempty"`
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
