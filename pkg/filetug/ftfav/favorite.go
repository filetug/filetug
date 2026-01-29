package ftfav

import (
	"errors"
	"net/url"
	"path/filepath"

	"github.com/filetug/filetug/pkg/filetug/ftsettings"
)

type Favorite struct {
	Store       url.URL `json:"store,omitempty" yaml:"store,omitempty"`
	Path        string  `json:"path" yaml:"path"`
	Shortcut    rune    `json:"shortcut,omitempty" yaml:"shortcut,omitempty"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
}

func (f Favorite) Key() string {
	key := f.Store
	key.Path = filepath.Join(key.Path, f.Path)
	return key.String()
}

const favoritesFileName = "datatug-favorites.yaml"

var favoritesFilePath string

var GetDatatugUserDir = ftsettings.GetDatatugUserDir

func init() {
	datatugUserDir, err := GetDatatugUserDir()
	if err == nil {
		favoritesFilePath = filepath.Join(datatugUserDir, favoritesFileName)
	}
}

var errUserHomeDirIsUnknown = errors.New("user home directory is unknown")

func GetFavorites() (favorites []Favorite, err error) {
	if favoritesFilePath == "" {
		return nil, errUserHomeDirIsUnknown
	}
	panic("TODO: implement")
}

func AddFavorite(f Favorite) (err error) {
	if favoritesFilePath == "" {
		return errUserHomeDirIsUnknown
	}
	panic("TODO: implement")
}

func DeleteFavorite(f Favorite) (err error) {
	if favoritesFilePath == "" {
		return errUserHomeDirIsUnknown
	}
	panic("TODO: implement")
}
