package gitutils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/stretchr/testify/assert"
)

func TestGetGlobalExcludesFile_ExpandsHome(t *testing.T) {
	origExecCommand := execCommand
	origHomeDir := osUserHomeDir

	defer func() {
		execCommand = origExecCommand
		osUserHomeDir = origHomeDir
	}()

	execCommand = fakeExecCommand("~/.config/gitignore\n", 0)
	osUserHomeDir = func() (string, error) {
		return "/home/tester", nil
	}

	path, ok := getGlobalExcludesFile("/repo")
	assert.True(t, ok)
	assert.Equal(t, "/home/tester/.config/gitignore", path)
}

func TestLoadGlobalIgnorePatterns_UsesExcludesFile(t *testing.T) {
	origExecCommand := execCommand
	origReadFile := osReadFile

	defer func() {
		execCommand = origExecCommand
		osReadFile = origReadFile
	}()

	execCommand = fakeExecCommand("/tmp/global_ignore\n", 0)
	osReadFile = func(_ string) ([]byte, error) {
		return []byte("*.log\n"), nil
	}

	patterns := loadGlobalIgnorePatterns("/repo")
	matcher := gitignore.NewMatcher(patterns)
	parts := strings.Split("app.log", "/")
	ignored := matcher.Match(parts, false)
	assert.True(t, ignored)
}

func TestLoadGlobalIgnorePatterns_DefaultsWhenGitFails(t *testing.T) {
	origExecCommand := execCommand

	defer func() {
		execCommand = origExecCommand
	}()

	execCommand = fakeExecCommand("", 1)

	patterns := loadGlobalIgnorePatterns("/repo")
	matcher := gitignore.NewMatcher(patterns)
	parts := strings.Split(".DS_Store", "/")
	ignored := matcher.Match(parts, false)
	assert.True(t, ignored)
}

func TestParseIgnorePatterns(t *testing.T) {
	content := []byte("# comment\n*.log\r\n\n.DS_Store\n")
	patterns := parseIgnorePatterns(content)
	matcher := gitignore.NewMatcher(patterns)
	logParts := strings.Split("app.log", "/")
	dsParts := strings.Split(".DS_Store", "/")
	assert.True(t, matcher.Match(logParts, false))
	assert.True(t, matcher.Match(dsParts, false))
}

func TestGetDirStatus_IgnoresDefaultDSStore(t *testing.T) {
	origExecCommand := execCommand

	defer func() {
		execCommand = origExecCommand
	}()

	execCommand = fakeExecCommand("", 1)

	tempDir := t.TempDir()
	repo, err := git.PlainInit(tempDir, false)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, ".DS_Store"), []byte("junk\n"), 0o644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("content\n"), 0o644)
	assert.NoError(t, err)

	status := GetDirStatus(context.Background(), repo, tempDir)
	assert.NotNil(t, status)
	assert.Equal(t, 1, status.FilesChanged)
}

func fakeExecCommand(output string, exitCode int) func(string, ...string) *exec.Cmd {
	return func(_ string, _ ...string) *exec.Cmd {
		args := []string{"-test.run=TestHelperProcess", "--", output}
		cmd := exec.Command(os.Args[0], args...)
		env := os.Environ()
		env = append(env, "GO_WANT_HELPER_PROCESS=1")
		exitCodeValue := strconv.Itoa(exitCode)
		exitEnv := "HELPER_EXIT_CODE=" + exitCodeValue
		env = append(env, exitEnv)
		cmd.Env = env
		return cmd
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	separatorIndex := -1
	for i, arg := range args {
		if arg == "--" {
			separatorIndex = i
			break
		}
	}

	output := ""
	if separatorIndex >= 0 && separatorIndex+1 < len(args) {
		output = args[separatorIndex+1]
	}
	_, _ = fmt.Fprint(os.Stdout, output)

	exitCode := 0
	exitEnv := os.Getenv("HELPER_EXIT_CODE")
	if exitEnv != "" {
		parsed, err := strconv.Atoi(exitEnv)
		if err == nil {
			exitCode = parsed
		} else {
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}
