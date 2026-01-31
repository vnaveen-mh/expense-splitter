package groups

import "strings"

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizeGroupName(name string) string {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return ""
	}
	return strings.ToLower(strings.Join(parts, "-"))
}
