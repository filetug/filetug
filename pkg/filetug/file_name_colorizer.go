package filetug

import (
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
)

var fileColors = map[string]tcell.Color{
	"exe":  tcell.ColorRed,
	"go":   tcell.ColorAqua,
	"cpp":  tcell.ColorDodgerBlue,
	"c":    tcell.ColorDodgerBlue,
	"h":    tcell.ColorDodgerBlue,
	"cs":   tcell.ColorLime,
	"js":   tcell.ColorYellow,
	"ts":   tcell.ColorDeepSkyBlue,
	"html": tcell.ColorOrangeRed,
	"css":  tcell.ColorViolet,
	"sql":  tcell.ColorSpringGreen,
	"json": tcell.ColorGold,
	"xml":  tcell.ColorLightYellow,
	"yaml": tcell.ColorLightYellow,
	"yml":  tcell.ColorLightYellow,
	"md":   tcell.ColorBisque,
	"py":   tcell.ColorLightGreen,
	"rb":   tcell.ColorRed,
	"php":  tcell.ColorPurple,
	"rs":   tcell.ColorOrange,
	"sh":   tcell.ColorGreen,
	"bat":  tcell.ColorDarkRed,
	"txt":  tcell.ColorWhite,
	"csv":  tcell.ColorLightGreen,
	"jpg":  tcell.ColorMediumPurple,
	"jpeg": tcell.ColorMediumPurple,
	"png":  tcell.ColorMediumPurple,
	"gif":  tcell.ColorMediumPurple,
	"webp": tcell.ColorMediumPurple,
	"mov":  tcell.ColorLightSalmon,
	"mp4":  tcell.ColorLightSalmon,
	"log":  tcell.ColorRosyBrown,
	"xls":  tcell.ColorGreen,
	"xlsx": tcell.ColorGreen,
	"doc":  tcell.ColorBlue,
	"docx": tcell.ColorBlue,
}

func GetColorByFileExt(name string) tcell.Color {
	ext := strings.TrimPrefix(filepath.Ext(name), ".")
	if color, ok := fileColors[ext]; ok {
		return color
	}
	return tcell.ColorWhiteSmoke
}
