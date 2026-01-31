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
	t.Parallel()
	tempDir := filepath.ToSlash(t.TempDir())
	ctx := files.NewDirContext(osfile.NewStore("/"), tempDir, nil)

	assert.Equal(t, path.Dir(tempDir), ctx.DirPath())
	assert.Equal(t, tempDir, ctx.FullName())
	assert.Equal(t, tempDir, ctx.String())
	assert.Equal(t, path.Base(tempDir), ctx.Name())
	assert.True(t, ctx.IsDir())
	assert.Equal(t, os.ModeDir, ctx.Type())
	info, err := ctx.Info()
	assert.NoError(t, err)
	assert.NotNil(t, info)

	root := files.NewDirContext(nil, "/", nil)
	assert.Equal(t, "/", root.Name())

	trailing := files.NewDirContext(nil, tempDir+"/", nil)
	assert.Equal(t, path.Base(tempDir), trailing.Name())

	empty := files.NewDirContext(nil, "", nil)
	assert.Equal(t, "", empty.DirPath())
	assert.Equal(t, "", empty.Name())
	info, err = empty.Info()
	assert.NoError(t, err)
	assert.Nil(t, info)

	nonFileStore := newMockStoreWithRoot(t, url.URL{Scheme: "ftp"})
	nonFileCtx := files.NewDirContext(nonFileStore, tempDir, nil)
	info, err = nonFileCtx.Info()
	assert.NoError(t, err)
	assert.Nil(t, info)
}
