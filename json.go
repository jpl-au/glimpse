package main

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	// base64MinLength is the minimum string length before we consider it
	// a possible base64-encoded image. Short strings are not worth the
	// cost of a trial decode.
	base64MinLength = 100
)

// humanizeFieldName converts a camelCase or PascalCase field name into
// space-separated title case (e.g. "firstName" → "First Name").
func humanizeFieldName(name string) string {
	var result strings.Builder
	for i, ch := range name {
		if i > 0 && ch >= 'A' && ch <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(ch)
	}
	words := strings.Fields(result.String())
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(string(w[0])) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

// formatJSON renders a top-level JSON object as human-readable indented text.
// Keys listed in imageFields are displayed as a placeholder instead of their
// raw base64 value.
func formatJSON(data map[string]any, imageFields map[string]bool) string {
	var sb strings.Builder
	for key, value := range data {
		fmt.Fprintf(&sb, "━━ %s ━━\n", humanizeFieldName(key))
		sb.WriteString(formatValue(value, 1, imageFields))
		sb.WriteString("\n")
	}
	return sb.String()
}

// formatValue recursively formats a JSON value as indented text. Values whose
// keys appear in imageFields are rendered as a placeholder.
func formatValue(value any, indent int, imageFields map[string]bool) string {
	prefix := strings.Repeat("  ", indent)
	switch v := value.(type) {
	case map[string]any:
		var sb strings.Builder
		for k, val := range v {
			hk := humanizeFieldName(k)
			if imageFields[k] {
				fmt.Fprintf(&sb, "%s%s: (base64 image data)\n", prefix, hk)
				continue
			}
			if val == nil || val == "" {
				fmt.Fprintf(&sb, "%s%s: (empty)\n", prefix, hk)
			} else {
				fmt.Fprintf(&sb, "%s%s: %s\n", prefix, hk,
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
