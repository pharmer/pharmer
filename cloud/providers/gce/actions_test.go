package gce

import (
	"reflect"
	"testing"

	api "github.com/pharmer/pharmer/apis/v1beta1"
)

func Test_addActs(t *testing.T) {
	type args struct {
		acts     []api.Action
		action   api.ActionType
		resource string
		message  string
	}
	tests := []struct {
		name string
		args args
		want []api.Action
	}{
		{
			name: "test-1",
			args: args{
				acts:     nil,
				action:   "a-1",
				resource: "r-1",
				message:  "m-1",
			},
			want: []api.Action{
				{
					Action:   "a-1",
					Resource: "r-1",
					Message:  "m-1",
				},
			},
		},
		{
			name: "test-2",
			args: args{
				acts: []api.Action{
					{
						Action:   "a-1",
						Resource: "r-1",
						Message:  "m-1",
					},
				},
				action:   "a-2",
				resource: "r-2",
				message:  "m-2",
			},
			want: []api.Action{
				{
					Action:   "a-1",
					Resource: "r-1",
					Message:  "m-1",
				},
				{
					Action:   "a-2",
					Resource: "r-2",
					Message:  "m-2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addActs(tt.args.acts, tt.args.action, tt.args.resource, tt.args.message); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addActs() = %v, want %v", got, tt.want)
			}
		})
	}
}
