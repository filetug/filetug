package filetug

import (
	"reflect"
	"testing"
)

func TestSplitNull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "empty_string",
			in:   "",
			want: nil,
		},
		{
			name: "two_entries",
			in:   "C:\x00D:\x00",
			want: []string{"C:", "D:"},
		},
		{
			name: "consecutive_nulls",
			in:   "A\x00\x00B\x00",
			want: []string{"A", "", "B"},
		},
		{
			name: "no_nulls",
			in:   "ABC",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitNull(tt.in)
			equal := reflect.DeepEqual(got, tt.want)
			if !equal {
				t.Fatalf("splitNull(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}
