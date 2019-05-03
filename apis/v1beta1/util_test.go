package v1beta1

import (
	"errors"
	"testing"
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
