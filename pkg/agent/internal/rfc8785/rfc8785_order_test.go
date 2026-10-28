package rfc8785

import (
	"testing"
	"unicode/utf16"
)

func TestPropertyOrderMatchesUnsignedUTF16(t *testing.T) {
	names := []string{"", "a", "aa", "ab", "A", "0", "_", "~", "\x00", "\r", "\"", "\\", "\x7f", "\u0080", "\u00f6", "\u20ac", "\ue000", "\ufb33", "\U0001f600", "a\u20ac", "\U00010000", "\U0010ffff"}
	reference := func(a, b string) bool {
		left, right := utf16.Encode([]rune(a)), utf16.Encode([]rune(b))
		for i := 0; i < len(left) && i < len(right); i++ {
			if left[i] != right[i] {
				return left[i] < right[i]
			}
		}
		return len(left) < len(right)
	}
	for _, left := range names {
		for _, right := range names {
			if got, want := lessUTF16(left, right), reference(left, right); got != want {
				t.Fatalf("UTF-16 property order %q/%q: got %v want %v", left, right, got, want)
			}
		}
	}
	// A UTF-8 comparison alone gets this non-ASCII order wrong.
	if !lessUTF16("\U0001f600", "\ue000") || lessUTF16("\ue000", "\U0001f600") {
		t.Fatal("supplementary code point bypassed UTF-16 ordering")
	}
}

func TestASCIIPropertyComparisonDoesNotAllocate(t *testing.T) {
	allocations := testing.AllocsPerRun(100, func() {
		if !lessUTF16("authority_metadata_root", "budget_root") {
			t.Fatal("ASCII metadata property order")
		}
	})
	if allocations != 0 {
		t.Fatalf("ASCII property comparison allocated %v times", allocations)
	}
}

func TestMixedPropertyNamesKeepCanonicalBytes(t *testing.T) {
	input := []byte(`{"\ue000":1,"budget_root":2,"\ud83d\ude00":3,"authority_metadata_root":4,"\r":5}`)
	got, err := Canonicalize(input)
	want := "{\"\\r\":5,\"authority_metadata_root\":4,\"budget_root\":2,\"\U0001f600\":3,\"\ue000\":1}"
	if err != nil || string(got) != want {
		t.Fatalf("mixed canonical order: %q %v", got, err)
	}
}
