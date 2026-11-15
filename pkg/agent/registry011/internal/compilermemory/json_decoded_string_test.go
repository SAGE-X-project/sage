package compilermemory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"
)

func TestRegionJSONDecodedStringsMatchReference(t *testing.T) {
	for index, raw := range []string{
		"\"\"", "\"plain\"", "\"<>&\"", "\"\\\"\\\\\\/\\b\\f\\n\\r\\t\"",
		"\"\\u0061\\u00af\\uabcd\"", "\"\\ud83d\\ude00\"",
		"\"\\ud800\"", "\"\\udfff\"", "\"\\ud800\\u0061\"",
		"\"\\ud800\\ud800\"", "\"\\u0000\"", "\"\\ufffd\"",
	} {
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			var expected string
			if err := json.Unmarshal([]byte(raw), &expected); err != nil {
				t.Fatal(err)
			}
			region, document := jsonTestDocument(t, raw)
			value := document.Root()
			length, err := value.DecodedStringLength()
			if err != nil || length != len(expected) {
				t.Fatalf("decoded length=%d, error=%v; expected=%d", length, err, len(expected))
			}
			destination := bytes.Repeat([]byte{0xcc}, length)
			if err := value.CopyString(destination); err != nil || string(destination) != expected {
				t.Fatalf("decoded bytes=%q, error=%v; expected=%q", destination, err, expected)
			}
			if len(destination) != 0 {
				destination[0] ^= 0xff
				if equal, err := value.StringEqual(expected); err != nil || !equal {
					t.Fatalf("detached destination changed original: %v, %v", equal, err)
				}
				if err := value.CopyString(destination); err != nil {
					t.Fatal(err)
				}
			}
			if err := region.Close(); err != nil {
				t.Fatal(err)
			}
			if string(destination) != expected {
				t.Fatal("decoded destination borrowed the closed mapping")
			}
		})
	}
}

func TestRegionJSONDecodedStringsRefuseInvalidOwnersAndDestinations(t *testing.T) {
	region, document := jsonTestDocument(t, "\"native\"")
	for _, length := range []int{0, 5, 7} {
		destination := bytes.Repeat([]byte{0xcc}, length)
		if err := document.Root().CopyString(destination); !errors.Is(err, ErrRange) || !bytes.Equal(destination, make([]byte, length)) {
			t.Fatalf("destination length %d: %x, %v", length, destination, err)
		}
	}
	for _, raw := range []string{"0", "true", "null", "[]", "{}"} {
		t.Run(raw, func(t *testing.T) {
			_, other := jsonTestDocument(t, raw)
			if length, err := other.Root().DecodedStringLength(); length != 0 || !errors.Is(err, ErrJSONType) {
				t.Fatalf("wrong-kind length: %d, %v", length, err)
			}
			destination := []byte{1, 2, 3}
			if err := other.Root().CopyString(destination); !errors.Is(err, ErrJSONType) || !bytes.Equal(destination, []byte{0, 0, 0}) {
				t.Fatalf("wrong-kind copy: %x, %v", destination, err)
			}
		})
	}
	copied := &Region{self: region}
	bad := document.Root()
	bad.document.owner = copied
	foreign, _ := jsonTestDocument(t, "\"other\"")
	mixed := document.Root()
	mixed.document.nodes.owner = foreign
	for index, value := range []JSONValue{{}, bad, mixed} {
		if length, err := value.DecodedStringLength(); length != 0 || err == nil {
			t.Fatalf("invalid owner %d length: %d, %v", index, length, err)
		}
		destination := []byte{1, 2}
		if err := value.CopyString(destination); err == nil || !bytes.Equal(destination, []byte{0, 0}) {
			t.Fatalf("invalid owner %d copy: %x, %v", index, destination, err)
		}
	}
	if err := region.Close(); err != nil {
		t.Fatal(err)
	}
	if length, err := document.Root().DecodedStringLength(); length != 0 || !errors.Is(err, ErrClosed) {
		t.Fatalf("closed length: %d, %v", length, err)
	}
	destination := []byte{1, 2}
	if err := document.Root().CopyString(destination); !errors.Is(err, ErrClosed) || !bytes.Equal(destination, []byte{0, 0}) {
		t.Fatalf("closed copy: %x, %v", destination, err)
	}
}

func TestRegionJSONDecodedStringsClearOnOriginalContextRefusal(t *testing.T) {
	for _, operation := range []string{"length", "copy"} {
		for checks := 1; checks <= 32; checks++ {
			t.Run(operation+"/"+strconv.Itoa(checks), func(t *testing.T) {
				action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				t.Cleanup(cancel)
				ctx := &integerReadContext{Context: action}
				region := jsonTestRegion(t, ctx)
				span, err := region.Store([]byte("\"copy\""))
				if err != nil {
					t.Fatal(err)
				}
				document, err := region.ParseJSON(span, JSONLimits{MaximumDepth: 2, MaximumNodes: 1})
				if err != nil {
					t.Fatal(err)
				}
				ctx.mu.Lock()
				ctx.failAt = ctx.calls + checks
				ctx.mu.Unlock()
				if operation == "length" {
					length, err := document.Root().DecodedStringLength()
					if err != nil {
						if !errors.Is(err, context.Canceled) || length != 0 {
							t.Fatalf("length on refusal: %d, %v", length, err)
						}
					} else if length != 4 {
						t.Fatalf("successful length=%d", length)
					}
					return
				}
				destination := []byte{1, 2, 3, 4}
				err = document.Root().CopyString(destination)
				if err != nil {
					if !errors.Is(err, context.Canceled) || !bytes.Equal(destination, []byte{0, 0, 0, 0}) {
						t.Fatalf("copy on refusal: %x, %v", destination, err)
					}
				} else if string(destination) != "copy" {
					t.Fatalf("successful copy=%q", destination)
				}
			})
		}
	}
}

func TestRegionJSONDecodedStringsRefuseCorruptedSimulatedTokens(t *testing.T) {
	for _, replacement := range []byte{'"', '\\', 0xff} {
		region, document := jsonTestDocument(t, "\"x\"")
		region.data[document.input.offset+1] = replacement
		if length, err := document.Root().DecodedStringLength(); length != 0 || err == nil {
			t.Fatalf("corrupt length: %d, %v", length, err)
		}
		destination := []byte{1}
		if err := document.Root().CopyString(destination); err == nil || destination[0] != 0 {
			t.Fatalf("corrupt copy: %x, %v", destination, err)
		}
	}
	region, document := jsonTestDocument(t, "0")
	node := readJSONRecord(region.data[document.nodes.offset : document.nodes.offset+jsonNodeBytes])
	node.kind = JSONString
	writeJSONRecord(region.data[document.nodes.offset:document.nodes.offset+jsonNodeBytes], node)
	if length, err := document.Root().DecodedStringLength(); length != 0 || !errors.Is(err, ErrJSON) {
		t.Fatalf("short simulated token: %d, %v", length, err)
	}
}
