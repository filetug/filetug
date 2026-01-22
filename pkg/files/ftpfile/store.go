package ftpfile

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/jlaffaye/ftp"
)

var _ files.Store = (*Store)(nil)

type Store struct {
	root     url.URL
	factory  func(addr string, options ...ftp.DialOption) (FtpClient, error)
	explicit bool
	implicit bool
}

func (s *Store) RootURL() url.URL {
	root := s.root
	root.User = nil
	return root
}

func (s *Store) RootTitle() string {
	root := s.RootURL()
	sRoot := root.String()
	if strings.HasSuffix(sRoot, "/%C6%92") {
		return strings.TrimSuffix(sRoot, "/%C6%92")
	}
	return strings.TrimSuffix(sRoot, "/ƒ")
}

type FtpClient interface {
	Login(user, password string) error
	List(path string) (entries []*ftp.Entry, err error)
	Quit() error
}

type StoreOption func(*Store)

func NewStore(root url.URL, options ...StoreOption) *Store {
	if root.Scheme != "ftp" {
		panic(fmt.Errorf("schema should be 'ftp', got '%s'", root.Scheme))
	}
	store := &Store{
		root: root,
	}
	for _, opt := range options {
		opt(store)
	}
	return store
}

func WithFtpClientFactory(factory func(addr string, options ...ftp.DialOption) (FtpClient, error)) StoreOption {
	return func(s *Store) {
		s.factory = factory
	}
}

func (s *Store) SetTLS(explicit, implicit bool) {
	s.explicit = explicit
	s.implicit = implicit
}

func (s *Store) ReadDir(ctx context.Context, name string) ([]os.DirEntry, error) {
	root := s.root
	host := root.Hostname()
	if port := root.Port(); port == "" {
		root.Host = host + ":21"
	}
	addr := root.Host
	//addr := net.JoinHostPort(host, port)
	options := []ftp.DialOption{
		ftp.DialWithTimeout(5 * time.Second),
		ftp.DialWithContext(ctx),
	}
	if s.implicit {
		options = append(options, ftp.DialWithTLS(&tls.Config{ServerName: host, InsecureSkipVerify: true}))
	}
	if s.explicit {
		options = append(options, ftp.DialWithExplicitTLS(&tls.Config{ServerName: host, InsecureSkipVerify: true}))
	}

	type dialResult struct {
		c   FtpClient
		err error
	}

	dialChan := make(chan dialResult, 1)
	go func() {
		if s.factory != nil {
			c, err := s.factory(addr, options...)
			dialChan <- dialResult{c, err}
		} else {
			conn, err := ftp.Dial(addr, options...)
			if err != nil {
				dialChan <- dialResult{nil, err}
			} else {
				dialChan <- dialResult{conn, nil}
			}
		}
	}()

	var c FtpClient
	select {
	case res := <-dialChan:
		if res.err != nil {
			return nil, fmt.Errorf("failed to connect to ftp server: %w", res.err)
		}
		c = res.c
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	defer func() {
		_ = c.Quit()
	}()

	if root.User != nil {
		username := root.User.Username()
		password, hasPassword := root.User.Password()
		if !hasPassword {
			return nil, errors.New("missing password")
		}

		loginChan := make(chan error, 1)
		go func() {
			loginChan <- c.Login(username, password)
		}()

		select {
		case err := <-loginChan:
			if err != nil {
				return nil, fmt.Errorf("failed to login to ftp server: %w", err)
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	type listResult struct {
		entries []*ftp.Entry
		err     error
	}
	listChan := make(chan listResult, 1)
	go func() {
		entries, err := c.List(name)
		listChan <- listResult{entries, err}
	}()

	var entries []*ftp.Entry
	select {
	case res := <-listChan:
		if res.err != nil {
			return nil, fmt.Errorf("failed to list directory: %w", res.err)
		}
		entries = res.entries
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	result := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Name == "." || entry.Name == ".." {
			continue
		}
		dirEntry := files.NewDirEntry(
			entry.Name,
			entry.Type == ftp.EntryTypeFolder,
			files.Size(int64(entry.Size)),
			files.ModTime(entry.Time),
		)
		result = append(result, dirEntry)
	}

	return result, nil
}
