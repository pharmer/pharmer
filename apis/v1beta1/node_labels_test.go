package v1beta1

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
