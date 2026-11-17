package provider

import (
	"encoding/json"
	"strings"
)

// toStringSlice coerces the list shapes Vault returns (a JSON array decodes to
// []interface{}) into a []string. A nil or non-list value yields an empty slice.
func toStringSlice(v interface{}) []string {
	switch xs := v.(type) {
	case []string:
		return xs
	case []interface{}:
		out := make([]string, 0, len(xs))
		for _, x := range xs {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return []string{}
	}
}

// toInt64 coerces the numeric shapes Vault returns (json.Number, float64, int)
// into an int64.
func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			f, ferr := n.Float64()
			if ferr != nil {
				return 0, false
			}
			return int64(f), true
		}
		return i, true
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	default:
		return 0, false
	}
}

// splitBackendName splits an import ID of the form "<backend>/<marker>/<name>"
// (e.g. "gitea/roles/ci") into its backend and name parts.
func splitBackendName(id, marker string) (backend, name string, ok bool) {
	sep := "/" + marker + "/"
	idx := strings.LastIndex(id, sep)
	if idx <= 0 {
		return "", "", false
	}
	backend = id[:idx]
	name = id[idx+len(sep):]
	if backend == "" || name == "" {
		return "", "", false
	}
	return backend, name, true
}
