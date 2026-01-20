package fsutils

import "strconv"

// GetSizeShortText returns a human readable size string.
func GetSizeShortText(size int64) string {
	const unit = 1024
	if size < unit {
		return strconv.FormatInt(size, 10) + "B"
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit && exp < 3; n /= unit {
		div *= unit
		exp++
	}
	// Rounding to nearest
	val := (size + div/2) / div
	// If rounding up pushes it to the next unit
	if val >= unit && exp < 3 { // TB is our last unit
		val /= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}
	return strconv.FormatInt(val, 10) + units[exp]
}
