package dsstoreviewer

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"
)

func TestWriteFile(t *testing.T) {
	testdata := filepath.Join(".", "testdata", "00.DS_Store")
	var s Store
	err := s.ReadFile(testdata)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.DS_Store")

	err = s.WriteFile(tempFile, 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var s2 Store
	err = s2.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("ReadFile of written file failed: %v", err)
	}

	if len(s.Records) != len(s2.Records) {
		t.Errorf("expected %d records, got %d", len(s.Records), len(s2.Records))
	}
}

func TestRecordTypes(t *testing.T) {
	types := []string{"bool", "type", "long", "shor", "comp", "dutc", "blob", "ustr"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			s := &Store{}
			r := Record{
				FileName: "test",
				Type:     typ,
				Data:     []byte{0, 0, 0, 0, 0, 0, 0, 0},
			}
			switch typ {
			case "bool":
				r.Data = []byte{1}
			case "type", "long", "shor":
				r.Data = []byte{0, 0, 0, 1}
			case "blob", "ustr":
				r.DataLen = 1
				if typ == "ustr" {
					r.Data = []byte{0, 65} // 'A' in UTF16-BE
				} else {
					r.Data = []byte{0x42}
				}
			}
			s.Records = append(s.Records, r)

			buf := new(bytes.Buffer)
			err := s.Write(buf)
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}

			var s2 Store
			err = s2.Read(buf)
			if err != nil {
				t.Fatalf("Read failed: %v", err)
			}

			if len(s2.Records) != 1 {
				t.Fatalf("expected 1 record, got %d", len(s2.Records))
			}
			if s2.Records[0].Type != typ {
				t.Errorf("expected type %s, got %s", typ, s2.Records[0].Type)
			}
		})
	}
}

func TestWriteFileError(t *testing.T) {
	var s Store
	err := s.WriteFile("/non/existent/path/file", 0644)
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestManyRecords(t *testing.T) {
	s := &Store{}
	for i := 0; i < 2000; i++ {
		s.Records = append(s.Records, Record{
			FileName: fmt.Sprintf("file%04d", i),
			Type:     "long",
			Data:     []byte{0, 0, 0, byte(i % 256)},
		})
	}
	buf := new(bytes.Buffer)
	err := s.Write(buf)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	var s2 Store
	err = s2.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(s2.Records) != 2000 {
		t.Errorf("expected 2000 records, got %d", len(s2.Records))
	}
}

func TestWriteFreeBlocksMultiple(t *testing.T) {
	s := &Store{}
	buf := new(bytes.Buffer)
	freeBlocks := make([]freeBlock, 0)
	for i := uint32(0); i < 40; i++ {
		freeBlocks = append(freeBlocks, freeBlock{offset: i * 1024, size: 1024})
	}
	err := s.writeFreeBlocks(buf, freeBlocks)
	if err != nil {
		t.Fatalf("writeFreeBlocks failed: %v", err)
	}
}

func TestWriteAlignBlock(t *testing.T) {
	s := &Store{}
	buf := new(bytes.Buffer)
	buf.WriteByte(1)
	err := s.writeAlignBlock(buf, 8)
	if err != nil {
		t.Fatalf("writeAlignBlock failed: %v", err)
	}
	if buf.Len() != 8 {
		t.Errorf("expected length 8, got %d", buf.Len())
	}
}
