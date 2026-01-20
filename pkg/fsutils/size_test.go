package fsutils

import (
	"testing"
)

func TestGetSizeShortText(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0B"},
		{500, "500B"},
		{1023, "1023B"},
		{1024, "1KB"},
		{1025, "1KB"},
		{1535, "1KB"},
		{1536, "2KB"},
		{2000, "2KB"},
		{1024 * 1024, "1MB"},
		{1024 * 1024 * 1024, "1GB"},
		{1024 * 1024 * 1024 * 1024, "1TB"},
		{2 * 1024 * 1024, "2MB"},
		{1024*1024 + 512*1024 - 1, "1MB"},
		{1024*1024 + 512*1024, "2MB"},
		{1024 * 1024 * 1024 * 1024 * 1024, "1024TB"},
		{1024*1024*1024*1024*1024 - 1024*1024*1024*1024/2, "1024TB"},
		{1024*1024*1024 - 1, "1GB"},
		{1024*1024*1024 - 1024*1024/2, "1GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			actual := GetSizeShortText(tt.size)
			if actual != tt.expected {
				t.Errorf("GetSizeShortText(%d) = %s; want %s", tt.size, actual, tt.expected)
			}
		})
	}
}
