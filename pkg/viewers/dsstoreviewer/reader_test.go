package dsstoreviewer

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"testing"
)

func TestReadFile(t *testing.T) {
	testdata := filepath.Join(".", "testdata", "00.DS_Store")
	var s Store
	err := s.ReadFile(testdata)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if len(s.Records) == 0 {
		t.Error("expected records to be loaded")
	}
}

func TestReadErrors(t *testing.T) {
	t.Run("ShortFile", func(t *testing.T) {
		var s Store
		err := s.Read(bytes.NewReader(make([]byte, 10)))
		if err == nil || err.Error() != "invalid file header" {
			t.Errorf("expected 'invalid file header' error, got %v", err)
		}
	})

	t.Run("InvalidMagic1", func(t *testing.T) {
		var s Store
		data := make([]byte, 36)
		data[0] = 0xFF // invalid magic 1
		err := s.Read(bytes.NewReader(data))
		if err == nil || err.Error() != "invalid first magic" {
			t.Errorf("expected 'invalid first magic' error, got %v", err)
		}
	})

	t.Run("InvalidMagic2", func(t *testing.T) {
		var s Store
		data := make([]byte, 36)
		data[3] = 0x01 // valid magic 1
		data[7] = 0xFF // invalid magic 2
		err := s.Read(bytes.NewReader(data))
		if err == nil || err.Error() != "invalid second magic" {
			t.Errorf("expected 'invalid second magic' error, got %v", err)
		}
	})

	t.Run("OffsetMismatch", func(t *testing.T) {
		var s Store
		data := make([]byte, 36)
		data[3] = 0x01                                              // magic 1
		data[4], data[5], data[6], data[7] = 0x42, 0x75, 0x64, 0x31 // magic 2
		data[11] = 0x10                                             // offset 1
		data[31] = 0x20                                             // offset 2
		err := s.Read(bytes.NewReader(data))
		if err == nil || err.Error() != "invalid header offset" {
			t.Errorf("expected 'invalid header offset' error, got %v", err)
		}
	})
}

func TestReadFileNonExistent(t *testing.T) {
	var s Store
	err := s.ReadFile("non-existent-file")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestUnknownRecordType(t *testing.T) {
	s := &Store{}
	s.Records = append(s.Records, Record{
		FileName: "test",
		Type:     "bool",
		Data:     []byte{1},
	})

	buf := new(bytes.Buffer)
	_ = s.Write(buf)

	data := buf.Bytes()
	idx := bytes.Index(data, []byte("bool"))
	if idx != -1 {
		copy(data[idx:], []byte("xxxx"))
	}

	var s2 Store
	err := s2.Read(bytes.NewReader(data))
	if err == nil {
		t.Error("expected error for unknown record type")
	}
}

func TestReadBlockNil(t *testing.T) {
	s := &Store{}
	// offset + 4 + size > len(fileData)
	buf := s.readBlock([]byte{1, 2, 3}, 0, 10)
	if buf != nil {
		t.Error("expected nil buffer for out of bounds block")
	}
}

func TestReadOffsetsError(t *testing.T) {
	s := &Store{}
	// Short buffer for count
	_, err := s.readOffsets(bytes.NewBuffer([]byte{1, 2, 3}))
	if err == nil {
		t.Error("expected error for short buffer in readOffsets")
	}
}

func TestReadTopicsError(t *testing.T) {
	s := &Store{}
	// Short buffer for count
	_, err := s.readTopics(bytes.NewBuffer([]byte{1, 2, 3}))
	if err == nil {
		t.Error("expected error for short buffer in readTopics")
	}
}

func TestReadFreeBlocksError(t *testing.T) {
	s := &Store{}
	// Short buffer
	err := s.readFreeBlocks(bytes.NewBuffer([]byte{1, 2, 3}))
	if err == nil {
		t.Error("expected error for short buffer in readFreeBlocks")
	}
}

func TestReadParseDSDBCorrupt(t *testing.T) {
	s := &Store{}
	s.Records = append(s.Records, Record{FileName: "test", Type: "bool", Data: []byte{1}})
	buf := new(bytes.Buffer)
	_ = s.Write(buf)
	data := buf.Bytes()

	// The DSDB block has a dummy value that must be 0x1000.
	// Let's find 0x1000 in the data and change it.
	// 0x1000 is [0, 0, 16, 0] in big endian.
	idx := bytes.Index(data, []byte{0, 0, 0x10, 0})
	if idx != -1 {
		data[idx+2] = 0xFF
	}

	var s2 Store
	err := s2.Read(bytes.NewReader(data))
	if err == nil || err.Error() != "invalid DSDB block" {
		t.Errorf("expected 'invalid DSDB block' error, got %v", err)
	}
}

func TestReadParseDataInvalidNode(t *testing.T) {
	s := &Store{}
	s.Records = append(s.Records, Record{FileName: "test", Type: "bool", Data: []byte{1}})
	buf := new(bytes.Buffer)
	_ = s.Write(buf)
	data := buf.Bytes()

	// readParseData checks if node >= len(offsets)
	err := s.readParseData(data, []uint32{0}, 10)
	if err == nil || err.Error() != "invalid data block" {
		t.Errorf("expected 'invalid data block' error, got %v", err)
	}
}

func TestReadParseDataNilBlock(t *testing.T) {
	s := &Store{}
	err := s.readParseData([]byte{1, 2, 3}, []uint32{0}, 0)
	if err == nil || err.Error() != "invalid data block" {
		t.Errorf("expected 'invalid data block' error, got %v", err)
	}
}

func TestReadParseRootNilBlock(t *testing.T) {
	s := &Store{}
	err := s.readParseRoot([]byte{1, 2, 3}, 100, 100)
	if err == nil || err.Error() != "invalid root block" {
		t.Errorf("expected 'invalid root block' error, got %v", err)
	}
}

func TestReadParseDSDBNilBlock(t *testing.T) {
	s := &Store{}
	err := s.readParseDSDB([]byte{1, 2, 3}, []uint32{0}, map[string]uint32{"DSDB": 0})
	if err == nil || err.Error() != "invalid DSDB block" {
		t.Errorf("expected 'invalid DSDB block' error, got %v", err)
	}
}

func TestReadParseDSDBInvalidNode(t *testing.T) {
	s := &Store{}
	err := s.readParseDSDB([]byte{1, 2, 3}, []uint32{0}, map[string]uint32{"DSDB": 10})
	if err == nil || err.Error() != "invalid DSDB block" {
		t.Errorf("expected 'invalid DSDB block' error, got %v", err)
	}
}

func TestReadParseDataRecursive(t *testing.T) {
	s := &Store{}
	offsets := []uint32{0, 32 + 5, 64 + 5} // size 32
	fileData := make([]byte, 256)

	// Block 1 (at offset 32): nextNode=2, count=1
	binary.BigEndian.PutUint32(fileData[36:], 2) // nextNode
	binary.BigEndian.PutUint32(fileData[40:], 1) // count
	// childNode
	binary.BigEndian.PutUint32(fileData[44:], 2) // childNode points to node 2
	// Then a record for this block
	binary.BigEndian.PutUint32(fileData[48:], 1) // lenBytes
	fileData[52] = 0
	fileData[53] = 65                            // "A"
	binary.BigEndian.PutUint32(fileData[54:], 0) // extra
	copy(fileData[58:], "bool")
	fileData[62] = 1 // bool data

	// Block 2 (at offset 64): nextNode=0, count=1
	binary.BigEndian.PutUint32(fileData[68:], 0) // nextNode
	binary.BigEndian.PutUint32(fileData[72:], 1) // count
	// readParseFile: lenBytes=1, name="B", extra=0, type="bool", data=1
	binary.BigEndian.PutUint32(fileData[76:], 1) // lenBytes
	fileData[80] = 0
	fileData[81] = 66                            // "B"
	binary.BigEndian.PutUint32(fileData[82:], 0) // extra
	copy(fileData[86:], "bool")
	fileData[90] = 1 // bool data

	err := s.readParseData(fileData, offsets, 1)
	if err != nil {
		t.Fatalf("readParseData recursive failed: %v", err)
	}
	if len(s.Records) != 3 {
		t.Errorf("expected 3 records, got %d", len(s.Records))
	}
}

func TestReadOffsetsLarge(t *testing.T) {
	s := &Store{}
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, uint32(300))
	_ = binary.Write(buf, binary.BigEndian, uint32(0)) // dummy
	for i := 0; i < 512; i++ {
		_ = binary.Write(buf, binary.BigEndian, uint32(i+1))
	}
	offsets, err := s.readOffsets(buf)
	if err != nil {
		t.Fatalf("readOffsets failed: %v", err)
	}
	if len(offsets) != 512 {
		t.Errorf("expected 512 offsets, got %d", len(offsets))
	}
}

func TestReadOffsetsSkipZero(t *testing.T) {
	s := &Store{}
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, uint32(1))
	_ = binary.Write(buf, binary.BigEndian, uint32(0)) // dummy
	for i := 0; i < 256; i++ {
		if i == 0 {
			_ = binary.Write(buf, binary.BigEndian, uint32(0))
		} else {
			_ = binary.Write(buf, binary.BigEndian, uint32(i))
		}
	}
	offsets, err := s.readOffsets(buf)
	if err != nil {
		t.Fatalf("readOffsets failed: %v", err)
	}
	if len(offsets) != 255 {
		t.Errorf("expected 255 offsets, got %d", len(offsets))
	}
}

func TestReadTopicsEmpty(t *testing.T) {
	s := &Store{}
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, uint32(0))
	topics, err := s.readTopics(buf)
	if err != nil {
		t.Fatalf("readTopics failed: %v", err)
	}
	if len(topics) != 0 {
		t.Errorf("expected 0 topics, got %d", len(topics))
	}
}

func TestReadFreeBlocksNonZero(t *testing.T) {
	s := &Store{}
	buf := new(bytes.Buffer)
	for i := 0; i < 32; i++ {
		if i == 0 {
			_ = binary.Write(buf, binary.BigEndian, uint32(1))
			_ = binary.Write(buf, binary.BigEndian, uint32(123))
		} else {
			_ = binary.Write(buf, binary.BigEndian, uint32(0))
		}
	}
	err := s.readFreeBlocks(buf)
	if err != nil {
		t.Fatalf("readFreeBlocks failed: %v", err)
	}
}
