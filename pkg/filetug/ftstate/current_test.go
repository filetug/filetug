package ftstate

import (
	"testing"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/osfile"
)

func TestCurrent_SetDir_Dir(t *testing.T) {
	t.Parallel()
	var c Current
	if c.Dir() != nil {
		t.Fatal("expected nil dir by default")
	}

	dir := files.NewDirContext(nil, "/tmp", nil)
	c.SetDir(dir)
	if c.Dir() != dir {
		t.Fatal("expected SetDir to store the provided dir")
	}
}

func TestCurrent_NewDirContext_UsesStore(t *testing.T) {
	t.Parallel()
	store := osfile.NewStore("/")
	var c Current
	c.SetDir(files.NewDirContext(store, "/root", nil))

	dir := c.NewDirContext("/root/child", nil)
	if dir.Store() != store {
		t.Fatal("expected NewDirContext to use current store")
	}
	if dir.Path() != "/root/child" {
		t.Fatalf("expected path /root/child, got %q", dir.Path())
	}
}

func TestCurrent_NewDirContext_NilCurrent(t *testing.T) {
	t.Parallel()
	var c Current
	dir := c.NewDirContext("/tmp", nil)
	if dir.Store() != nil {
		t.Fatal("expected nil store when current dir is nil")
	}
}

func TestCurrent_ChangeDir(t *testing.T) {
	t.Parallel()
	store := osfile.NewStore("/")
	var c Current
	c.SetDir(files.NewDirContext(store, "/root", nil))

	c.ChangeDir("/root/next")
	if c.Dir() == nil || c.Dir().Path() != "/root/next" {
		t.Fatal("expected ChangeDir to update current dir")
	}
}

func TestCurrent_Store(t *testing.T) {
	t.Parallel()
	var c Current
	if c.Store() != nil {
		t.Fatal("expected nil store with no current dir")
	}

	store := osfile.NewStore("/")
	c.SetDir(files.NewDirContext(store, "/root", nil))
	if c.Store() != store {
		t.Fatal("expected Store to return current dir store")
	}
}

func TestCurrent_Store_NilReceiver(t *testing.T) {
	t.Parallel()
	var c *Current
	if c.Store() != nil {
		t.Fatal("expected nil store for nil receiver")
	}
}
