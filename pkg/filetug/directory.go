package filetug

import "github.com/filetug/filetug/pkg/gitutils"

type DirInfo struct {
	Git *DirGitInfo
}

type DirGitInfo struct {
	Repo *gitutils.RepoStatus
}
