package main

import (
	"fmt"
	"sort"
	"strings"
)

const (
	// base64MinLength filters out short strings that are almost certainly
	// normal text rather than encoded image data.
	base64MinLength = 16
)

// formatJSON renders a top-level JSON object as human-readable indented text.
// String values that look like base64 image data are replaced with a
// placeholder.
func formatJSON(data map[string]any) string {
	return formatValue(data, 0)
}

// formatValue recursively formats a JSON value as indented text.
func formatValue(value any, indent int) string {
	prefix := strings.Repeat("  ", indent)
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var sb strings.Builder
		for _, k := range keys {
			val := v[k]
			if val == nil {
				fmt.Fprintf(&sb, "%s%s: null\n", prefix, k)
			} else if s, ok := val.(string); ok && s == "" {
				fmt.Fprintf(&sb, "%s%s: \"\"\n", prefix, k)
			} else if isCompound(val) {
				fmt.Fprintf(&sb, "%s%s:\n%s", prefix, k, formatValue(val, indent+1))
			} else {
				fmt.Fprintf(&sb, "%s%s: %s\n", prefix, k, formatValue(val, indent+1))
			}
		}
		return sb.String()
	case []any:
		var sb strings.Builder
		for i, val := range v {
			fmt.Fprintf(&sb, "%s[%d]: %s\n", prefix, i, formatValue(val, indent+1))
		}
		return sb.String()
	case string:
		if v == "" {
			return `""`
		}
		return v
	case float64:
		return fmt.Sprintf("%g", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// isCompound reports whether v is a map or slice (i.e. a value that formats
// across multiple lines).
func isCompound(v any) bool {
	switch v.(type) {
	case map[string]any, []any:
		return true
	}
	return false
}

// applyFilter performs a case-insensitive text search across keys and
// values, returning the formatted result. An empty query returns the
// full document.
func applyFilter(query string, data map[string]any) string {
	if query == "" {
		return formatJSON(data)
	}
	needle := strings.ToLower(query)
	matched := filterMap(data, needle)
	if len(matched) == 0 {
		return "  No results"
	}
	return formatJSON(matched)
}

// filterMap returns entries from data where the key or any string value
// (recursively) contains needle (case-insensitive).
func filterMap(data map[string]any, needle string) map[string]any {
	result := make(map[string]any)
	for k, v := range data {
		if matchesValue(k, v, needle) {
			result[k] = v
		}
	}
	return result
}

// matchesValue reports whether a key or its value tree contains needle.
func matchesValue(key string, value any, needle string) bool {
	if strings.Contains(strings.ToLower(key), needle) {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.Contains(strings.ToLower(v), needle)
	case float64:
		return strings.Contains(fmt.Sprintf("%g", v), needle)
	case bool:
		return strings.Contains(fmt.Sprintf("%v", v), needle)
	case map[string]any:
		for k, v := range v {
			if matchesValue(k, v, needle) {
				return true
			}
		}
	case []any:
		for _, item := range v {
			if matchesValue("", item, needle) {
				return true
			}
		}
	}
	return false
}

// isBase64 reports whether s looks like valid standard base64. It performs
// a fast, allocation-free character check rather than decoding the full
// string — actual decoding happens later in decodeImage.
func isBase64(s string) bool {
	if len(s)%4 != 0 {
		return false
	}
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '+' && r != '/' && r != '=' {
			return false
		}
	}
	return true
}

// isImageString reports whether s looks like base64-encoded image data,
// either as a data URI or a sufficiently long valid base64 string.
func isImageString(s string) bool {
	if strings.HasPrefix(s, "data:image/") {
		return true
	}
	return len(s) > base64MinLength && isBase64(s)
}
