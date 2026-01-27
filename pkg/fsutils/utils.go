package fsutils

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Decoder decodes
type Decoder interface {
	Decode(o interface{}) error
}

func ReadJSONFile(filePath string, required bool, o interface{}) (err error) {
	jsonDecoderFactory := func(r io.Reader) Decoder {
		return json.NewDecoder(r)
	}
	return ReadFile(filePath, required, o, jsonDecoderFactory)
}

func WriteJSONFile(filePath string, o interface{}) (err error) {
	var file *os.File
	file, err = os.Create(filePath)
	if err != nil {
		return
	}
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")
	return encoder.Encode(o)
}

func ReadFile(filePath string, required bool, o interface{}, newDecoder func(r io.Reader) Decoder) (err error) {
	var file *os.File
	if file, err = os.Open(filePath); err != nil {
		if os.IsNotExist(err) && !required {
			err = nil
		}
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("failed to close file %v: %v", filePath, err)
		}
	}()
	decoder := newDecoder(file)
	if err = decoder.Decode(o); err != nil {
		return err
	}
	return err
}

func DirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err // some other error
	}
	return info.IsDir(), nil
}

// ExpandHome expands leading ~ to the user's home directory.
func ExpandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			if p == "~" {
				return home
			}
			trimmed := strings.TrimPrefix(p, "~/")
			p = filepath.Join(home, trimmed)
			p = strings.TrimSuffix(p, "/")
			return p
		}
	}
	return p
}
