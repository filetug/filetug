package filetug

var fileExtTypes = map[string]string{
	// Image file extStats
	".jpg":  "Image",
	".jpeg": "Image",
	".png":  "Image",
	".gif":  "Image",
	".webp": "Image",
	// Video file extStats
	".mov":  "Video",
	".mp4":  "Video",
	".webm": "Video",
	// Code file extStats
	".go":   "Code",
	".css":  "Code",
	".js":   "Code",
	".cpp":  "Code",
	".java": "Code",
	".cs":   "Code",
	// Data file extStats
	".json": "Data",
	".xml":  "Data",
	".dbf":  "Data",
	// Text file extStats
	".txt": "Text",
	".md":  "Text",
	// Log file extStats
	".log": "Log",
}

const otherExtensionsGroupID = "Other"

var fileExtPlurals = map[string]string{
	"Data": "Data",
}
