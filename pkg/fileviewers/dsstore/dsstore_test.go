package dsstore

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRead(t *testing.T) {
	testdata := filepath.Join(".", "testdata", "00.DS_Store")
	data, err := os.ReadFile(testdata)
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}
	bufferRead := bytes.NewBuffer(data)
	var s Store
	err = s.Read(bufferRead)
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}
	t.Logf("HeaderExtra size is %d", len(s.HeaderExtra))
	t.Logf("RootExtra size is %d", len(s.RootExtra))
	t.Logf("DSDBExtra size is %d", len(s.DSDBExtra))
	t.Logf("Records %d loaded", len(s.Records))
}

func TestWriteEmpty(t *testing.T) {
	bufferWrite := new(bytes.Buffer)
	var s Store
	s.HeaderExtra = []byte{1, 2, 3, 4, 5, 7, 8, 9, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 3, 4}
	err := s.Write(bufferWrite)
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}
	bufferRead := bytes.NewBuffer(bufferWrite.Bytes())
	err = s.Read(bufferRead)
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}
}

func TestWrite(t *testing.T) {
	testdata := filepath.Join(".", "testdata", "00.DS_Store")
	data, err := os.ReadFile(testdata)
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}
	bufferRead := bytes.NewBuffer(data)
	var s1, s2 Store
	err = s1.Read(bufferRead)
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}
	bufferWrite := new(bytes.Buffer)
	err = s1.Write(bufferWrite)
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}
	bufferRead = bytes.NewBuffer(bufferWrite.Bytes())
	err = s2.Read(bufferRead)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
}

func testMassiveFile(t *testing.T, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("%s : %s", path, err.Error())
		return
	}
	bufferRead := bytes.NewBuffer(data)
	var s1, s2 Store
	err = s1.Read(bufferRead)
	if err != nil {
		t.Errorf("%s : %s", path, err.Error())
		return
	}
	bufferWrite := new(bytes.Buffer)
	err = s1.Write(bufferWrite)
	if err != nil {
		t.Errorf("%s : %s", path, err.Error())
		return
	}
	bufferRead = bytes.NewBuffer(bufferWrite.Bytes())
	err = s2.Read(bufferRead)
	if err != nil {
		t.Errorf("%s : %s", path, err.Error())
	}
	// compare stores
	if !bytes.Equal(s1.HeaderExtra, s2.HeaderExtra) {
		t.Errorf("%s : HeaderExtra is different", path)
	}
	if !bytes.Equal(s1.RootExtra, s2.RootExtra) {
		t.Errorf("%s : RootExtra is different", path)
	}
	if !bytes.Equal(s1.DSDBExtra, s2.DSDBExtra) {
		t.Errorf("%s : DSDBExtra is different", path)
	}
	// records
	if len(s1.Records) != len(s2.Records) {
		t.Errorf("%s : Records is different", path)
	} else {
		for i := 0; i < len(s1.Records); i++ {
			if s1.Records[i].FileName != s2.Records[i].FileName {
				t.Errorf("%s : Records FileName is different", path)
			}
			if s1.Records[i].Type != s2.Records[i].Type {
				t.Errorf("%s : Records FileName is different", path)
			}
			if s1.Records[i].Extra != s2.Records[i].Extra {
				t.Errorf("%s : Records Extra is different", path)
			}
			if s1.Records[i].DataLen != s2.Records[i].DataLen {
				t.Errorf("%s : Records DataLen is different", path)
			}
			if !bytes.Equal(s1.Records[i].Data, s2.Records[i].Data) {
				t.Errorf("%s : Records Data is different", path)
			}
		}
	}
}

func TestMassive(t *testing.T) {
	testdata := filepath.Join(".", "testdata")
	files, err := os.ReadDir(testdata)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		testMassiveFile(t, filepath.Join(testdata, f.Name()))
	}
}
