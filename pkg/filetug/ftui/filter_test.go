package ftui

import (
	"os"
	"testing"

	"github.com/filetug/filetug/pkg/files"
)

func TestFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		filter Filter
		want   bool
	}{
		{
			name:   "empty_extensions",
			filter: Filter{Extensions: []string{}},
			want:   true,
		},
		{
			name:   "non_empty_extensions",
			filter: Filter{Extensions: []string{".go"}},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_IsVisible(t *testing.T) {
	tests := []struct {
		name   string
		filter Filter
		entry  files.DirEntry
		want   bool
	}{
		{
			name:   "unconfigured_filter_shows_everything",
			filter: Filter{},
			entry:  files.NewDirEntry("some_file.txt", false),
			want:   true,
		},
		{
			name:   "unconfigured_filter_hides_hidden",
			filter: Filter{},
			entry:  files.NewDirEntry(".hidden", false),
			want:   false,
		},
		{
			name:   "unconfigured_filter_hides_dirs",
			filter: Filter{},
			entry:  files.NewDirEntry("dir", true),
			want:   false,
		},
		{
			name:   "hidden_file_show_hidden_filter",
			filter: Filter{ShowHidden: true},
			entry:  files.NewDirEntry(".hidden", false),
			want:   true,
		},
		{
			name:   "directory_no_show_dirs",
			filter: Filter{ShowDirs: false},
			entry:  files.NewDirEntry("dir", true),
			want:   false,
		},
		{
			name:   "directory_show_dirs",
			filter: Filter{ShowDirs: true},
			entry:  files.NewDirEntry("dir", true),
			want:   true,
		},
		{
			name:   "extension_match",
			filter: Filter{Extensions: []string{".go"}},
			entry:  files.NewDirEntry("main.go", false),
			want:   true,
		},
		{
			name:   "extension_mismatch",
			filter: Filter{Extensions: []string{".txt"}},
			entry:  files.NewDirEntry("main.go", false),
			want:   false,
		},
		{
			name: "mask_filter_match",
			filter: Filter{
				Extensions: []string{".go"},
				MaskFilter: func(entry os.DirEntry) bool {
					return entry.Name() == "main.go"
				},
			},
			entry: files.NewDirEntry("main.go", false),
			want:  true,
		},
		{
			name: "mask_filter_mismatch",
			filter: Filter{
				Extensions: []string{".go"},
				MaskFilter: func(entry os.DirEntry) bool {
					return entry.Name() != "main.go"
				},
			},
			entry: files.NewDirEntry("main.go", false),
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Filter.IsVisible expects os.DirEntry, files.DirEntry implements it.
			// But Filter.MaskFilter field is also FilterFunc which expects os.DirEntry.
			// In our tests we use files.DirEntry which implements os.DirEntry.
			if got := tt.filter.IsVisible(tt.entry); got != tt.want {
				t.Errorf("IsVisible() = %v, want %v", got, tt.want)
			}
		})
	}
}
