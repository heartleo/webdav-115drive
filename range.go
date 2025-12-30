package main

import (
	"strconv"
	"strings"
)

func parseRange(v string, size int64) (ok bool, start, end int64) {
	if !strings.HasPrefix(v, "bytes=") {
		return false, 0, 0
	}
	spec := strings.TrimPrefix(v, "bytes=")
	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 {
		return false, 0, 0
	}

	// Suffix range "-N": last N bytes
	if parts[0] == "" {
		if parts[1] == "" {
			return false, 0, 0
		}
		suffix, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || suffix <= 0 {
			return false, 0, 0
		}
		if suffix >= size {
			return true, 0, size - 1
		}
		return true, size - suffix, size - 1
	}

	// Normal range "start-end" or "start-"
	s, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || s < 0 {
		return false, 0, 0
	}

	var e int64
	if parts[1] == "" {
		e = size - 1
	} else {
		ee, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || ee < s {
			return false, 0, 0
		}
		e = ee
	}

	if size <= 0 || s >= size {
		return false, 0, 0
	}
	if e >= size {
		e = size - 1
	}
	return true, s, e
}
