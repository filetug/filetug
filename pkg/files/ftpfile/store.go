package ftpfile

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/datatug/filetug/pkg/files"
	"github.com/jlaffaye/ftp"
)

var _ files.Store = (*Store)(nil)

type Store struct {
	root     url.URL
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
	return root.String()
}

func NewStore(root url.URL) *Store {
	if root.Scheme != "ftp" {
		panic(fmt.Errorf("schema should be 'ftp', got '%s'", root.Scheme))
	}
	store := &Store{
		root: root,
	}
	return store
}

func (s *Store) SetTLS(explicit, implicit bool) {
	s.explicit = explicit
	s.implicit = implicit
}

func (s *Store) ReadDir(name string) ([]os.DirEntry, error) {
	root := s.root
	host := root.Hostname()
	if port := root.Port(); port == "" {
		root.Host = host + ":21"
	}
	addr := root.Host
	//addr := net.JoinHostPort(host, port)
	options := []ftp.DialOption{
		ftp.DialWithTimeout(5 * time.Second),
	}
	if s.implicit {
		options = append(options, ftp.DialWithTLS(&tls.Config{ServerName: host, InsecureSkipVerify: true}))
	}
	if s.explicit {
		options = append(options, ftp.DialWithExplicitTLS(&tls.Config{ServerName: host, InsecureSkipVerify: true}))
	}

	c, err := ftp.Dial(addr, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ftp server: %w", err)
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
		err = c.Login(username, password)
		if err != nil {
			return nil, fmt.Errorf("failed to login to ftp server: %w", err)
		}
	}

	entries, err := c.List(name)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
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
