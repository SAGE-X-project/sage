package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const testRegistry = "0x9361223865e641f9941f4640d8fe34faad54bf23"
const testBlock = "0x" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const testParent = "0x" + "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func readABI(t *testing.T) abi.ABI {
	t.Helper()
	data, err := os.ReadFile("testdata/chain-registry-abi.json")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "0f31f995b0169b27c2ce4797b967a754b4c0b2b0a733e4c21da1b54089380f5f" {
		t.Fatal("spec ABI identity changed")
	}
	parsed, err := abi.JSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

// Independent go-ethereum ABI encoder consumes the spec-pinned thirteen-method
// ABI; production uses a separately implemented bounded decoder/encoder.
func returnABI(t *testing.T, name string, values ...any) string {
	t.Helper()
	parsed := readABI(t)
	encoded, err := parsed.Methods[name].Outputs.Pack(values...)
	if err != nil {
		t.Fatal(err)
	}
	return "0x" + hex.EncodeToString(encoded)
}

type rpcReadFixture struct {
	mu           sync.Mutex
	calls        []string
	mode         string
	historyCalls int
}

func fixtureRPC(t *testing.T, mode string) (*ReadRPC, *rpcReadFixture) {
	t.Helper()
	f := &rpcReadFixture{mode: mode}
	c := testReadRPC(t, func(req *http.Request) (*http.Response, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		var request struct {
			JSONRPC string `json:"jsonrpc"`
			ID      int
			Method  string
			Params  []json.RawMessage
		}
		body, err := io.ReadAll(req.Body)
		if err != nil || json.Unmarshal(body, &request) != nil {
			t.Fatal("request parse")
		}
		if request.JSONRPC != "2.0" || request.ID != 1 || req.URL.String() != SepoliaRPCURL {
			t.Fatal("envelope/endpoint")
		}
		f.calls = append(f.calls, request.Method)
		var result any
		switch request.Method {
		case "eth_chainId":
			result = "0xaa36a7"
			if mode == "wrong chain" {
				result = "0x1"
			}
		case "eth_getBlockByNumber":
			if string(request.Params[1]) != "false" {
				t.Fatal("full transactions requested")
			}
			hash := testBlock
			number := "0x2"
			stamp := "0x30"
			if string(request.Params[0]) == `"0x0"` {
				hash = sepoliaGenesis
				number = "0x0"
				stamp = "0x0"
				if mode == "wrong genesis" {
					hash = testBlock
				}
				if mode == "wrong genesis height" {
					number = "0x1"
				}
			}
			if string(request.Params[0]) == `"finalized"` {
				switch mode {
				case "null header":
					result = nil
					return fixtureResult(result), nil
				case "noncanonical height":
					number = "0x02"
				case "bad hash":
					hash = "0x1"
				case "unsafe timestamp":
					stamp = "0x20000000000000"
				}
			}
			result = map[string]any{"hash": hash, "parentHash": testParent, "number": number, "timestamp": stamp, "futureExtension": true}
		case "eth_getCode", "eth_call":
			var block map[string]any
			if json.Unmarshal(request.Params[1], &block) != nil || len(block) != 2 || block["blockHash"] != testBlock || block["requireCanonical"] != true {
				t.Fatal("mixed/noncanonical block")
			}
			if mode == "pin unsupported" {
				return jsonResponse(`{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"unavailable"}}`), nil
			}
			if request.Method == "eth_getCode" {
				result = "0x6001"
				break
			}
			var call struct{ To, Data string }
			if json.Unmarshal(request.Params[0], &call) != nil || call.To != testRegistry {
				t.Fatal("target")
			}
			encoded, err := hex.DecodeString(strings.TrimPrefix(call.Data, "0x"))
			if err != nil || len(encoded) < 4 {
				t.Fatal("calldata")
			}
			parsed := readABI(t)
			method, err := parsed.MethodById(encoded[:4])
			if err != nil {
				t.Fatal(err)
			}
			args, err := method.Inputs.Unpack(encoded[4:])
			if err != nil {
				t.Fatal(err)
			}
			canonical, err := method.Inputs.Pack(args...)
			if err != nil || !bytes.Equal(canonical, encoded[4:]) {
				t.Fatal("noncanonical calldata")
			}
			if method.Name != "registryId" && (len(args) == 0 || args[0] != "Agent_1") {
				t.Fatal("agent swapped")
			}
			switch method.Name {
			case "registryId":
				id := "eip155:11155111:" + testRegistry
				if mode == "wrong registry" {
					id += "x"
				}
				result = returnABI(t, method.Name, id)
			case "record":
				result = returnABI(t, method.Name, true, []byte(`{"version":"2"}`))
				if mode == "absent" {
					result = returnABI(t, method.Name, false, []byte{})
				}
				if mode == "false positive bytes" {
					result = returnABI(t, method.Name, false, []byte("x"))
				}
				if mode == "empty positive" {
					result = returnABI(t, method.Name, true, []byte{})
				}
			case "activeGrants":
				result = returnABI(t, method.Name, []byte("[]"))
				if mode == "empty grants" {
					result = returnABI(t, method.Name, []byte{})
				}
			case "historyItem":
				f.historyCalls++
				version := args[1].(uint64)
				if mode == "history gap" && f.historyCalls == 2 {
					return jsonResponse(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"unavailable"}}`), nil
				}
				entry := []byte(fmt.Sprintf(`{"version":"%d"}`, version))
				command := []byte("{}")
				if mode == "empty history" {
					entry = nil
				}
				if mode == "empty command" {
					command = nil
				}
				result = returnABI(t, method.Name, entry, command)
			default:
				t.Fatal("non-read method")
			}
		default:
			t.Fatal("unapproved method")
		}
		return fixtureResult(result), nil
	})
	return c, f
}
func fixtureResult(result any) *http.Response {
	encoded, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result})
	return jsonResponse(string(encoded))
}
func pinFixture(t *testing.T, mode string) (*PinnedAgentReads, *rpcReadFixture) {
	t.Helper()
	c, f := fixtureRPC(t, mode)
	r, err := c.PinAgentReads(context.Background(), testRegistry, "Agent_1")
	if err != nil {
		t.Fatal(err)
	}
	return r, f
}
func TestPinnedReadsNormalAndCopies(t *testing.T) {
	r, f := pinFixture(t, "")
	ctx := context.Background()
	header, err := r.Header()
	if err != nil || header.Hash != testBlock {
		t.Fatal(header, err)
	}
	header.Hash = testParent
	code, err := r.CodeBytes(ctx, testRegistry)
	if err != nil || !bytes.Equal(code, []byte{0x60, 1}) {
		t.Fatal(code, err)
	}
	present, record, err := r.RecordBytes(ctx)
	if err != nil || !present || string(record) != `{"version":"2"}` {
		t.Fatal(present, string(record), err)
	}
	grants, err := r.GrantsBytes(ctx)
	if err != nil || string(grants) != "[]" {
		t.Fatal(string(grants), err)
	}
	rows, err := r.HistoryRange(ctx, 0, 2, HistoryReadLimits{2, 100})
	if err != nil || len(rows) != 2 || rows[0].Version != 1 || rows[1].Version != 2 {
		t.Fatal(rows, err)
	}
	record[0] = 'x'
	grants[0] = 'x'
	rows[0].Entry[0] = 'x'
	rows[0].Command[0] = 'x'
	_, again, err := r.RecordBytes(ctx)
	if err != nil || again[0] != '{' {
		t.Fatal("alias")
	}
	if f.historyCalls != 2 {
		t.Fatal("range")
	}
	unchanged, err := r.HistoryRange(ctx, math.MaxUint64, math.MaxUint64, HistoryReadLimits{1, 1})
	if err != nil || len(unchanged) != 0 {
		t.Fatal(unchanged, err)
	}
	rows, err = r.HistoryRange(ctx, math.MaxUint64-1, math.MaxUint64, HistoryReadLimits{1, 100})
	if err != nil || len(rows) != 1 || rows[0].Version != math.MaxUint64 {
		t.Fatal(rows, err)
	}
}
func TestPinnedAdmissionRefusals(t *testing.T) {
	for _, mode := range []string{"wrong chain", "wrong genesis", "wrong genesis height", "null header", "noncanonical height", "bad hash", "unsafe timestamp", "wrong registry", "pin unsupported"} {
		t.Run(mode, func(t *testing.T) {
			c, f := fixtureRPC(t, mode)
			r, err := c.PinAgentReads(context.Background(), testRegistry, "Agent_1")
			if r != nil || err == nil {
				t.Fatal(r, err)
			}
			if len(f.calls) > 4 {
				t.Fatal("fallback/retry")
			}
		})
	}
	for _, agent := range []string{"", ".", "..", "a/b", "é", strings.Repeat("a", 65)} {
		t.Run("agent "+agent, func(t *testing.T) {
			c, f := fixtureRPC(t, "")
			r, err := c.PinAgentReads(context.Background(), testRegistry, agent)
			if r != nil || !errors.Is(err, ErrConfiguration) || len(f.calls) != 0 {
				t.Fatal(r, err)
			}
		})
	}
	for _, address := range []string{"", "0x" + strings.Repeat("0", 40), strings.ToUpper(testRegistry), "xx" + testRegistry[2:], "0x" + strings.Repeat("1", 40)} {
		t.Run("address "+address, func(t *testing.T) {
			c, f := fixtureRPC(t, "")
			r, err := c.PinAgentReads(context.Background(), address, "Agent_1")
			if r != nil || !errors.Is(err, ErrConfiguration) || len(f.calls) != 0 {
				t.Fatal(r, err)
			}
		})
	}
}
func TestPinnedReadRefusalsAndBudgets(t *testing.T) {
	for _, mode := range []string{"false positive bytes", "empty positive", "empty grants", "empty history", "empty command", "history gap"} {
		t.Run(mode, func(t *testing.T) {
			r, _ := pinFixture(t, mode)
			ctx := context.Background()
			var err error
			switch mode {
			case "false positive bytes", "empty positive":
				_, _, err = r.RecordBytes(ctx)
			case "empty grants":
				_, err = r.GrantsBytes(ctx)
			default:
				var rows []HistoryBytes
				rows, err = r.HistoryRange(ctx, 0, 2, HistoryReadLimits{2, 100})
				if rows != nil {
					t.Fatal("partial success")
				}
			}
			if err == nil {
				t.Fatal("accepted")
			}
		})
	}
	r, f := pinFixture(t, "")
	ctx := context.Background()
	for _, tc := range []struct {
		name           string
		after, current uint64
		limits         HistoryReadLimits
	}{
		{"rollback", 2, 1, HistoryReadLimits{2, 100}}, {"count cap", 0, 3, HistoryReadLimits{2, 100}}, {"zero count cap", 0, 1, HistoryReadLimits{0, 100}},
		{"zero byte cap", 0, 1, HistoryReadLimits{2, 0}},
		{"unsupported item cap", 0, 1, HistoryReadLimits{4097, 100}},
		{"uint64 item cap", 0, 1, HistoryReadLimits{math.MaxUint64, 100}},
		{"unsupported byte cap", 0, 1, HistoryReadLimits{2, (8 << 20) + 1}},
		{"platform byte cap", 0, 1, HistoryReadLimits{2, math.MaxInt}}, {"entry cap", 0, 1, HistoryReadLimits{2, 1}}, {"command cap", 0, 1, HistoryReadLimits{2, 15}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := r.HistoryRange(ctx, tc.after, tc.current, tc.limits)
			if rows != nil || err == nil {
				t.Fatal(rows, err)
			}
		})
	}
	if f.historyCalls != 2 {
		t.Fatal("preflight failed to bound requests", f.historyCalls)
	}
	a, _ := pinFixture(t, "absent")
	present, body, err := a.RecordBytes(ctx)
	if err != nil || present || len(body) != 0 {
		t.Fatal(present, body, err)
	}
	var nilReads *PinnedAgentReads
	if _, err := nilReads.Header(); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	if _, _, err := nilReads.RecordBytes(ctx); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	if _, err := nilReads.GrantsBytes(ctx); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	if _, err := nilReads.CodeBytes(ctx, testRegistry); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	if _, err := nilReads.HistoryRange(ctx, 0, 1, HistoryReadLimits{1, 100}); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	if _, err := r.CodeBytes(ctx, ""); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	var nilRPC *ReadRPC
	if _, err := nilRPC.PinAgentReads(ctx, testRegistry, "Agent_1"); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
}
func TestPinnedReadsConcurrent(t *testing.T) {
	r, _ := pinFixture(t, "")
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			_, body, err := r.RecordBytes(context.Background())
			if err != nil || len(body) == 0 {
				t.Error(err)
			}
		})
	}
	workers.Wait()
}
func TestPinnedQuantityAndData(t *testing.T) {
	for _, v := range []string{"0x0", "0x1", "0xffffffffffffffff"} {
		if _, ok := rpcQuantity(v); !ok {
			t.Fatal(v)
		}
	}
	for _, v := range []string{"", "0x", "0X1", "0x00", "0xA", "0x+1", "0x10000000000000000"} {
		if _, ok := rpcQuantity(v); ok {
			t.Fatal(v)
		}
	}
	for _, v := range []string{`null`, `42`, `"ab"`, `"0x0"`, `"0xAA"`, `"0xgg"`} {
		if _, err := rpcData(json.RawMessage(v)); err == nil {
			t.Fatal(v)
		}
	}
}
func TestPinnedCanonicalABIBoundaries(t *testing.T) {
	valid := returnABI(t, "historyItem", []byte("a"), []byte("b"))
	raw, _ := hex.DecodeString(valid[2:])
	for _, name := range []string{"short head", "unaligned", "offset high bits", "gap", "aliased tail", "length high bits", "missing tail", "nonzero padding", "trailing word", "over limit"} {
		t.Run(name, func(t *testing.T) {
			b := append([]byte(nil), raw...)
			max := 262144
			switch name {
			case "short head":
				b = b[:32]
			case "unaligned":
				b = b[:len(b)-1]
			case "offset high bits":
				b[0] = 1
			case "gap":
				b[31] = 96
			case "aliased tail":
				copy(b[32:64], b[:32])
			case "length high bits":
				b[64] = 1
			case "missing tail":
				b = b[:64]
			case "nonzero padding":
				b[97] = 1
			case "trailing word":
				b = append(b, make([]byte, 32)...)
			case "over limit":
				max = 0
			}
			encoded, _ := json.Marshal("0x" + hex.EncodeToString(b))
			if _, _, err := rpcABI(encoded, false, max, max); err == nil {
				t.Fatal("accepted", name)
			}
		})
	}
	for _, name := range []string{"bool high bits", "bool not one", "record too large"} {
		t.Run(name, func(t *testing.T) {
			encoded := returnABI(t, "record", true, []byte("x"))
			b, _ := hex.DecodeString(encoded[2:])
			max := 65536
			switch name {
			case "bool high bits":
				b[0] = 1
			case "bool not one":
				b[31] = 2
			case "record too large":
				max = 0
			}
			text, _ := json.Marshal("0x" + hex.EncodeToString(b))
			if _, _, err := rpcABI(text, true, max); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	for _, size := range []int{0, 1, 31, 32, 33, 65536} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			b := bytes.Repeat([]byte{'x'}, size)
			text, _ := json.Marshal(returnABI(t, "activeGrants", b))
			values, _, err := rpcABI(text, false, 65536)
			if err != nil || !bytes.Equal(values[0], b) {
				t.Fatal(size, err)
			}
		})
	}
}
