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
	"testing"

	cloudapi "pharmer.dev/cloud/pkg/apis/cloud/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPharmerConfig_GetStoreType(t *testing.T) {
	type fields struct {
		TypeMeta    metav1.TypeMeta
		Context     string
		Credentials []cloudapi.Credential
		Store       StorageBackend
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "local",
			fields: fields{
				Store: StorageBackend{
					Local: &LocalSpec{},
				},
			},
			want: "Local",
		},
		{
			name: "s3",
			fields: fields{
				Store: StorageBackend{
					S3: &S3Spec{},
				},
			},
			want: "S3",
		},
		{
			name: "gcs",
			fields: fields{
				Store: StorageBackend{
					GCS: &GCSSpec{},
				},
			},
			want: "GCS",
		},
		{
			name: "azure",
			fields: fields{
				Store: StorageBackend{
					Azure: &AzureStorageSpec{},
				},
			},
			want: "Azure",
		},
		{
			name: "swift",
			fields: fields{
				Store: StorageBackend{
					Swift: &SwiftSpec{},
				},
			},
			want: "OpenStack Swift",
		},
		{
			name: "postgres",
			fields: fields{
				Store: StorageBackend{
					Postgres: &PostgresSpec{},
				},
			},
			want: "Postgres",
		},
		{
			name: "unknown",
			want: "<Unknown>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := PharmerConfig{
				TypeMeta:    tt.fields.TypeMeta,
				Context:     tt.fields.Context,
				Credentials: tt.fields.Credentials,
				Store:       tt.fields.Store,
			}
			if got := pc.GetStoreType(); got != tt.want {
				t.Errorf("PharmerConfig.GetStoreType() = %v, want %v", got, tt.want)
			}
		})
	}
}
