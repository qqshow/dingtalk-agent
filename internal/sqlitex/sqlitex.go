// Package sqlitex holds small helpers shared by the SQLite users.
package sqlitex

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

// URI turns a filesystem path into an SQLite URI filename with the given
// query parameters (e.g. "mode=ro&immutable=1").
func URI(path, query string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	p := filepath.ToSlash(abs)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p // Windows drive letter: file:///D:/...
	}
	r := strings.NewReplacer("%", "%25", " ", "%20", "#", "%23", "?", "%3f")
	u := "file://" + r.Replace(p)
	if query != "" {
		u += "?" + query
	}
	return u
}

// Open opens path with modernc sqlite.
func Open(path, query string) (*sql.DB, error) {
	return sql.Open("sqlite", URI(path, query))
}

// Int converts a scanned SQLite value to int64.
func Int(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case []byte:
		n, _ := strconv.ParseInt(strings.TrimSpace(string(x)), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		return n
	case bool:
		if x {
			return 1
		}
	}
	return 0
}

// Str converts a scanned SQLite value to string.
func Str(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []byte:
		return string(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return fmt.Sprint(x)
	}
}
