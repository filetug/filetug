package ftsettings

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDatatugUserDir_Success(t *testing.T) {
	t.Parallel()
	oldOsUserHomeDir := osUserHomeDir
	t.Cleanup(func() {
		osUserHomeDir = oldOsUserHomeDir
	})

	osUserHomeDir = func() (string, error) {
		return "/tmp/home", nil
	}

	userDir, err := GetDatatugUserDir()
	assert.NoError(t, err)
	expected := filepath.Join("/tmp/home", DatatugUserDir[2:])
	assert.Equal(t, expected, userDir)
}

func TestGetDatatugUserDir_Error(t *testing.T) {
	t.Parallel()
	oldOsUserHomeDir := osUserHomeDir
	t.Cleanup(func() {
		osUserHomeDir = oldOsUserHomeDir
	})

	wantErr := errors.New("home dir error")
	osUserHomeDir = func() (string, error) {
		return "", wantErr
	}

	userDir, err := GetDatatugUserDir()
	assert.ErrorIs(t, err, wantErr)
	assert.Equal(t, DatatugUserDir, userDir)
}
