package rfc9421

import (
	"io"
	"strings"
)

// readCloser wraps a string as an io.ReadCloser for request bodies in tests.
func readCloser(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}
