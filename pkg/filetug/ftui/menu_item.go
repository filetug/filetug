package ftui

type MenuItem struct {
	Title      string
	HotKeys    []string
	Action     func()
	IsAltHotkey bool
}
