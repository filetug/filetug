package viewers

import (
	"log"
	"os"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/filetug/filetug/pkg/filetug/navigator"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/strongo/strongo-tui/pkg/colors"
)

var _ Previewer = (*DirPreviewer)(nil)

type DirPreviewer struct {
	*sneatv.Boxed
	flex *tview.Flex
	tabs *sneatv.Tabs

	ExtTable     *tview.Table
	GitPreviewer *GitDirStatusPreviewer

	app DirPreviewerApp

	dirPath string

	tableMu sync.Mutex

	extByID  map[string]*ExtStat
	ExtStats []*ExtStat

	extGroupsByID map[string]*ExtensionsGroup
	ExtGroups     []*ExtensionsGroup

	setFilter       func(ftui.Filter)
	focusLeft       func()
	queueUpdateDraw navigator.UpdateDrawQueuer
	colorByExt      func(string) tcell.Color
}

func NewDirPreviewer(app DirPreviewerApp, options ...DirSummaryOption) *DirPreviewer {
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.SetTitle("Dir Summary")

	extTable := tview.NewTable()
	extTable.SetSelectable(true, false)

	d := &DirPreviewer{
		app:      app,
		flex:     flex,
		ExtTable: extTable,
	}
	d.Boxed = sneatv.NewBoxed(
		flex,
		sneatv.WithLeftBorder(0, -1),
	)
	d.colorByExt = func(_ string) tcell.Color { return colors.TableHeaderColor }
	d.GitPreviewer = NewGitDirStatusPreviewer()
	d.setTabs(false)

	selectedStyle := tcell.StyleDefault
	selectedStyle = selectedStyle.Foreground(tcell.ColorBlack)
	selectedStyle = selectedStyle.Background(colors.TableHeaderColor)
	d.ExtTable.SetSelectedStyle(selectedStyle)

	d.ExtTable.SetInputCapture(d.inputCapture)
	d.ExtTable.SetSelectionChangedFunc(d.selectionChanged)

	for _, option := range options {
		option(d)
	}

	return d
}

func (d *DirPreviewer) PreviewSingle(entry files.EntryWithDirPath, _ []byte, _ error) {
	dirContext, ok := entry.(*files.DirContext)
	if ok {
		d.SetDirEntries(dirContext)
		return
	}
	dirPath := entry.DirPath()
	if entry.IsDir() {
		dirPath = entry.FullName()
	}
	fallbackContext := files.NewDirContext(nil, dirPath, nil)
	d.SetDirEntries(fallbackContext)
}

func (d *DirPreviewer) Main() tview.Primitive {
	return d
}

func (d *DirPreviewer) Meta() tview.Primitive {
	return nil
}

func (d *DirPreviewer) Focus(delegate func(p tview.Primitive)) {
	d.ExtTable.Focus(delegate)
}

func (d *DirPreviewer) SetDirEntries(dirContext *files.DirContext) {
	var dirPath string
	var entries []os.DirEntry
	if dirContext != nil {
		dirPath = dirContext.Path()
		entries = dirContext.Children()
	}
	d.dirPath = dirPath

	extByID := make(map[string]*ExtStat)
	extStats := make([]*ExtStat, 0)
	extGroupsByID := make(map[string]*ExtensionsGroup)
	extGroups := make([]*ExtensionsGroup, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		extID := path.Ext(name)
		if extID == name {
			continue
		}
		ext, ok := extByID[extID]
		if !ok {
			ext = &ExtStat{
				ID: extID,
			}
			extByID[extID] = ext
			extStats = append(extStats, ext)
		}
		ext.entries = append(ext.entries, entry)
		ext.Count++

		groupID := fileExtTypes[extID]
		if groupID == "" {
			groupID = otherExtensionsGroupID
		}
		extGroup, existingExtGroup := extGroupsByID[groupID]

		if !existingExtGroup {
			extGroup = &ExtensionsGroup{
				ID:         groupID,
				Title:      fileExtPlurals[groupID],
				GroupStats: new(GroupStats),
			}
			if extGroup.Title == "" {
				extGroup.Title = groupID + "s"
			}
			extGroupsByID[groupID] = extGroup
			extGroups = append(extGroups, extGroup)
		}
		extGroup.Count++

		groupHasExt := false
		for _, extStat := range extGroup.ExtStats {
			if extStat.ID == extID {
				groupHasExt = true
				break
			}
		}
		if !groupHasExt {
			extGroup.ExtStats = append(extGroup.ExtStats, ext)
		}
	}

	slices.SortFunc(extStats, func(a, b *ExtStat) int {
		return strings.Compare(a.ID, b.ID)
	})

	slices.SortFunc(extGroups, func(a, b *ExtensionsGroup) int {
		if a.ID == otherExtensionsGroupID {
			return 1
		}
		if b.ID == otherExtensionsGroupID {
			return -1
		}
		return strings.Compare(a.Title, b.Title)
	})

	for _, group := range extGroups {
		slices.SortFunc(group.ExtStats, func(a, b *ExtStat) int {
			return strings.Compare(a.ID, b.ID)
		})
	}

	//d.updateTable()

	hasRepo := gitutils.GetRepositoryRoot(dirPath) != ""
	d.setTabs(hasRepo)
	if hasRepo && dirContext != nil {
		if d.GitPreviewer.statusLoader != nil {
			d.GitPreviewer.SetDir(dirContext, d.queueUpdateDraw)
		}
	}
	if hasRepo {
		d.activateGitTabIfDirty(dirPath)
	}

	if d.queueUpdateDraw == nil {
		d.extByID = extByID
		d.ExtStats = extStats
		d.extGroupsByID = extGroupsByID
		d.ExtGroups = extGroups
		err := d.GetSizes()
		if err == nil {
			d.updateTable()
		}
		return
	}

	currentDirPath := dirPath
	go func() {
		err := getSizesForGroups(extGroups)
		if err != nil {
			return
		}
		d.queueUpdate(func() {
			if d.dirPath != currentDirPath {
				return
			}
			d.extByID = extByID
			d.ExtStats = extStats
			d.extGroupsByID = extGroupsByID
			d.ExtGroups = extGroups
			d.updateTable()
		})
	}()
}

func (d *DirPreviewer) activateGitTabIfDirty(dirPath string) {
	if d.GitPreviewer == nil {
		return
	}
	statusLoader := d.GitPreviewer.statusLoader
	if statusLoader == nil {
		return
	}
	currentDirPath := dirPath
	go func() {
		result, err := statusLoader(dirPath)
		if err != nil {
			return
		}
		if result.repoRoot == "" || len(result.entries) == 0 {
			return
		}
		d.queueUpdate(func() {
			if d.dirPath != currentDirPath {
				return
			}
			if d.tabs == nil {
				return
			}
			d.tabs.SwitchTo(1)
		})
	}()
}

func (d *DirPreviewer) setTabs(hasGit bool) {
	defer func() {
		r := recover()
		if r != nil {
			log.Println("panic in setting tabs: ", r)
		}
	}()
	if d.tabs != nil {
		d.flex.RemoveItem(d.tabs)
	}
	tabs := sneatv.NewTabs(d.app, sneatv.UnderlineTabsStyle)
	fileTab := sneatv.NewTab("file_types", "File types", false, d.ExtTable)
	tabs.AddTabs(fileTab)
	if hasGit {
		gitTab := sneatv.NewTab("git", "Git", false, d.GitPreviewer.Main())
		tabs.AddTabs(gitTab)
	}
	d.tabs = tabs
	d.flex.AddItem(tabs, 0, 1, false)
}

func (d *DirPreviewer) queueUpdate(f func()) {
	if d.queueUpdateDraw != nil {
		d.queueUpdateDraw(f)
		return
	}
	f()
}
