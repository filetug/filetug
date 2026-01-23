package masks

import (
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Panel struct {
	*sneatv.Boxed
	table *tview.Table
	masks []Mask
}

func (p *Panel) Focus(delegate func(p tview.Primitive)) {
	p.table.Focus(delegate)
}

func NewPanel() *Panel {
	p := new(Panel)
	p.masks = createBuiltInMasks()

	p.table = tview.NewTable()
	p.table.SetSelectable(true, true)
	p.table.SetFixed(1, 1)

	p.Boxed = sneatv.NewBoxed(p.table,
		sneatv.WithLeftBorder(0, -1),
	)
	p.SetTitle("Masks")

	p.table.SetCell(0, 0, tview.NewTableCell("Mask").SetExpansion(1))
	p.table.SetCell(0, 1, tview.NewTableCell("CurrDir").SetAlign(tview.AlignRight))
	p.table.SetCell(0, 2, tview.NewTableCell("SubDirs").SetAlign(tview.AlignRight))

	for i, m := range p.masks {

		nameCell := tview.NewTableCell(m.Name)
		nameCell.SetExpansion(1)
		p.table.SetCell(i+1, 0, nameCell)

		currDirCell := tview.NewTableCell("...")
		currDirCell.SetAlign(tview.AlignRight)
		p.table.SetCell(i+1, 1, currDirCell)

		subDirsCell := tview.NewTableCell("...")
		subDirsCell.SetAlign(tview.AlignRight)
		subDirsCell.SetTextColor(tcell.ColorGray)
		p.table.SetCell(i+1, 2, subDirsCell)
	}

	return p
}
