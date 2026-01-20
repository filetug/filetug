package httpfile

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func Test_NewStore(t *testing.T) {
	t.Run("https://example.com/pub/", func(t *testing.T) {
		root, _ := url.Parse("https://example.com/pub/")
		store := NewStore(*root)
		assert.NotNil(t, store)
	})

	t.Run("https://example.com/pub/", func(t *testing.T) {
		root, _ := url.Parse("https://example.com/pub/")
		store := NewStore(*root, WithHttpClient(&http.Client{}))
		assert.NotNil(t, store)
	})
}

type errorReader struct{}

func (e errorReader) Read(p []byte) (n int, err error) {
	_ = p
	return 0, fmt.Errorf("mock read error")
}

func (e errorReader) Close() error {
	return nil
}

func Test_httpFileStore_ReadDir(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	mockClient := &http.Client{
		Transport: &mockTransport{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				var body string
				switch req.URL.Path {
				case "/pub/":
					body = `<a href="linux/">linux/</a><a href="scm/">scm/</a><a href="tools/">tools/</a><a href="../">../</a><a href="/">/</a>`
				case "/pub/linux/":
					body = `<a href="kernel/">kernel/</a><a href="utils/">utils/</a>`
				case "/error/":
					return nil, fmt.Errorf("mock error")
				case "/read-error/":
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       errorReader{},
					}, nil
				default:
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(body)),
				}, nil
			},
		},
	}

	root, _ := url.Parse("https://cdn.kernel.org/")
	store := NewStore(*root, WithHttpClient(mockClient))

	t.Run("Root", func(t *testing.T) {
		entries, err := store.ReadDir(ctx, "/pub/")
		assert.NoError(t, err)
		assert.Equal(t, 3, len(entries))
		expectedNames := []string{"linux", "scm", "tools"}
		for _, name := range expectedNames {
			found := false
			for _, entry := range entries {
				if entry.Name() == name {
					found = true
					break
				}
			}
			assert.True(t, found, "expected to find %s in /pub/", name)
		}
	})

	t.Run("NoTrailingSlash", func(t *testing.T) {
		entries, err := store.ReadDir(ctx, "/pub")
		assert.NoError(t, err)
		assert.Equal(t, 3, len(entries))
	})

	t.Run("Error_Do", func(t *testing.T) {
		_, err := store.ReadDir(ctx, "/error/")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch directory listing")
	})

	t.Run("Error_ReadBody", func(t *testing.T) {
		_, err := store.ReadDir(ctx, "/read-error/")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read response body")
	})

	t.Run("Error_Status", func(t *testing.T) {
		_, err := store.ReadDir(ctx, "/notfound/")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status code: 404")
	})

	t.Run("Error_NewRequest", func(t *testing.T) {
		root := url.URL{Scheme: "http", Host: "example.com", Path: "/"}
		s := NewStore(root)
		var nilCtx context.Context = nil
		_, err := s.ReadDir(nilCtx, "/pub/") // Passing nil context should make NewRequestWithContext fail
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create request")
	})

	t.Run("NilClient", func(t *testing.T) {
		// This will actually try to make a real network request if we don't mock DefaultClient
		// or use a local server. For testing purpose, we can use a local server.
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintln(w, `<a href="file1">file1</a>`)
		}))
		defer ts.Close()

		u, _ := url.Parse(ts.URL)
		store2 := NewStore(*u)
		entries, err := store2.ReadDir(ctx, "/")
		assert.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, "file1", entries[0].Name())
	})
}

func TestHttpStore_RootURL(t *testing.T) {
	root, _ := url.Parse("https://example.com/pub/")
	store := NewStore(*root)
	assert.Equal(t, *root, store.RootURL())
}

func TestHttpStore_RootTitle(t *testing.T) {
	root, _ := url.Parse("https://user:pass@example.com/pub/")
	store := NewStore(*root)
	assert.Equal(t, "https://example.com/pub/", store.RootTitle())
}
