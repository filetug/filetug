package ftpfile

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/datatug/filetug/pkg/files"
	"github.com/jlaffaye/ftp"
)

const schema = "ftp"

var _ files.Store = (*Store)(nil)

type Store struct {
	host     string
	path     string
	user     string
	password string
	explicit bool
	implicit bool
}

func (s *Store) RootURL() url.URL {
	u := url.URL{
		Scheme: schema,
		Host:   s.host,
		Path:   s.path,
	}
	return u
}

func (s *Store) RootTitle() string {
	return schema + "://" + s.host
}

func NewStore(addr, user, password string) *Store {
	return &Store{
		host:     addr,
		user:     user,
		password: password,
	}
}

func (s *Store) SetTLS(explicit, implicit bool) {
	s.explicit = explicit
	s.implicit = implicit
}

func (s *Store) ReadDir(name string) ([]os.DirEntry, error) {
	host, port, err := net.SplitHostPort(s.host)
	if err != nil {
		host = s.host
		port = "21"
	}
	addr := net.JoinHostPort(host, port)
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

	if s.user != "" {
		err = c.Login(s.user, s.password)
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
		result = append(result, &ftpDirEntry{entry: entry})
	}

	return result, nil
}

type ftpDirEntry struct {
	entry *ftp.Entry
}

func (e *ftpDirEntry) Name() string {
	return e.entry.Name
}

func (e *ftpDirEntry) IsDir() bool {
	return e.entry.Type == ftp.EntryTypeFolder
}

func (e *ftpDirEntry) Type() os.FileMode {
	if e.IsDir() {
		return os.ModeDir
	}
	return 0
}

func (e *ftpDirEntry) Info() (os.FileInfo, error) {
	return &ftpFileInfo{entry: e.entry}, nil
}

type ftpFileInfo struct {
	entry *ftp.Entry
}

func (f *ftpFileInfo) Name() string       { return f.entry.Name }
func (f *ftpFileInfo) Size() int64        { return int64(f.entry.Size) }
func (f *ftpFileInfo) Mode() os.FileMode  { return (&ftpDirEntry{entry: f.entry}).Type() }
func (f *ftpFileInfo) ModTime() time.Time { return f.entry.Time }
func (f *ftpFileInfo) IsDir() bool        { return f.entry.Type == ftp.EntryTypeFolder }
func (f *ftpFileInfo) Sys() any           { return f.entry }
