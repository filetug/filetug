package filetug

import (
	"context"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftfav"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/go-git/go-git/v5"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestActivateFavorite_NilNav(t *testing.T) {
	t.Parallel()
	panel := newTestFavoritesPanel(nil)
	panel.activateFavorite(ftfav.Favorite{}, false)
}

func TestActivateFavorite_NilStore(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	nav.store = nil
	panel := newTestFavoritesPanel(nav)
	panel.activateFavorite(ftfav.Favorite{}, false)
}

func TestOnGitStatus_NilStatus(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	fp := nav.files
	rows := NewFileRows(files.NewDirContext(nil, "/tmp", nil))
	table := tview.NewTable()
	called := false
	queueUpdateDraw := func(f func()) { called = true }

	fp.onGitStatus(nil, rows, table, queueUpdateDraw, "/tmp/file.txt", false)
	assert.False(t, called)
}

func TestOnGitStatus_EmptyStatusText(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	fp := nav.files
	rows := NewFileRows(files.NewDirContext(nil, "/tmp", nil))
	table := tview.NewTable()
	called := false
	queueUpdateDraw := func(f func()) { called = true }

	// Status with no changes and not a repo root => gitStatusText returns ""
	status := &gitutils.RepoStatus{Branch: "main"}
	fp.onGitStatus(status, rows, table, queueUpdateDraw, "/tmp/file.txt", false)
	assert.False(t, called)
}

func TestOnGitStatus_NotUpdated(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	fp := nav.files

	repoDir := t.TempDir()
	dirContext := files.NewDirContext(nil, repoDir, nil)
	rows := NewFileRows(dirContext)
	entry := files.NewEntryWithDirPath(files.NewDirEntry("file.txt", false), repoDir)
	rows.AllEntries = []files.EntryWithDirPath{entry}
	rows.VisibleEntries = rows.AllEntries
	table := tview.NewTable()
	called := false
	queueUpdateDraw := func(f func()) { called = true }

	fullPath := filepath.Join(repoDir, "file.txt")
	status := &gitutils.RepoStatus{
		Branch:             "main",
		DirGitChangesStats: gitutils.DirGitChangesStats{FilesChanged: 1},
	}

	// First call sets the text
	statusText := nav.gitStatusText(status, fullPath, false)
	assert.NotEmpty(t, statusText)
	updated := rows.SetGitStatusText(fullPath, statusText)
	assert.True(t, updated)

	// Second call with same text => SetGitStatusText returns false
	fp.onGitStatus(status, rows, table, queueUpdateDraw, fullPath, false)
	assert.False(t, called)
}

func TestOnGitStatus_StaleRows(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	fp := nav.files

	repoDir := t.TempDir()
	dirContext := files.NewDirContext(nil, repoDir, nil)
	rows := NewFileRows(dirContext)
	entry := files.NewEntryWithDirPath(files.NewDirEntry("file.txt", false), repoDir)
	rows.AllEntries = []files.EntryWithDirPath{entry}
	rows.VisibleEntries = rows.AllEntries
	table := tview.NewTable()

	fullPath := filepath.Join(repoDir, "file.txt")
	status := &gitutils.RepoStatus{
		Branch:             "main",
		DirGitChangesStats: gitutils.DirGitChangesStats{FilesChanged: 1},
	}

	// Set fp.rows to different rows so the stale check triggers
	otherRows := NewFileRows(dirContext)
	fp.rows = otherRows

	tableContentSet := false
	queueUpdateDraw := func(f func()) {
		if f != nil {
			f()
		}
		tableContentSet = true
	}

	fp.onGitStatus(status, rows, table, queueUpdateDraw, fullPath, false)
	// queueUpdateDraw was called, but table.SetContent was skipped due to stale rows
	assert.True(t, tableContentSet)
}

func TestGetGitStatus_NilFromGetDirStatus(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)

	repoDir := t.TempDir()
	repo, err := git.PlainInit(repoDir, false)
	assert.NoError(t, err)

	oldGetDirStatus := getDirStatus
	defer func() { getDirStatus = oldGetDirStatus }()
	getDirStatus = func(_ context.Context, _ *git.Repository, _ string) *gitutils.RepoStatus {
		return nil
	}

	subdir := filepath.Join(repoDir, "sub")
	status := nav.getGitStatus(context.Background(), repo, subdir, true)
	assert.Nil(t, status)
}

func TestGetGitStatus_NilFromGetFileStatus(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)

	repoDir := t.TempDir()
	repo, err := git.PlainInit(repoDir, false)
	assert.NoError(t, err)

	oldGetFileStatus := getFileStatus
	defer func() { getFileStatus = oldGetFileStatus }()
	getFileStatus = func(_ context.Context, _ *git.Repository, _ string) *gitutils.RepoStatus {
		return nil
	}

	filePath := filepath.Join(repoDir, "file.txt")
	status := nav.getGitStatus(context.Background(), repo, filePath, false)
	assert.Nil(t, status)
}

func TestActivateFavorite_NilNavStore_Coverage(t *testing.T) {
	t.Parallel()
	// Test the exact guard: f.nav != nil but f.nav.store == nil
	nav, _, _ := newNavigatorForTest(t)
	nav.store = newMockStoreWithRoot(t, url.URL{Scheme: "file", Path: "/"})
	panel := newTestFavoritesPanel(nav)

	// Now set store to nil after creating the panel
	nav.store = nil
	fav := ftfav.Favorite{Store: url.URL{Scheme: "file"}, Path: "/tmp"}
	panel.activateFavorite(fav, true)
	panel.activateFavorite(fav, false)
	// Should return early without panic
}
