package ftstate

import (
	"log"
	"os"
	"path/filepath"

	"github.com/datatug/filetug/pkg/fsutils"
)

const defaultSettingsDir = "~/.filetug"
const stateFileName = "filetug-state.json"

var settingsDir = defaultSettingsDir
var settingsDirPath = fsutils.ExpandHome(settingsDir)

type State struct {
	CurrentDir      string `json:"current_dir,omitempty"`
	CurrentFileName string `json:"current_file_name,omitempty"`
}

func getStateFilePath() string {
	return filepath.Join(settingsDirPath, stateFileName)
}

var logErr = log.Println

func GetCurrentDir() string {
	var state State
	filePath := getStateFilePath()
	_ = readJSON(filePath, false, &state)
	return state.CurrentDir
}

func SaveCurrentDir(currentDir string) {
	saveSettingValue(func(state *State) {
		state.CurrentDir = currentDir
	})
}

var readJSON = fsutils.ReadJSONFile
var writeJSON = fsutils.WriteJSONFile

func saveSettingValue(f func(state *State)) {
	filePath := getStateFilePath()
	var state State
	err := readJSON(filePath, false, &state)
	if err != nil {
		log.Println("SaveCurrentDir: Error reading state file:", err)
		return
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
	if err = writeJSON(filePath, state); err != nil {
		logErr("SaveCurrentDir: Error writing state file:", err)
		return
	}
}
