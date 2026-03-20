package pathutil

import (
	"path"
	"strings"
)

// Normalize enforces a leading slash and cleans path traversal.
func Normalize(p string) (string, error) {
	if p == "" {
		return "/", nil
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	clean := path.Clean(p)
	if !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	if strings.Contains(clean, "..") {
		return "", ErrInvalidPath
	}
	return clean, nil
}

// ErrInvalidPath is returned when a path fails validation.
var ErrInvalidPath = Err("invalid path")

// Err is a lightweight error type for sentinel errors.
type Err string

func (e Err) Error() string { return string(e) }
