package main

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

const (
	// base64MinLength is the minimum string length before we consider it
	// a possible base64-encoded image. Short strings are not worth the
	// cost of a trial decode.
	base64MinLength = 100
)

// formatJSON renders a top-level JSON object as human-readable indented text.
// Keys listed in imageFields are displayed as a placeholder instead of their
// raw base64 value.
func formatJSON(data map[string]any, imageFields map[string]bool) string {
	return formatValue(data, 0, imageFields)
}

// formatValue recursively formats a JSON value as indented text. Values whose
// keys appear in imageFields are rendered as a placeholder.
func formatValue(value any, indent int, imageFields map[string]bool) string {
	prefix := strings.Repeat("  ", indent)
	switch v := value.(type) {
	case map[string]any:
		var sb strings.Builder
		for k, val := range v {
			if imageFields[k] {
				fmt.Fprintf(&sb, "%s%s: (base64 image data)\n", prefix, k)
				continue
			}
			if val == nil || val == "" {
				fmt.Fprintf(&sb, "%s%s: (empty)\n", prefix, k)
			} else if isCompound(val) {
				fmt.Fprintf(&sb, "%s%s:\n%s", prefix, k,
					formatValue(val, indent+1, imageFields))
			} else {
				fmt.Fprintf(&sb, "%s%s: %s\n", prefix, k,
					formatValue(val, indent+1, imageFields))
			}
		}
		return sb.String()
	case []any:
		var sb strings.Builder
		for i, val := range v {
			fmt.Fprintf(&sb, "%s[%d]: %s\n", prefix, i, formatValue(val, indent+1, imageFields))
		}
		return sb.String()
	case string:
		if v == "" {
			return "(empty)"
		}
		return v
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%.2f", v)
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

// applyFilter runs a gjson query against the raw JSON and formats the result.
// An empty query returns the full formatted document.
func applyFilter(raw []byte, query string, data map[string]any, imageFields map[string]bool) string {
	if query == "" {
		return formatJSON(data, imageFields)
	}
	result := gjson.GetBytes(raw, query)
	if !result.Exists() {
		return "  No results"
	}
	return formatResult(result, imageFields)
}

// formatResult formats a gjson query result as human-readable text.
func formatResult(result gjson.Result, imageFields map[string]bool) string {
	val := result.Value()
	if val == nil {
		return "  null"
	}
	if m, ok := val.(map[string]any); ok {
		return formatJSON(m, imageFields)
	}
	s := formatValue(val, 0, imageFields)
	if s == "" {
		return "  (empty)"
	}
	return s
}

// isBase64 reports whether s is valid standard base64.
func isBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

// isImageString reports whether s looks like base64-encoded image data,
// either as a data URI or a sufficiently long valid base64 string.
func isImageString(s string) bool {
	if strings.HasPrefix(s, "data:image/") {
		return true
	}
	return len(s) > base64MinLength && isBase64(s)
}
