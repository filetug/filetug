package gitutils

import (
	"os"
	"path/filepath"
)

var OsStat = os.Stat

// GetRepositoryRoot check parent directories if this is a subdirectory of a repo
func GetRepositoryRoot(dirPath string) (repoRootDir string) {
	dirPath, err := filepath.Abs(dirPath)
	if err != nil {
		return ""
	}
	for {
		gitPath := filepath.Join(dirPath, ".git")
		if stat, err := OsStat(gitPath); err == nil {
			if stat.IsDir() {
				return dirPath
			}
		}
		parent := filepath.Dir(dirPath)
		if parent == dirPath {
			break
		}
		dirPath = parent
	}
	return ""
}
