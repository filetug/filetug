package masks

import "testing"

func TestMask_Match(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		mask     Mask
		fileName string
		want     bool
		wantErr  bool
	}{
		{
			name: "Inclusive match",
			mask: Mask{
				Name: "Go files",
				Patterns: []Pattern{
					{Type: Inclusive, Regex: `\.go$`},
				},
			},
			fileName: "main.go",
			want:     true,
			wantErr:  false,
		},
		{
			name: "Inclusive no match",
			mask: Mask{
				Name: "Go files",
				Patterns: []Pattern{
					{Type: Inclusive, Regex: `\.go$`},
				},
			},
			fileName: "README.md",
			want:     false,
			wantErr:  false,
		},
		{
			name: "Exclusive match (should exclude)",
			mask: Mask{
				Name: "No tests",
				Patterns: []Pattern{
					{Type: Inclusive, Regex: `\.go$`},
					{Type: Exclusive, Regex: `_test\.go$`},
				},
			},
			fileName: "main_test.go",
			want:     false,
			wantErr:  false,
		},
		{
			name: "Exclusive no match (should include)",
			mask: Mask{
				Name: "No tests",
				Patterns: []Pattern{
					{Type: Inclusive, Regex: `\.go$`},
					{Type: Exclusive, Regex: `_test\.go$`},
				},
			},
			fileName: "main.go",
			want:     true,
			wantErr:  false,
		},
		{
			name: "Multiple inclusive",
			mask: Mask{
				Name: "Web",
				Patterns: []Pattern{
					{Type: Inclusive, Regex: `\.html$`},
					{Type: Inclusive, Regex: `\.css$`},
				},
			},
			fileName: "style.css",
			want:     true,
			wantErr:  false,
		},
		{
			name: "Invalid regex in one pattern",
			mask: Mask{
				Name: "Invalid",
				Patterns: []Pattern{
					{Type: Inclusive, Regex: `[`},
				},
			},
			fileName: "any",
			want:     false,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.mask.Match(tt.fileName)
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

func TestMask_String(t *testing.T) {
	t.Parallel()
	m := &Mask{
		Name: "Test",
		Patterns: []Pattern{
			{Type: Inclusive, Regex: ".*"},
		},
	}
	got := m.String()
	want := `Mask{Name: "Test", Patterns: [{Type:inclusive Regex:.* re:<nil>}]}`
	if got != want {
		t.Errorf("String() = %v, want %v", got, want)
	}
}
