package viewers

import (
	"testing"
)

func TestMetaTable_SetMeta(t *testing.T) {
	mt := NewMetaTable()
	meta := &Meta{
		Groups: []*MetaGroup{
			{
				ID:    "group1",
				Title: "Group 1",
				Records: []*MetaRecord{
					{
						ID:         "rec1",
						Title:      "Rec 1",
						Value:      "Val 1",
						ValueAlign: AlignLeft,
					},
					{
						ID:         "rec2",
						Title:      "Rec 2",
						Value:      "Val 2",
						ValueAlign: AlignRight,
					},
				},
			},
		},
	}

	mt.SetMeta(meta)

	// Check row count: 1 group title + 2 records = 3 rows
	if mt.GetRowCount() != 3 {
		t.Errorf("expected 3 rows, got %d", mt.GetRowCount())
	}

	// Check group title
	if mt.GetCell(0, 0).Text != "Group 1" {
		t.Errorf("expected 'Group 1' at (0,0), got '%s'", mt.GetCell(0, 0).Text)
	}

	// Check record 1
	if mt.GetCell(1, 0).Text != "  Rec 1" {
		t.Errorf("expected '  Rec 1' at (1,0), got '%s'", mt.GetCell(1, 0).Text)
	}
	if mt.GetCell(1, 1).Text != "Val 1" {
		t.Errorf("expected 'Val 1' at (1,1), got '%s'", mt.GetCell(1, 1).Text)
	}

	// Check record 2
	if mt.GetCell(2, 0).Text != "  Rec 2" {
		t.Errorf("expected '  Rec 2' at (2,0), got '%s'", mt.GetCell(2, 0).Text)
	}
	if mt.GetCell(2, 1).Text != "Val 2" {
		t.Errorf("expected 'Val 2' at (2,1), got '%s'", mt.GetCell(2, 1).Text)
	}
}
