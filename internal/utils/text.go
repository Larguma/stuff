package utils

import "strings"

func NormalizeName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func SplitTags(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		switch r {
		case ',', ';', '\n', '\t':
			return true
		default:
			return false
		}
	})

	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		clean = append(clean, trimmed)
	}

	return clean
}
