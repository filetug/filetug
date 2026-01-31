package ftfav

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFavoriteKey(t *testing.T) {
	t.Parallel()
	store := url.URL{Scheme: "file", Path: "/base"}
	fav := Favorite{Store: store, Path: "docs"}
	keyURL := store
	keyURL.Path = filepath.Join(keyURL.Path, fav.Path)
	expected := keyURL.String()
	actual := fav.Key()
	assert.Equal(t, expected, actual)
}

func TestFavorites_FileOperations(t *testing.T) {
	//t.Parallel()
	tempDir := t.TempDir()
	tempPath := filepath.Join(tempDir, "favorites.yaml")
	oldPath := favoritesFilePath
	favoritesFilePath = tempPath
	defer func() {
		favoritesFilePath = oldPath
	}()

	err := os.WriteFile(tempPath, []byte(""), 0o644)
	assert.NoError(t, err)

	favorites, err := GetFavorites()
	assert.NoError(t, err)
	favoritesCount := len(favorites)
	assert.Equal(t, 0, favoritesCount)

	store := url.URL{Scheme: "file", Path: "/base"}
	fav1 := Favorite{Store: store, Path: "dir", Description: "first"}
	fav2 := Favorite{Store: store, Path: "dir", Description: "second"}

	err = AddFavorite(fav1)
	assert.NoError(t, err)
	err = AddFavorite(fav2)
	assert.NoError(t, err)

	favorites, err = GetFavorites()
	assert.NoError(t, err)
	favoritesCount = len(favorites)
	assert.Equal(t, 2, favoritesCount)

	err = DeleteFavorite(fav1)
	assert.NoError(t, err)

	favorites, err = GetFavorites()
	assert.NoError(t, err)
	favoritesCount = len(favorites)
	assert.Equal(t, 0, favoritesCount)
}

func TestFavorites_EmptyPathError(t *testing.T) {
	t.Parallel()
	oldPath := favoritesFilePath
	favoritesFilePath = ""
	defer func() {
		favoritesFilePath = oldPath
	}()

	favorites, err := GetFavorites()
	_ = favorites
	assert.ErrorIs(t, err, errUserHomeDirIsUnknown)
}
