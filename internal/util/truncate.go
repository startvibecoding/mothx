package util

// TruncateString returns a valid UTF-8 prefix of s whose byte length is at most maxBytes.
func TruncateString(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	end := 0
	for idx := range s {
		if idx > maxBytes {
			break
		}
		end = idx
	}
	if end == 0 {
		return ""
	}
	return s[:end]
}

// TruncateWithSuffix truncates s with TruncateString and appends suffix when truncation occurs.
func TruncateWithSuffix(s string, maxBytes int, suffix string) string {
	if len(s) <= maxBytes {
		return s
	}
	return TruncateString(s, maxBytes) + suffix
}
