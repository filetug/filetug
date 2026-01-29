package ftfav

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/filetug/filetug/pkg/filetug/ftsettings"
	"gopkg.in/yaml.v3"
)

type favorite struct {
	Store       string `yaml:"store"`
	Path        string `yaml:"path"`
	Shortcut    rune   `yaml:"shortcut,omitempty"`
	Description string `yaml:"description,omitempty"`
}

const favoritesFileName = "datatug-favorites.yaml"

var favoritesFilePath string

var GetDatatugUserDir = ftsettings.GetDatatugUserDir
var yamlMarshal = yaml.Marshal
var yamlUnmarshal = yaml.Unmarshal
var parseURL = url.Parse

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
	data, err := os.ReadFile(favoritesFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			defaults := defaultFavorites()
			writeErr := writeFavorites(defaults)
			if writeErr != nil {
				return nil, writeErr
			}
			return defaults, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return []Favorite{}, nil
	}
	var persisted []favorite
	err = yamlUnmarshal(data, &persisted)
	if err != nil {
		return nil, err
	}
	favorites = make([]Favorite, 0, len(persisted))
	var (
		homeDir        string
		homeErr        error
		homeDirChecked bool
	)
	for _, item := range persisted {
		mapped, mapErr := mapFavorite(item)
		if mapErr != nil {
			return nil, mapErr
		}
		if mapped.Store.Scheme == "file" && mapped.Store.Path == "" && mapped.Path != "" {
			if !homeDirChecked {
				homeDir, homeErr = os.UserHomeDir()
				homeDirChecked = true
			}
			if homeErr == nil && homeDir != "" {
				cleanHome := filepath.Clean(homeDir)
				cleanPath := filepath.Clean(mapped.Path)
				if cleanPath == cleanHome {
					mapped.Path = "~"
				} else {
					homePrefix := cleanHome + string(filepath.Separator)
					if strings.HasPrefix(cleanPath, homePrefix) {
						relative := strings.TrimPrefix(cleanPath, homePrefix)
						mapped.Path = filepath.Join("~", relative)
					}
				}
			}
		}
		favorites = append(favorites, mapped)
	}
	return favorites, nil
}

func AddFavorite(f Favorite) (err error) {
	if favoritesFilePath == "" {
		return errUserHomeDirIsUnknown
	}
	if f.Store.Scheme == "file" && f.Path != "" {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr == nil && homeDir != "" {
			cleanHome := filepath.Clean(homeDir)
			cleanPath := filepath.Clean(f.Path)
			if cleanPath == cleanHome {
				f.Path = "~"
			} else {
				homePrefix := cleanHome + string(filepath.Separator)
				if strings.HasPrefix(cleanPath, homePrefix) {
					relative := strings.TrimPrefix(cleanPath, homePrefix)
					f.Path = filepath.Join("~", relative)
				}
			}
		}
	}
	favorites, err := GetFavorites()
	if err != nil {
		return err
	}
	favorites = append(favorites, f)
	return writeFavorites(favorites)
}

func DeleteFavorite(f Favorite) (err error) {
	if favoritesFilePath == "" {
		return errUserHomeDirIsUnknown
	}
	favorites, err := GetFavorites()
	if err != nil {
		return err
	}
	deleteKey := f.Key()
	updated := make([]Favorite, 0, len(favorites))
	for _, item := range favorites {
		if item.Key() == deleteKey {
			continue
		}
		updated = append(updated, item)
	}
	return writeFavorites(updated)
}

func writeFavorites(favorites []Favorite) error {
	persisted := make([]favorite, 0, len(favorites))
	for _, item := range favorites {
		mapped := mapFavoriteToPersisted(item)
		persisted = append(persisted, mapped)
	}
	data, err := yamlMarshal(persisted)
	if err != nil {
		return err
	}
	dir := filepath.Dir(favoritesFilePath)
	err = os.MkdirAll(dir, 0o755)
	if err != nil {
		return err
	}
	return os.WriteFile(favoritesFilePath, data, 0o644)
}

func mapFavorite(item favorite) (Favorite, error) {
	var store url.URL
	if item.Store != "" {
		parsed, err := url.Parse(item.Store)
		if err != nil {
			return Favorite{}, err
		}
		store = *parsed
	}
	return Favorite{
		Store:       store,
		Path:        item.Path,
		Shortcut:    item.Shortcut,
		Description: item.Description,
	}, nil
}

func mapFavoriteToPersisted(item Favorite) favorite {
	store := item.Store.String()
	return favorite{
		Store:       store,
		Path:        item.Path,
		Shortcut:    item.Shortcut,
		Description: item.Description,
	}
}

func defaultFavorites() []Favorite {
	ftpURL, _ := parseURL("ftp://demo:password@test.rebex.net")
	httpsURL, _ := parseURL("https://cdn.kernel.org/pub/")
	defaults := []Favorite{
		{
			Store:       url.URL{Scheme: "file"},
			Path:        "~/.filetug",
			Description: "FileTug settings dir",
		},
		{
			Store: *ftpURL,
		},
		{
			Store: *httpsURL,
			Path:  httpsURL.Path,
		},
	}
	return defaults
}
