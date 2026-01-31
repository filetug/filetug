package masks

import (
	"testing"
)

func TestPattern_Match(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		pattern  Pattern
		fileName string
		want     bool
		wantErr  bool
	}{
		{
			name: "Inclusive match",
			pattern: Pattern{
				Type:  Inclusive,
				Regex: `\.go$`,
			},
			fileName: "test.go",
			want:     true,
			wantErr:  false,
		},
		{
			name: "Inclusive no match",
			pattern: Pattern{
				Type:  Inclusive,
				Regex: `\.go$`,
			},
			fileName: "test.txt",
			want:     false,
			wantErr:  false,
		},
		{
			name: "Invalid regex",
			pattern: Pattern{
				Type:  Inclusive,
				Regex: `[`,
			},
			fileName: "test.go",
			want:     false,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.pattern.Match(tt.fileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Match() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Match() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuiltInRegex(t *testing.T) {
	t.Parallel()
	// Testing if built-in regex patterns are valid
	masks := createBuiltInMasks()
	for _, mask := range masks {
		for _, pattern := range mask.Patterns {
			t.Run(mask.Name+"_"+string(pattern.Type), func(t *testing.T) {
				_, err := pattern.Match("any_file")
				if err != nil {
					t.Errorf("Built-in pattern %q is invalid: %v", pattern.Regex, err)
				}
			})
		}
	}
}
