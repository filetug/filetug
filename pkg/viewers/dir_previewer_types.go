package viewers

import "os"

// ExtensionsGroup represents a group of file extensions with aggregated statistics.
type ExtensionsGroup struct {
	ID    string
	Title string
	*GroupStats
	ExtStats []*ExtStat
}

// GroupStats contains aggregated statistics for a group of files.
type GroupStats struct {
	Count     int
	TotalSize int64
}

// ExtStat contains statistics for a specific file extension.
type ExtStat struct {
	ID string
	GroupStats
	entries []os.DirEntry
}
