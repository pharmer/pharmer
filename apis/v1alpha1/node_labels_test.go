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

import "testing"

func TestNodeLabels_String(t *testing.T) {
	tests := []struct {
		name string
		n    NodeLabels
		want string
	}{
		{
			name: "",
			n: NodeLabels{
				"a": "b",
				"c": "d",
				"e": "f",
			},
			want: "a=b,c=d,e=f",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.n.String(); got != tt.want {
				t.Errorf("NodeLabels.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
