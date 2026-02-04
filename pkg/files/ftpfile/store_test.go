package ftpfile

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/jlaffaye/ftp"
	"github.com/stretchr/testify/assert"
)

type mockFtpClient struct {
	LoginFunc func(user, password string) error
	ListFunc  func(path string) ([]*ftp.Entry, error)
	QuitFunc  func() error
}

func (m *mockFtpClient) Login(user, password string) error {
	if m.LoginFunc != nil {
		return m.LoginFunc(user, password)
	}
	return nil
}

func (m *mockFtpClient) List(path string) ([]*ftp.Entry, error) {
	if m.ListFunc != nil {
		return m.ListFunc(path)
	}
	return nil, nil
}

func (m *mockFtpClient) Quit() error {
	if m.QuitFunc != nil {
		return m.QuitFunc()
	}
	return nil
}

func TestStore_RootURL_and_Title(t *testing.T) {
	t.Parallel()
	root, _ := url.Parse("ftp://user:pass@example.com/some/path/f")
	s := NewStore(*root)

	rootURL := s.RootURL()
	assert.Equal(t, "ftp://example.com/some/path/f", rootURL.String())
	assert.Nil(t, rootURL.User)

	rootTitle := s.RootTitle()
	assert.Equal(t, "ftp://example.com/some/path/f", rootTitle)

	// Test the suffix trimming
	rootWithSuffix, _ := url.Parse("ftp://example.com/path/ƒ")
	s2 := NewStore(*rootWithSuffix)
	assert.Equal(t, "ftp://example.com/path", s2.RootTitle())

	rootWithEncodedSuffix, _ := url.Parse("ftp://example.com/path/%C6%92")
	s3 := NewStore(*rootWithEncodedSuffix)
	assert.Equal(t, "ftp://example.com/path", s3.RootTitle())
}

func TestNewStore_InvalidScheme(t *testing.T) {
	t.Parallel()
	root, _ := url.Parse("http://example.com")
	store := NewStore(*root)
	assert.Nil(t, store)
}

func TestStore_SetTLS(t *testing.T) {
	t.Parallel()
	root, _ := url.Parse("ftp://example.com")
	s := NewStore(*root)
	if s == nil {
		t.Fatal("store should not be nil")
	}
	s.SetTLS(true, true)
	assert.True(t, s.explicit)
	assert.True(t, s.implicit)
}

func TestStore_ReadDir_Errors(t *testing.T) {
	t.Parallel()
	root, _ := url.Parse("ftp://user:pass@example.com/")

	t.Run("dial_error", func(t *testing.T) {
		factory := func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			return nil, fmt.Errorf("dial error")
		}
		s := NewStore(*root, WithFtpClientFactory(factory))
		_, err := s.ReadDir(context.Background(), ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to ftp server")
	})

	t.Run("missing_password", func(t *testing.T) {
		rootNoPass, _ := url.Parse("ftp://user@example.com/")
		s := NewStore(*rootNoPass, WithFtpClientFactory(func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			return &mockFtpClient{}, nil
		}))
		_, err := s.ReadDir(context.Background(), ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing password")
	})

	t.Run("login_error", func(t *testing.T) {
		mockClient := &mockFtpClient{
			LoginFunc: func(user, password string) error {
				return fmt.Errorf("login failed")
			},
		}
		factory := func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			return mockClient, nil
		}
		s := NewStore(*root, WithFtpClientFactory(factory))
		_, err := s.ReadDir(context.Background(), ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to login to ftp server")
	})

	t.Run("list_error", func(t *testing.T) {
		mockClient := &mockFtpClient{
			ListFunc: func(path string) ([]*ftp.Entry, error) {
				return nil, fmt.Errorf("list failed")
			},
		}
		factory := func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			return mockClient, nil
		}
		s := NewStore(*root, WithFtpClientFactory(factory))
		_, err := s.ReadDir(context.Background(), ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list directory")
	})

	t.Run("context_cancelled_dial", func(t *testing.T) {
		factory := func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			time.Sleep(100 * time.Millisecond)
			return &mockFtpClient{}, nil
		}
		s := NewStore(*root, WithFtpClientFactory(factory))
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		_, err := s.ReadDir(ctx, ".")
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("context_cancelled_login", func(t *testing.T) {
		mockClient := &mockFtpClient{
			LoginFunc: func(user, password string) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
		}
		factory := func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			return mockClient, nil
		}
		s := NewStore(*root, WithFtpClientFactory(factory))
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		_, err := s.ReadDir(ctx, ".")
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("context_cancelled_list", func(t *testing.T) {
		mockClient := &mockFtpClient{
			ListFunc: func(path string) ([]*ftp.Entry, error) {
				time.Sleep(100 * time.Millisecond)
				return nil, nil
			},
		}
		factory := func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			return mockClient, nil
		}
		s := NewStore(*root, WithFtpClientFactory(factory))
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		_, err := s.ReadDir(ctx, ".")
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("dot_and_dotdot_skipped", func(t *testing.T) {
		mockClient := &mockFtpClient{
			ListFunc: func(path string) ([]*ftp.Entry, error) {
				return []*ftp.Entry{
					{Name: ".", Type: ftp.EntryTypeFolder},
					{Name: "..", Type: ftp.EntryTypeFolder},
					{Name: "realfile.txt", Type: ftp.EntryTypeFile},
				}, nil
			},
		}
		factory := func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			return mockClient, nil
		}
		s := NewStore(*root, WithFtpClientFactory(factory))
		entries, err := s.ReadDir(context.Background(), ".")
		assert.NoError(t, err)
		assert.Equal(t, 1, len(entries))
		assert.Equal(t, "realfile.txt", entries[0].Name())
	})

	// Commented out as hanging. Needs proper testing that will perform quick
	//t.Run("real_dial_success", func(t *testing.T) {
	//	// We can't easily have a real FTP server here without external dependencies,
	//	// but we can try to dial something that exists but isn't an FTP server.
	//	// It will fail, but it will cover the ftp.Dial(addr, options...) line.
	//	rootInvalid, _ := url.Parse("ftp://google.com:80")
	//	s := NewStore(*rootInvalid)
	//	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	//	defer cancel()
	//	_, err := s.ReadDir(ctx, ".")
	//	assert.Error(t, err)
	//	// It could be a context deadline exceeded OR a connection error depending on environment
	//})

	t.Run("real_dial_error", func(t *testing.T) {
		origFtpDial := ftpDial
		defer func() { ftpDial = origFtpDial }()
		ftpDial = func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			return nil, fmt.Errorf("dial error")
		}

		// Use a port that is likely not used to trigger an error in ftp.Dial
		rootInvalid, _ := url.Parse("ftp://localhost:1")
		s := NewStore(*rootInvalid) // No factory
		_, err := s.ReadDir(context.Background(), ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to ftp server")
	})

	t.Run("real_dial_success_mock", func(t *testing.T) {
		origFtpDial := ftpDial
		defer func() { ftpDial = origFtpDial }()
		ftpDial = func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			return &mockFtpClient{}, nil
		}

		root, _ := url.Parse("ftp://example.com/")
		s := NewStore(*root) // No factory
		_, err := s.ReadDir(context.Background(), ".")
		assert.NoError(t, err)
	})

	t.Run("default_port_assignment", func(t *testing.T) {
		dialedAddr := ""
		factory := func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			dialedAddr = addr
			return &mockFtpClient{}, nil
		}
		rootNoPort, _ := url.Parse("ftp://example.com/")
		s := NewStore(*rootNoPort, WithFtpClientFactory(factory))
		_, _ = s.ReadDir(context.Background(), ".")
		assert.Equal(t, "example.com:21", dialedAddr)
	})

	t.Run("with_port_assignment", func(t *testing.T) {
		dialedAddr := ""
		factory := func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			dialedAddr = addr
			return &mockFtpClient{}, nil
		}
		rootWithPort, _ := url.Parse("ftp://example.com:2121/")
		s := NewStore(*rootWithPort, WithFtpClientFactory(factory))
		_, _ = s.ReadDir(context.Background(), ".")
		assert.Equal(t, "example.com:2121", dialedAddr)
	})
}

func TestStore_Create_Delete_NotImplemented(t *testing.T) {
	t.Parallel()
	root, _ := url.Parse("ftp://example.com")
	s := NewStore(*root)
	ctx := context.Background()

	assert.Error(t, s.Delete(ctx, "/path"))
	assert.Error(t, s.CreateDir(ctx, "/path"))
	assert.Error(t, s.CreateFile(ctx, "/path"))
	_, err := s.GetDirReader(ctx, "/path")
	assert.ErrorIs(t, err, files.ErrNotSupported)
}

func TestStore_ReadDir_TLS_Options(t *testing.T) {
	t.Parallel()
	root, _ := url.Parse("ftp://example.com")

	t.Run("explicit_tls", func(t *testing.T) {
		dialed := false
		factory := func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			dialed = true
			// We can't easily inspect options because they are functions,
			// but we can at least ensure it doesn't crash and we cover the branches.
			return &mockFtpClient{}, nil
		}
		s := NewStore(*root, WithFtpClientFactory(factory))
		s.SetTLS(true, false)
		_, _ = s.ReadDir(context.Background(), ".")
		assert.True(t, dialed)
	})

	t.Run("implicit_tls", func(t *testing.T) {
		dialed := false
		factory := func(addr string, options ...ftp.DialOption) (FtpClient, error) {
			dialed = true
			return &mockFtpClient{}, nil
		}
		s := NewStore(*root, WithFtpClientFactory(factory))
		s.SetTLS(false, true)
		_, _ = s.ReadDir(context.Background(), ".")
		assert.True(t, dialed)
	})
}

func TestStore_ReadDir_Mock(t *testing.T) {
	t.Parallel()
	root, _ := url.Parse("ftp://demo:password@example.com/")
	mockClient := &mockFtpClient{
		ListFunc: func(path string) ([]*ftp.Entry, error) {
			return []*ftp.Entry{
				{Name: "file1.txt", Type: ftp.EntryTypeFile, Size: 100},
				{Name: "dir1", Type: ftp.EntryTypeFolder},
			}, nil
		},
	}

	factory := func(addr string, options ...ftp.DialOption) (FtpClient, error) {
		return mockClient, nil
	}

	s := NewStore(*root, WithFtpClientFactory(factory))
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	entries, err := s.ReadDir(ctx, ".")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(entries))
	assert.Equal(t, "file1.txt", entries[0].Name())
	assert.False(t, entries[0].IsDir())
	assert.Equal(t, "dir1", entries[1].Name())
	assert.True(t, entries[1].IsDir())
}

func TestFtpDial_Default(t *testing.T) {
	t.Parallel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skipping: failed to bind listener: %v", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	done := make(chan struct{})
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			close(done)
			return
		}
		defer func() {
			_ = conn.Close()
			close(done)
		}()
		_, _ = fmt.Fprint(conn, "220 test server\r\n")
		reader := bufio.NewReader(conn)
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				return
			}
			if strings.HasPrefix(line, "QUIT") {
				_, _ = fmt.Fprint(conn, "221 bye\r\n")
				return
			}
		}
	}()

	options := []ftp.DialOption{
		ftp.DialWithTimeout(1 * time.Second),
	}
	addr := listener.Addr().String()
	client, err := ftpDial(addr, options...)
	assert.NoError(t, err)
	if err == nil {
		quitErr := client.Quit()
		assert.NoError(t, quitErr)
	}
	<-done
}

func TestStore_ReadDir(t *testing.T) {
	//t.Parallel()
	if os.Getenv("RUN_FTP_INTEGRATION_TESTS") != "true" {
		t.Skip("skipping integration test; set RUN_FTP_INTEGRATION_TESTS=true to run")
	}
	const host = "test.rebex.net"
	const port = 21
	root := url.URL{
		Scheme: "ftp",
		Host:   fmt.Sprintf("%s:%d", host, port),
		User:   url.UserPassword("demo", "password"),
	}
	t.Run("host_with_port", func(t *testing.T) {
		root := root
		root.Host = fmt.Sprintf("%s:%d", host, port)
		s := NewStore(root)
		testReadDir(t, s)
	})

	t.Run("plain_default_port", func(t *testing.T) {
		root := root
		root.Host = host
		s := NewStore(root)
		testReadDir(t, s)
	})

	t.Run("explicit_TLS", func(t *testing.T) {
		t.Skip("test.rebex.net requires TLS session resumption which github.com/jlaffaye/ftp might not support or needs more config")
		s := NewStore(root)
		s.SetTLS(true, false)
		testReadDir(t, s)
	})

	t.Run("implicit_TLS", func(t *testing.T) {
		t.Skip("test.rebex.net requires TLS session resumption which github.com/jlaffaye/ftp might not support or needs more config")
		s := NewStore(root)
		s.SetTLS(false, true)
		testReadDir(t, s)
	})
}

func testReadDir(t *testing.T, s *Store) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	entries, err := s.ReadDir(ctx, ".")
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}

	if len(entries) == 0 {
		t.Error("expected at least one entry, got 0")
	}

	for _, entry := range entries {
		t.Logf("Entry: %s, IsDir: %v", entry.Name(), entry.IsDir())
	}
}
