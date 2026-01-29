package ftsettings

import (
	"os"
	"path/filepath"
)

const DatatugUserDir = "~/.filetug"

var osUserHomeDir = os.UserHomeDir

func GetDatatugUserDir() (string, error) {
	userHomeDir, err := osUserHomeDir()
	if err != nil {
		return DatatugUserDir, err
	}
	return filepath.Join(userHomeDir, DatatugUserDir[2:]), nil
}
