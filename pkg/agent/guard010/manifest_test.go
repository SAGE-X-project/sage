package guard010

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

func manifestJSON(files int) []byte {
	var b strings.Builder
	b.WriteString(`{"version":"0.10.0","files":[`)
	for i := 0; i < files; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"sha256":"%s","path":"f/%05d"}`, strings.Repeat("a", 64), i)
	}
	b.WriteString(`]}`)
	return []byte(b.String())
}

// CanonicalManifest returns exactly the bytes ManifestCommitment hashes, under
// the manifest's own member limit rather than the general 4096 limit.
func TestCanonicalManifestMatchesCommitment(t *testing.T) {
	for _, files := range []int{1, 3000} {
		raw := manifestJSON(files)
		b, err := CanonicalManifest(raw)
		if err != nil {
			t.Fatalf("%d files: %v", files, err)
		}
		digest, err := ManifestCommitment(raw)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(b)
		if digest != hex.EncodeToString(sum[:]) {
			t.Fatalf("%d files: commitment %s does not hash the canonical bytes", files, digest)
		}
		if !bytes.HasPrefix(b, []byte(`{"files":[{"path":"f/00000","sha256":"`)) {
			t.Fatalf("not canonical: %.60s", b)
		}
	}
	// 3000 files exceed the general member limit, so only the manifest path accepts them.
	if _, err := Canonicalize(manifestJSON(3000)); err == nil {
		t.Fatal("general canonicalization accepted a manifest-sized document")
	}
}

func TestCanonicalManifestRefusesInvalidManifests(t *testing.T) {
	for name, raw := range map[string]string{
		"duplicate": `{"version":"0.10.0","files":[],"files":[]}`,
		"version":   `{"version":"0.9.0","files":[]}`,
		"extra":     `{"version":"0.10.0","files":[],"x":1}`,
		"unsorted":  `{"version":"0.10.0","files":[{"path":"b","sha256":"` + strings.Repeat("a", 64) + `"},{"path":"a","sha256":"` + strings.Repeat("a", 64) + `"}]}`,
		"traversal": `{"version":"0.10.0","files":[{"path":"a/../b","sha256":"` + strings.Repeat("a", 64) + `"}]}`,
		"too-many":  string(manifestJSON(4097)),
		"not-json":  `{"version":`,
	} {
		if b, err := CanonicalManifest([]byte(raw)); err == nil || b != nil {
			t.Fatalf("%s accepted", name)
		}
	}
}
