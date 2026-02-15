package ftfav

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setupFavoritesTestFile sets up a temporary favorites file for testing.
// Returns the path to the temporary file and a cleanup function.
func setupFavoritesTestFile(t *testing.T, filename string) (tempPath string, cleanup func()) {
	t.Helper()
	tempDir := t.TempDir()
	tempPath = filepath.Join(tempDir, filename)
	oldPath := favoritesFilePath
	favoritesFilePath = tempPath
	cleanup = func() {
		favoritesFilePath = oldPath
	}
	return tempPath, cleanup
}

func Test_GetFavorites_InvalidYaml(t *testing.T) {
	//t.Parallel()
	tempPath, cleanup := setupFavoritesTestFile(t, "favorites.yaml")
	defer cleanup()

	err := os.WriteFile(tempPath, []byte("invalid: ["), 0o644)
	assert.NoError(t, err)

	favorites, err := GetFavorites()
	assert.Nil(t, favorites)
	assert.Error(t, err)
}

func Test_GetFavorites_InvalidStoreURL(t *testing.T) {
	//t.Parallel()
	tempPath, cleanup := setupFavoritesTestFile(t, "favorites.yaml")
	defer cleanup()

	data := []byte("- store: \"http://[::1\"\n  path: /tmp\n")
	err := os.WriteFile(tempPath, data, 0o644)
	assert.NoError(t, err)

	favorites, err := GetFavorites()
	assert.Nil(t, favorites)
	assert.Error(t, err)
}

func Test_GetFavorites_FileNotExists(t *testing.T) {
	//t.Parallel()
	tempPath, cleanup := setupFavoritesTestFile(t, "missing.yaml")
	defer cleanup()

	favorites, err := GetFavorites()
	assert.NoError(t, err)
	assert.Len(t, favorites, 3)

	_, statErr := os.Stat(tempPath)
	assert.NoError(t, statErr)
	assert.Equal(t, "~/.filetug", favorites[0].Path)
	assert.Equal(t, "file", favorites[0].Store.Scheme)
	assert.Equal(t, "ftp", favorites[1].Store.Scheme)
	assert.Equal(t, "https", favorites[2].Store.Scheme)
}

func Test_GetFavorites_EmptyFile(t *testing.T) {
	//t.Parallel()
	tempPath, cleanup := setupFavoritesTestFile(t, "favorites.yaml")
	defer cleanup()

	err := os.WriteFile(tempPath, []byte(""), 0o644)
	assert.NoError(t, err)

	favorites, err := GetFavorites()
	assert.NoError(t, err)
	assert.Len(t, favorites, 0)
}

func Test_GetFavorites_FileExists_NoDefaults(t *testing.T) {
	//t.Parallel()
	tempPath, cleanup := setupFavoritesTestFile(t, "favorites.yaml")
	defer cleanup()

	expected := []Favorite{{Path: "/custom"}}
	err := writeFavorites(expected)
	assert.NoError(t, err)

	before, err := os.ReadFile(tempPath)
	assert.NoError(t, err)

	favorites, err := GetFavorites()
	assert.NoError(t, err)
	assert.Len(t, favorites, 1)
	assert.Equal(t, "/custom", favorites[0].Path)

	after, err := os.ReadFile(tempPath)
	assert.NoError(t, err)
	assert.Equal(t, before, after)
}

func Test_GetFavorites_ReplacesHomeDir(t *testing.T) {
	//t.Parallel()
	tempPath, cleanup := setupFavoritesTestFile(t, "favorites.yaml")
	defer cleanup()

	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err)

	data := []byte("- store: \"file://\"\n  path: \"" + filepath.Join(homeDir, "notes") + "\"\n- store: \"file://\"\n  path: \"" + homeDir + "\"\n")
	err = os.WriteFile(tempPath, data, 0o644)
	assert.NoError(t, err)

	favorites, err := GetFavorites()
	assert.NoError(t, err)
	assert.Len(t, favorites, 2)
	assert.Equal(t, filepath.Join("~", "notes"), favorites[0].Path)
	assert.Equal(t, "~", favorites[1].Path)
}

func Test_GetFavorites_DefaultWriteError(t *testing.T) {
	//t.Parallel()
	tempDir := t.TempDir()
	tempPath := filepath.Join(tempDir, "missing.yaml")
	oldPath := favoritesFilePath
	oldMarshal := yamlMarshal
	favoritesFilePath = tempPath
	defer func() {
		favoritesFilePath = oldPath
		yamlMarshal = oldMarshal
	}()

	yamlMarshal = func(in any) ([]byte, error) {
		_ = in
		return nil, assert.AnError
	}

	favorites, err := GetFavorites()
	assert.Nil(t, favorites)
	assert.Error(t, err)
	_, statErr := os.Stat(tempPath)
	assert.Error(t, statErr)
}

func Test_AddDelete_EmptyPath(t *testing.T) {
	t.Parallel()
	favoritesTestLock.Lock()
	t.Cleanup(favoritesTestLock.Unlock)
	oldPath := favoritesFilePath
	favoritesFilePath = ""
	defer func() {
		favoritesFilePath = oldPath
	}()

	addErr := AddFavorite(Favorite{Path: "/tmp"})
	assert.ErrorIs(t, addErr, errUserHomeDirIsUnknown)

	deleteErr := DeleteFavorite(Favorite{Path: "/tmp"})
	assert.ErrorIs(t, deleteErr, errUserHomeDirIsUnknown)
}

func Test_AddDelete_GetFavoritesError(t *testing.T) {
	//t.Parallel()
	tempDir := t.TempDir()
	oldPath := favoritesFilePath
	favoritesFilePath = tempDir
	defer func() {
		favoritesFilePath = oldPath
	}()

	addErr := AddFavorite(Favorite{Path: "/tmp"})
	assert.Error(t, addErr)

	deleteErr := DeleteFavorite(Favorite{Path: "/tmp"})
	assert.Error(t, deleteErr)
}

func Test_AddFavorite_ReplacesHomeDir(t *testing.T) {
	//t.Parallel()
	tempPath, cleanup := setupFavoritesTestFile(t, "favorites.yaml")
	defer cleanup()

	err := os.WriteFile(tempPath, []byte(""), 0o644)
	assert.NoError(t, err)

	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err)

	store := url.URL{Scheme: "file"}
	err = AddFavorite(Favorite{Store: store, Path: filepath.Join(homeDir, "docs")})
	assert.NoError(t, err)
	err = AddFavorite(Favorite{Store: store, Path: homeDir})
	assert.NoError(t, err)

	favorites, err := GetFavorites()
	assert.NoError(t, err)
	assert.Len(t, favorites, 2)
	assert.Equal(t, filepath.Join("~", "docs"), favorites[0].Path)
	assert.Equal(t, "~", favorites[1].Path)
}

func Test_WriteFavorites_MkdirError(t *testing.T) {
	//t.Parallel()
	tempDir := t.TempDir()
	parentFile := filepath.Join(tempDir, "parent")
	oldPath := favoritesFilePath
	favoritesFilePath = filepath.Join(parentFile, "favorites.yaml")
	defer func() {
		favoritesFilePath = oldPath
	}()

	err := os.WriteFile(parentFile, []byte("x"), 0o644)
	assert.NoError(t, err)

	writeErr := writeFavorites([]Favorite{{Path: "/tmp"}})
	assert.Error(t, writeErr)
}

func Test_WriteFavorites_MarshalError(t *testing.T) {
	//t.Parallel()
	oldPath := favoritesFilePath
	oldMarshal := yamlMarshal
	favoritesFilePath = filepath.Join(t.TempDir(), "favorites.yaml")
	defer func() {
		favoritesFilePath = oldPath
		yamlMarshal = oldMarshal
	}()

	yamlMarshal = func(in any) ([]byte, error) {
		_ = in
		return nil, assert.AnError
	}

	writeErr := writeFavorites([]Favorite{{Path: "/tmp"}})
	assert.Error(t, writeErr)
}

func Test_DeleteFavorite_KeepsOtherItems(t *testing.T) {
	//t.Parallel()
	tempPath, cleanup := setupFavoritesTestFile(t, "favorites.yaml")
	defer cleanup()

	err := os.WriteFile(tempPath, []byte(""), 0o644)
	assert.NoError(t, err)

	first := Favorite{Path: "/first"}
	second := Favorite{Path: "/second"}

	err = AddFavorite(first)
	assert.NoError(t, err)
	err = AddFavorite(second)
	assert.NoError(t, err)

	err = DeleteFavorite(first)
	assert.NoError(t, err)

	favorites, err := GetFavorites()
	assert.NoError(t, err)
	assert.Len(t, favorites, 1)
	assert.Equal(t, "/second", favorites[0].Path)
}
