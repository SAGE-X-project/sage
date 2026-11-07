package registry011

import (
	"strings"
	"testing"
)

func TestLowerHashValidatesEveryByteAtEveryDigestPosition(t *testing.T) {
	base := "0x" + strings.Repeat("0123456789abcdef", 4)
	if !lowerHash(base) || !lowerHash("0x"+strings.Repeat("0", 64)) || !lowerHash("0x"+strings.Repeat("f", 64)) {
		t.Fatal("complete lower-case hexadecimal digests must retain their original spelling")
	}
	// Independent byte grammar, including NUL, uppercase and invalid UTF-8.
	// Every position is checked so invalid middle bytes cannot disappear
	// when the valid leading and trailing characters are consumed.
	for position := 2; position < len(base); position++ {
		for value := 0; value < 256; value++ {
			raw := []byte(base)
			raw[position] = byte(value)
			want := value >= 48 && value <= 57 || value >= 97 && value <= 102
			if lowerHash(string(raw)) != want {
				t.Fatal("digest accepted a disallowed byte or rejected a lower-case hexadecimal byte", position, value)
			}
		}
	}
}

func TestLowerHashRequiresTheExactByteLengthAndPrefix(t *testing.T) {
	for length := 0; length <= 132; length++ {
		value := strings.Repeat("a", length)
		if length >= 2 {
			value = "0x" + value[2:]
		}
		if lowerHash(value) != (length == 66) {
			t.Fatal("digest length was interpreted as runes or normalized", length)
		}
	}
	for _, prefix := range []string{"0X", "1x", "ox", "  ", "\x00x", "é"} {
		if lowerHash(prefix + strings.Repeat("a", 64)) {
			t.Fatal("digest prefix was normalized or accepted as hexadecimal payload", prefix)
		}
	}
}
