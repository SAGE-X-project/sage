package registry011

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestRPCResponseConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name   string
		id     uint64
		limits RPCResponseLimits
	}{
		{"zero byte limit", 1, RPCResponseLimits{0, 8}},
		{"negative byte limit", 1, RPCResponseLimits{-1, 8}},
		{"zero depth", 1, RPCResponseLimits{1024, 0}},
		{"negative depth", 1, RPCResponseLimits{1024, -1}},
		{"unsupported depth", 1, RPCResponseLimits{1024, 129}},
		{"notification id", 0, RPCResponseLimits{1024, 8}},
		{"unsafe numeric id", 1 << 53, RPCResponseLimits{1024, 8}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DecodeRPCResult([]byte(`{"jsonrpc":"2.0","id":1,"result":null}`), tc.id, tc.limits)
			if result != nil || !errors.Is(err, ErrConfiguration) {
				t.Fatalf("result=%q err=%v", result, err)
			}
		})
	}
}

func TestRPCResponseResults(t *testing.T) {
	for _, tc := range []struct{ name, body, result string }{
		{"hex string", `{"jsonrpc":"2.0","id":1,"result":"0xab"}`, `"0xab"`},
		{"null is only a value", `{"jsonrpc":"2.0","id":1,"result":null}`, `null`},
		{"false is only a value", `{"jsonrpc":"2.0","id":1,"result":false}`, `false`},
		{"structured result", `{"result":{"list":[true,0,{"a":1}]},"id":1,"jsonrpc":"2.0"}`, `{"list":[true,0,{"a":1}]}`},
		{"empty structures", `{"result":[{},[]],"id":1,"jsonrpc":"2.0"}`, `[{},[]]`},
		{"escaped version", `{"jsonrpc":"\u0032.0","id":1,"result":"ok"}`, `"ok"`},
		{"paired surrogate", `{"jsonrpc":"2.0","id":1,"result":"\ud83d\ude00"}`, `"\ud83d\ude00"`},
		{"ordinary escapes", `{"jsonrpc":"2.0","id":1,"result":"\\uD800\n\""}`, `"\\uD800\n\""`},
		{"replacement literal", `{"jsonrpc":"2.0","id":1,"result":"�"}`, `"�"`},
		{"separate key scopes", `{"jsonrpc":"2.0","id":1,"result":[{"a":1},{"a":2}]}`, `[{"a":1},{"a":2}]`},
		{"whitespace", " \n{\"jsonrpc\": \"2.0\", \"id\": 1, \"result\": [1, 2]}\t", `[1, 2]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DecodeRPCResult([]byte(tc.body), 1, RPCResponseLimits{4096, 8})
			if err != nil || string(got) != tc.result {
				t.Fatalf("got=%q err=%v", got, err)
			}
		})
	}

	const maximumID = 1<<53 - 1
	body := []byte(`{"jsonrpc":"2.0","id":9007199254740991,"result":[]}`)
	if _, err := DecodeRPCResult(body, maximumID, RPCResponseLimits{len(body), 2}); err != nil {
		t.Fatal(err)
	}
}

func TestRPCResponseRefusals(t *testing.T) {
	for name, body := range map[string]string{
		"empty":                  ``,
		"truncated":              `{"jsonrpc":"2.0","id":1,"result":`,
		"trailing object":        `{"jsonrpc":"2.0","id":1,"result":null}{}`,
		"trailing text":          `{"jsonrpc":"2.0","id":1,"result":null} x`,
		"batch":                  `[{"jsonrpc":"2.0","id":1,"result":null}]`,
		"scalar":                 `null`,
		"wrong version":          `{"jsonrpc":"1.0","id":1,"result":null}`,
		"version type":           `{"jsonrpc":2,"id":1,"result":null}`,
		"null version":           `{"jsonrpc":null,"id":1,"result":null}`,
		"missing version":        `{"id":1,"result":null,"extra":0}`,
		"missing id":             `{"jsonrpc":"2.0","result":null,"extra":0}`,
		"wrong id":               `{"jsonrpc":"2.0","id":2,"result":null}`,
		"string id":              `{"jsonrpc":"2.0","id":"1","result":null}`,
		"null id":                `{"jsonrpc":"2.0","id":null,"result":null}`,
		"fractional id spelling": `{"jsonrpc":"2.0","id":1.0,"result":null}`,
		"exponent id spelling":   `{"jsonrpc":"2.0","id":1e0,"result":null}`,
		"missing result":         `{"jsonrpc":"2.0","id":1}`,
		"remote error":           `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"unavailable"}}`,
		"result with error":      `{"jsonrpc":"2.0","id":1,"result":null,"error":null}`,
		"extra member":           `{"jsonrpc":"2.0","id":1,"result":null,"extra":true}`,
		"duplicate id":           `{"jsonrpc":"2.0","id":1,"id":1,"result":null}`,
		"escaped duplicate":      `{"jsonrpc":"2.0","id":1,"\u0069d":1,"result":null}`,
		"duplicate nested":       `{"jsonrpc":"2.0","id":1,"result":[{"a":1,"a":1}]}`,
		"nested truncated":       `{"jsonrpc":"2.0","id":1,"result":[{"a":`,
		"bad object key":         `{"jsonrpc":"2.0","id":1,"result":{1:0}}`,
		"mismatched closer":      `{"jsonrpc":"2.0","id":1,"result":[1}}`,
		"trailing comma":         `{"jsonrpc":"2.0","id":1,"result":[1,]}`,
		"unpaired high":          `{"jsonrpc":"2.0","id":1,"result":"\ud800"}`,
		"unpaired low":           `{"jsonrpc":"2.0","id":1,"result":"\udc00"}`,
		"wrong low pair":         `{"jsonrpc":"2.0","id":1,"result":"\ud800\u0041"}`,
		"truncated pair":         `{"jsonrpc":"2.0","id":1,"result":"\ud800\u12`,
		"nonhex pair":            `{"jsonrpc":"2.0","id":1,"result":"\ud800\uzzzz"}`,
		"truncated escape":       `{"jsonrpc":"2.0","id":1,"result":"\u12`,
		"nonhex escape":          `{"jsonrpc":"2.0","id":1,"result":"\uzzzz"}`,
		"trailing backslash":     "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"\\",
		"invalid escape":         `{"jsonrpc":"2.0","id":1,"result":"\q"}`,
		"invalid utf8":           "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"\xff\"}",
	} {
		t.Run(name, func(t *testing.T) {
			got, err := DecodeRPCResult([]byte(body), 1, RPCResponseLimits{4096, 8})
			if got != nil || !errors.Is(err, ErrRPCResponse) {
				t.Fatalf("got=%q err=%v", got, err)
			}
		})
	}
}

func TestRPCResponseLimitsAndCopies(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"result":[1]}`)
	if _, err := DecodeRPCResult(body, 1, RPCResponseLimits{len(body) - 1, 2}); !errors.Is(err, ErrRPCResponse) {
		t.Fatalf("byte limit: %v", err)
	}
	if _, err := DecodeRPCResult(body, 1, RPCResponseLimits{len(body), 1}); !errors.Is(err, ErrRPCResponse) {
		t.Fatalf("depth limit: %v", err)
	}
	result, err := DecodeRPCResult(body, 1, RPCResponseLimits{len(body), 2})
	if err != nil {
		t.Fatal(err)
	}
	body[bytes.Index(body, []byte("[1]"))+1] = '9'
	if string(result) != "[1]" {
		t.Fatalf("result aliases body: %s", result)
	}
	result[1] = '3'
	if !bytes.Contains(body, []byte("[9]")) {
		t.Fatal("body aliases result")
	}
	deep := `{"jsonrpc":"2.0","id":1,"result":` + strings.Repeat("[", 127) + "0" + strings.Repeat("]", 127) + "}"
	if _, err := DecodeRPCResult([]byte(deep), 1, RPCResponseLimits{1024, 128}); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeRPCResult([]byte(deep), 1, RPCResponseLimits{1024, 127}); !errors.Is(err, ErrRPCResponse) {
		t.Fatalf("depth boundary: %v", err)
	}
}

func TestRPCResponseConcurrent(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"result":{"blockHash":"0x00"}}`)
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			result, err := DecodeRPCResult(body, 1, RPCResponseLimits{1024, 8})
			if err != nil || string(result) != `{"blockHash":"0x00"}` {
				t.Errorf("result=%q err=%v", result, err)
			}
		})
	}
	workers.Wait()
}

func TestRPCResponseHistoricalFinalizedHeader(t *testing.T) {
	body, err := os.ReadFile("testdata/official-finalized-rpc-response.json")
	if err != nil {
		t.Fatal(err)
	}
	result, err := DecodeRPCResult(body, 3, RPCResponseLimits{1 << 20, 8})
	if err != nil {
		t.Fatal(err)
	}
	var header struct{ Hash, ParentHash, Number string }
	if err := json.Unmarshal(result, &header); err != nil {
		t.Fatal(err)
	}
	if header.Hash != "0x8897590310b098f735c7bbb1b8e4d197b1641b208d09425743d0726f365f9af2" ||
		header.ParentHash != "0x06a3082b91d0ff912e0c7da9840dbd044b2dec20a41f2f53db9f67bf239983af" ||
		header.Number != "0xb6c0e6" {
		t.Fatalf("unexpected historical header: %+v", header)
	}
	// Pin the original response bytes; this offline check does not establish
	// current freshness, finality, state authentication or Source readiness.
	digest := sha256.Sum256(body)
	if hex.EncodeToString(digest[:]) != "963f38403f1381e8f5e66e65f791b51bc102f7965070c0e917ae3e8305abdae9" {
		t.Fatal("historical response bytes changed")
	}
}

func ExampleDecodeRPCResult() {
	result, err := DecodeRPCResult([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`), 1,
		RPCResponseLimits{MaxBytes: 256, MaxDepth: 4})
	fmt.Println(string(result), err)
	// Output: "0x1" <nil>
}
