package viewers

// fileExtTypes maps file extensions to their category types.
var fileExtTypes = map[string]string{
	// Image file extStats
	".jpg":  "Image",
	".jpeg": "Image",
	".png":  "Image",
	".gif":  "Image",
	".bmp":  "Image",
	".riff": "Image",
	".tiff": "Image",
	".vp8":  "Image",
	".vp8l": "Image",
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

// otherExtensionsGroupID is the ID for extensions not in fileExtTypes.
const otherExtensionsGroupID = "Other"

// fileExtPlurals maps category types to their plural display names.
var fileExtPlurals = map[string]string{
	"Data": "Data",
	"Code": "Code",
}
