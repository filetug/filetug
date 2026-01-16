package ftstate

import (
	"os"
	"path/filepath"

	"github.com/datatug/filetug/pkg/fsutils"
)

const defaultSettingsDir = "~/.filetug"
const stateFileName = "filetug-state.json"

var settingsDir = defaultSettingsDir
var settingsDirPath = fsutils.ExpandHome(settingsDir)

type State struct {
	Store           string `json:"store,omitempty"`
	CurrentDir      string `json:"current_dir,omitempty"`
	SelectedTreeDir string `json:"selected_tree_dir,omitempty"`
	CurrentFileName string `json:"current_file_name,omitempty"`
}

func getStateFilePath() string {
	return filepath.Join(settingsDirPath, stateFileName)
}

var logErr = func(v ...any) {

}

func GetState() (*State, error) {
	filePath := getStateFilePath()
	var state State
	return &state, readJSON(filePath, false, &state)
}

func GetCurrentDir() string {
	var state State
	filePath := getStateFilePath()
	_ = readJSON(filePath, false, &state)
	return state.CurrentDir
}

func SaveCurrentDir(store, currentDir string) {
	saveSettingValue(func(state *State) {
		state.Store = store
		state.CurrentDir = currentDir
	})
}

func SaveSelectedTreeDir(dir string) {
	saveSettingValue(func(state *State) {
		state.SelectedTreeDir = dir
	})
}

func SaveCurrentFileName(name string) {
	saveSettingValue(func(state *State) {
		state.CurrentFileName = name
	})
}

var readJSON = fsutils.ReadJSONFile
var writeJSON = fsutils.WriteJSONFile

func saveSettingValue(f func(state *State)) {
	filePath := getStateFilePath()
	var state State
	err := readJSON(filePath, false, &state)
	if err != nil {
		logErr("SaveCurrentDir: Error reading state file:", err)
	}

	if dirInfo, err := os.Stat(settingsDirPath); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(settingsDirPath, os.ModePerm); err != nil {
				logErr("SaveCurrentDir: Error creating settings directory:", err)
				return
			}
		}
	} else if !dirInfo.IsDir() {
		logErr("SaveCurrentDir: State file is not a directory")
		return
	}

	f(&state)
	if err := writeJSON(filePath, state); err != nil {
		logErr("SaveCurrentDir: Error writing state file:", err)
		return
	}
}
