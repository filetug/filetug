package gitutils

import (
	"strings"
	"testing"
)

func TestDirGitStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status *DirGitStatus
		want   string
	}{
		{
			name:   "nil",
			status: nil,
			want:   "",
		},
		{
			name:   "clean",
			status: &DirGitStatus{Branch: "main"},
			want:   "[gray]🌿main±0[-]",
		},
		{
			name:   "dirty",
			status: &DirGitStatus{Branch: "feature", FilesChanged: 2, Insertions: 10, Deletions: 5},
			want:   "[gray]🌿feature📄2[-][green]+10[-][red]-5[-]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("DirGitStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetGitStatus(t *testing.T) {
	status := GetGitStatus(".")
	if status != nil {
		s := status.String()
		if !strings.HasPrefix(s, "[gray]🌿") {
			t.Errorf("Expected status string starting with '[gray]🌿', got '%s'", s)
		}
	}
}
