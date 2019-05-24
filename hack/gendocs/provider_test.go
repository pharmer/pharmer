package main

import "testing"

func Test_genCloudProviderDocs(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "test",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := genCloudProviderDocs(); (err != nil) != tt.wantErr {
				t.Errorf("genCloudProviderDocs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
