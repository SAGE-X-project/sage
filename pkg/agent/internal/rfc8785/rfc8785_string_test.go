package rfc8785

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestStringSpansPreserveEveryEscapeAndUTF8Byte(t *testing.T) {
	inputs := []string{"", "ASCII property and 0123456789", strings.Repeat("0xabcdef0123456789", 64), "é中😀\u2028\u2029", "\"\\", "beforeé\"😀after"}
	for i := 0; i < 32; i++ {
		inputs = append(inputs, "prefix😀"+string(rune(i))+"中suffix")
	}
	for _, left := range inputs {
		for _, right := range []string{"", "\n", "😀", "\\\"é"} {
			input := left + right
			var got, want bytes.Buffer
			if err := writeString(&got, input); err != nil {
				t.Fatal(err)
			}
			if err := referenceRuneString(&want, input); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got.Bytes(), want.Bytes()) {
				t.Fatalf("changed string bytes for %q: got %q want %q", input, got.Bytes(), want.Bytes())
			}
		}
	}
	var got bytes.Buffer
	if err := writeString(&got, string([]byte{0xff})); err == nil || got.Len() != 0 {
		t.Fatal("invalid UTF-8 was serialized")
	}
}

func referenceRuneString(buf *bytes.Buffer, s string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("invalid UTF-8")
	}
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(buf, `\u%04x`, r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
	return nil
}
