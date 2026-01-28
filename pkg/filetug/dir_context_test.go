package filetug

import (
	"net/url"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/stretchr/testify/assert"
)

func TestDirContextEntryMethods(t *testing.T) {
	tempDir := filepath.ToSlash(t.TempDir())
	ctx := &files.DirContext{
		Store: osfile.NewStore("/"),
		Path:  tempDir,
	}

	assert.Equal(t, path.Dir(tempDir), ctx.DirPath())
	assert.Equal(t, tempDir, ctx.FullName())
	assert.Equal(t, tempDir, ctx.String())
	assert.Equal(t, path.Base(tempDir), ctx.Name())
	assert.True(t, ctx.IsDir())
	assert.Equal(t, os.ModeDir, ctx.Type())
	info, err := ctx.Info()
	assert.NoError(t, err)
	assert.NotNil(t, info)

	root := &files.DirContext{Path: "/"}
	assert.Equal(t, "/", root.Name())

	trailing := &files.DirContext{Path: tempDir + "/"}
	assert.Equal(t, path.Base(tempDir), trailing.Name())

	empty := &files.DirContext{}
	assert.Equal(t, "", empty.DirPath())
	assert.Equal(t, "", empty.Name())
	info, err = empty.Info()
	assert.NoError(t, err)
	assert.Nil(t, info)

	nonFileStore := mockStore{root: url.URL{Scheme: "ftp"}}
	nonFileCtx := &files.DirContext{Store: nonFileStore, Path: tempDir}
	info, err = nonFileCtx.Info()
	assert.NoError(t, err)
	assert.Nil(t, info)
}
