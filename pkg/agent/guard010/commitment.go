package guard010

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode/utf8"
)

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var digest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var segment = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
var keyName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
var toolName = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,128}$`)
var chain = regexp.MustCompile(`^[1-9][0-9]*$`)
var address = regexp.MustCompile(`^0x[0-9a-f]{40}$`)
var domainLabel = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

func split(s string) []string               { return strings.Fields(s) }
func str(m map[string]any, k string) string { s, _ := m[k].(string); return s }
func hash(b []byte) string                  { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func did(s string) bool {
	if len(s) > 256 {
		return false
	}
	p := strings.Split(s, ":")
	if len(p) < 5 || p[0] != "did" || p[1] != "sage" || !segment.MatchString(p[len(p)-1]) || p[len(p)-1] == "." || p[len(p)-1] == ".." {
		return false
	}
	if p[2] == "eip155" {
		return len(p) == 6 && chain.MatchString(p[3]) && len(p[3]) <= 32 && address.MatchString(p[4])
	}
	if p[2] != "web" || len(p) != 5 || len(p[3]) > 64 {
		return false
	}
	for _, l := range strings.Split(p[3], ".") {
		if len(l) > 63 || !domainLabel.MatchString(l) {
			return false
		}
	}
	return true
}
func ascii(s string, max int) bool {
	if len(s) < 1 || len(s) > max {
		return false
	}
	for _, c := range []byte(s) {
		if c > 127 {
			return false
		}
	}
	return true
}

// OriginalCommitment commits exact ordered UTF-8 input bytes without normalization.
func OriginalCommitment(items [][]byte) (string, error) {
	if len(items) > 1024 {
		return "", ErrInvalid
	}
	total := 0
	for _, b := range items {
		if len(b) > MaxBytes-total || !utf8.Valid(b) {
			return "", ErrInvalid
		}
		total += len(b)
	}
	h := sha256.New()
	h.Write([]byte("sage-original|0.10.0\x00"))
	var n [8]byte
	binary.BigEndian.PutUint32(n[:4], uint32(len(items)))
	h.Write(n[:4])
	for _, b := range items {
		binary.BigEndian.PutUint64(n[:], uint64(len(b)))
		h.Write(n[:])
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func manifest(m map[string]any, nonempty bool) bool {
	if !closed(m, "version files") || str(m, "version") != "0.10.0" {
		return false
	}
	a, ok := m["files"].([]any)
	if !ok || len(a) > 4096 || (nonempty && len(a) == 0) {
		return false
	}
	last := ""
	for _, v := range a {
		f, ok := v.(map[string]any)
		if !ok || !closed(f, "path sha256") {
			return false
		}
		p := str(f, "path")
		if len(p) == 0 || len(p) > 1024 || p <= last || strings.ContainsAny(p, "\\\x00") || !digest.MatchString(str(f, "sha256")) {
			return false
		}
		for _, s := range strings.Split(p, "/") {
			if s == "" || s == "." || s == ".." {
				return false
			}
		}
		last = p
	}
	return true
}

// ManifestCommitment validates a descriptor, not filesystem identity or loading.
func ManifestCommitment(raw []byte) (string, error) {
	m, b, e := objectLimit(raw, 8194)
	if e != nil || !manifest(m, false) {
		return "", ErrInvalid
	}
	return hash(b), nil
}

// Artifact holds the exact bytes supplied by a trusted loader.
type Artifact struct {
	Path  string
	Bytes []byte
}

// VerifyManifest checks the exact artifact set. Hosts separately enforce regular
// files, no symlinks and immutable instance loading.
func VerifyManifest(raw []byte, items []Artifact) (string, error) {
	artifacts := map[string][]byte{}
	for _, item := range items {
		if _, ok := artifacts[item.Path]; ok {
			return "", ErrInvalid
		}
		artifacts[item.Path] = item.Bytes
	}
	m, b, e := objectLimit(raw, 8194)
	if e != nil || !manifest(m, false) {
		return "", ErrInvalid
	}
	a := m["files"].([]any)
	if len(a) != len(artifacts) {
		return "", ErrInvalid
	}
	for _, v := range a {
		f := v.(map[string]any)
		data, ok := artifacts[str(f, "path")]
		if !ok || hash(data) != str(f, "sha256") {
			return "", ErrInvalid
		}
	}
	return hash(b), nil
}

// PolicyCommitment validates a descriptor; a digest does not grant authority.
func PolicyCommitment(raw []byte) (string, error) {
	m, b, e := object(raw)
	if e != nil || !closed(m, "version issuer epoch engine artifacts") || str(m, "version") != "0.10.0" || !did(str(m, "issuer")) || !uuid.MatchString(str(m, "epoch")) || !ascii(str(m, "engine"), 128) {
		return "", ErrInvalid
	}
	a, ok := m["artifacts"].(map[string]any)
	if !ok || !manifest(a, true) {
		return "", ErrInvalid
	}
	return hash(append([]byte("sage-policy|0.10.0\x00"), b...)), nil
}
