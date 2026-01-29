package httpfile

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/filetug/filetug/pkg/files"
)

type StoreOption func(*HttpStore)

func NewStore(root url.URL, o ...StoreOption) *HttpStore {
	store := &HttpStore{
		Root: root,
	}
	for _, opt := range o {
		opt(store)
	}
	return store
}

func WithHttpClient(client *http.Client) StoreOption {
	return func(store *HttpStore) {
		store.client = client
	}
}

var _ files.Store = (*HttpStore)(nil)

type HttpStore struct {
	Root   url.URL
	client *http.Client
}

func (h HttpStore) Delete(ctx context.Context, path string) error {
	_, _ = ctx, path
	return files.ErrNotImplemented
}

func (h HttpStore) RootURL() url.URL {
	return h.Root
}

func (h HttpStore) RootTitle() string {
	root := h.Root
	root.User = nil
	return root.String()
}

func (h HttpStore) ReadDir(ctx context.Context, name string) ([]os.DirEntry, error) {
	u := h.Root
	u.Path = name
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}

	client := h.client
	if client == nil {
		client = http.DefaultClient
	}

	reqURL := u.String()
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
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
	bodyText := string(body)
	matches := re.FindAllStringSubmatch(bodyText, -1)

	var entries []os.DirEntry
	for _, match := range matches {
		href := match[1]
		if href == "../" || href == "/" {
			continue
		}
		isDir := strings.HasSuffix(href, "/")
		entryName := strings.TrimSuffix(href, "/")
		entryName = strings.TrimPrefix(entryName, "/ƒ")
		dirEntry := files.NewDirEntry(entryName, isDir)
		entries = append(entries, dirEntry)
	}

	return entries, nil
}

func (h HttpStore) CreateDir(ctx context.Context, path string) error {
	_, _ = ctx, path
	return fmt.Errorf("CreateDir not implemented for HTTP")
}

func (h HttpStore) CreateFile(ctx context.Context, path string) error {
	_, _ = ctx, path
	return fmt.Errorf("CreateFile not implemented for HTTP")
}
