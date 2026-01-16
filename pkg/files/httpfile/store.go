package httpfile

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/datatug/filetug/pkg/files"
)

func NewStore(root url.URL) *HttpStore {
	return &HttpStore{Root: root}
}

var _ files.Store = (*HttpStore)(nil)

type HttpStore struct {
	Root url.URL
}

func (h HttpStore) RootURL() url.URL {
	return h.Root
}

func (h HttpStore) RootTitle() string {
	root := h.Root
	root.Path = ""
	root.User = nil
	return root.String()
}

func (h HttpStore) ReadDir(name string) ([]os.DirEntry, error) {
	u := h.Root
	u.Path = name
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch directory listing: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	re := regexp.MustCompile(`<a href="([^"]+)">`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	var entries []os.DirEntry
	for _, match := range matches {
		href := match[1]
		if href == "../" || href == "/" {
			continue
		}
		isDir := strings.HasSuffix(href, "/")
		name := strings.TrimSuffix(href, "/")
		entries = append(entries, httpDirEntry{name: name, isDir: isDir})
	}

	return entries, nil
}
