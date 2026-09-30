package registry010

import (
	"errors"
	"strings"
)

// ErrInvalidRecord010 is the REG-08 record.invalid result for a malformed web
// Registry response. A successful media check does not validate its body.
var ErrInvalidRecord010 = errors.New("record.invalid")

// HeaderField010 preserves one HTTP field line. The trusted HTTP adapter must
// retain duplicate lines and trailers before any coalescing or decompression.
type HeaderField010 struct {
	Name  string
	Value string
}

func asciiEqualFold010(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		c := a[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != b[i] {
			return false
		}
	}
	return true
}

func fieldLine010(f HeaderField010) bool {
	if f.Name == "" {
		return false
	}
	for i := 0; i < len(f.Name); i++ {
		c := f.Name[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c))) {
			return false
		}
	}
	for i := 0; i < len(f.Value); i++ {
		c := f.Value[i]
		if (c < 0x20 && c != '\t') || c == 0x7f {
			return false
		}
	}
	return true
}

// CheckWebRegistryMedia010 checks only the REG-08 response media and content
// coding boundary. It is not an HTTP fetcher, body parser, authoritative Source,
// or proof that a Registry record may be used for authentication.
func CheckWebRegistryMedia010(header, trailer []HeaderField010) error {
	var contentType string
	seenType := false
	for _, f := range header {
		if !fieldLine010(f) {
			return ErrInvalidRecord010
		}
		if asciiEqualFold010(f.Name, "content-encoding") {
			return ErrInvalidRecord010
		}
		if asciiEqualFold010(f.Name, "content-type") {
			if seenType {
				return ErrInvalidRecord010
			}
			seenType = true
			contentType = strings.Trim(f.Value, " \t")
		}
	}
	for _, f := range trailer {
		if !fieldLine010(f) || asciiEqualFold010(f.Name, "content-type") ||
			asciiEqualFold010(f.Name, "content-encoding") {
			return ErrInvalidRecord010
		}
	}
	if !seenType || !asciiEqualFold010(contentType, "application/json") {
		return ErrInvalidRecord010
	}
	return nil
}
