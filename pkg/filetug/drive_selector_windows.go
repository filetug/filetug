package filetug

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

var getLogicalDriveStrings = windows.GetLogicalDriveStrings

func getWindowsDrives() []string {
	buf := make([]uint16, 254)
	n, err := getLogicalDriveStrings(uint32(len(buf)), &buf[0])

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to read windows drives: %v\n", err)
		return nil
	}

	// Convert UTF-16 buffer to Go string list
	drives := windows.UTF16ToString(buf[:n])
	return splitNull(drives)
}
