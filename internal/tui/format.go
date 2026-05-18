package tui

import (
	"fmt"
	"strings"
)

// humanSize renders a byte count as a short human-readable string.
func humanSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	const unit = 1024.0
	v := float64(n)
	suffixes := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	i := -1
	for v >= unit && i < len(suffixes)-1 {
		v /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s", v, suffixes[i])
}

// bytesWithCommas formats an int64 with thousand separators using commas.
// Negative values are formatted with a leading minus sign.
func bytesWithCommas(n int64) string {
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	// Insert commas from the right.
	var b []byte
	rem := len(s) % 3
	if rem > 0 {
		b = append(b, s[:rem]...)
	}
	for i := rem; i < len(s); i += 3 {
		if len(b) > 0 {
			b = append(b, ',')
		}
		b = append(b, s[i:i+3]...)
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// humanSizeRight returns humanSize(n) right-padded to width w with spaces.
// If the rendered size already exceeds w, the original is returned unpadded.
func humanSizeRight(n int64, w int) string {
	s := humanSize(n)
	if len(s) >= w {
		return s
	}
	return strings.Repeat(" ", w-len(s)) + s
}
