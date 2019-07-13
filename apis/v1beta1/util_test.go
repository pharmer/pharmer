package v1beta1

import (
	"testing"

	"github.com/pkg/errors"
	v1 "pharmer.dev/cloud/pkg/apis/cloud/v1"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func TestErrObjectModified(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "error object modified",
			args: args{
				err: errors.New("Operation cannot be fulfilled on machines.cluster.k8s.io \"pharmer-test-txxttp-master-0\": the object has been modified; please apply your changes to the latest version and try again"),
			},
			want: true,
		},
		{
			name: "not correct error",
			args: args{
				err: errors.New("hello-world"),
			},
			want: false,
		},
		{
			name: "nil error",
			args: args{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrObjectModified(tt.args.err); got != tt.want {
				t.Errorf("ErrObjectModified() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAssignTypeKind(t *testing.T) {
	type args struct {
		v interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "pharmerconfig",
			args: args{
				v: &PharmerConfig{},
			},
			wantErr: false,
		},
		{
			name: "cluster",
			args: args{
				v: &Cluster{},
			},
			wantErr: false,
		},
		{
			name: "credential",
			args: args{
				v: &v1.Credential{},
			},
			wantErr: false,
		},
		{
			name: "capi cluster",
			args: args{
				v: &v1alpha1.Cluster{},
			},
			wantErr: false,
		},
		{
			name: "capi machine",
			args: args{
				v: &v1alpha1.Machine{},
			},
			wantErr: false,
		},
		{
			name: "capi machineset",
			args: args{
				v: &v1alpha1.MachineSet{},
			},
			wantErr: false,
		},
		{
			name: "nil",
			args: args{
				v: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AssignTypeKind(tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("AssignTypeKind() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestErrAlreadyExist(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "exists",
			args: args{
				err: errors.New("cluster already exists"),
			},
			want: true,
		},
		{
			name: "doesn't",
			args: args{
				err: errors.New("abcd"),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrAlreadyExist(tt.args.err); got != tt.want {
				t.Errorf("ErrAlreadyExist() = %v, want %v", got, tt.want)
			}
		})
	}
}
