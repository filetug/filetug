package gitext

import (
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/rivo/tview"
)

type StatusPanel struct {
	*sneatv.Boxed
	flex *tview.Flex
}

func NewStatusPanel() *StatusPanel {
	flex := tview.NewFlex()
	flex.SetTitle("Git Status")
	p := &StatusPanel{
		flex:  flex,
		Boxed: sneatv.NewBoxed(flex),
	}
	return p
}

func (p *StatusPanel) SetDir(dir string) {

}
