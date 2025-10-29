package lb

import (
	"net/http"
	"strings"
)

// ExtractPriority derives a task priority from request metadata.
// It checks headers first, falling back to query parameters, and defaults to "normal".
func ExtractPriority(r *http.Request) string {
	if r == nil {
		return "normal"
	}

	if header := r.Header.Get("X-Task-Priority"); header != "" {
		return normalizePriority(header)
	}

	if q := r.URL.Query().Get("priority"); q != "" {
		return normalizePriority(q)
	}

	return "normal"
}

func normalizePriority(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "low", "medium", "high", "critical":
		return v
	default:
		return "normal"
	}
}
