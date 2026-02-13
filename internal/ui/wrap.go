package ui

import "strings"

// wrapText wraps text to the given width at word boundaries.
// It preserves existing newlines and hard-breaks lines with no spaces.
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if len(line) <= width {
			out = append(out, line)
			continue
		}
		for len(line) > width {
			// Check if we can break exactly at the width boundary
			if line[width] == ' ' {
				out = append(out, line[:width])
				line = line[width+1:]
				continue
			}
			// Find last space within the width
			idx := strings.LastIndex(line[:width], " ")
			if idx <= 0 {
				// No space found — hard break
				out = append(out, line[:width])
				line = line[width:]
			} else {
				out = append(out, line[:idx])
				line = line[idx+1:]
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
