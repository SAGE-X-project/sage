package registry011

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestProviderHistoryCommandBinaryUsesActualDecodeEncodeAndLateEquality(t *testing.T) {
	raw := bytes.Repeat([]byte{3}, 32)
	text := base64.RawURLEncoding.EncodeToString(raw)
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 43, 10: 3787})
	got, ok := replayCommandBinaryObserved(text, 32, m)
	// Decode43^2+1=1850; actual encode1850; equality86+1=87.
	if !ok || !bytes.Equal(got, raw) || m.actual != (candidateWorkVector{2: 43, 10: 3787}) {
		t.Fatal("exact binary actual", got, ok, m.actual)
	}
	got[0] = 9
	if text != base64.RawURLEncoding.EncodeToString(raw) {
		t.Fatal("binary output changed input string")
	}
	for _, upper := range []candidateWorkVector{{}, {2: 42, 10: 3787}, {2: 43, 10: 3699}, {2: 43, 10: 3786}} {
		short := providerHistoryContentUnitMeter(upper)
		if got, ok := replayCommandBinaryObserved(text, 32, short); ok || got != nil || !short.failed {
			t.Fatal("short binary observer returned partial decoded bytes", short.actual)
		}
		before := short.actual
		if got, ok := replayCommandBinaryObserved(text, 32, short); ok || got != nil || short.actual != before {
			t.Fatal("failed binary original resumed")
		}
	}
	for _, tc := range []struct {
		value any
		size  int
	}{{text, 0}, {text, 66}, {1, 32}, {"bad", 32}, {strings.Repeat("!", 43), 32}, {text[:42] + "B", 32}} {
		if got, ok := replayCommandBinaryObserved(tc.value, tc.size, providerHistoryExecutionUnitMeter()); ok || got != nil {
			t.Fatal("noncanonical binary scalar")
		}
	}
	if got, ok := replayCommandBinaryObserved(text, 32, nil); ok || got != nil {
		t.Fatal("missing binary original observer")
	}
}

func TestProviderHistoryCommandNumbersPreserveCanonicalAndExpiryBounds(t *testing.T) {
	for _, text := range []string{"0", "1", "9007199254740991", "18446744073709551615"} {
		m := providerHistoryExecutionUnitMeter()
		if _, ok := replayCommandUintObserved(text, m); !ok || m.failed {
			t.Fatal("canonical uint64", text)
		}
	}
	for _, text := range []string{"", "01", "-1", "+1", "1e1", "18446744073709551616", strings.Repeat("1", 21)} {
		if _, ok := replayCommandUintObserved(text, providerHistoryExecutionUnitMeter()); ok {
			t.Fatal("noncanonical integer", text)
		}
	}
	for _, v := range []any{json.Number("0"), json.Number("9007199254740991")} {
		if _, ok := replayCommandSecondsObserved(v, providerHistoryExecutionUnitMeter()); !ok {
			t.Fatal("exact seconds", v)
		}
	}
	for _, v := range []any{0, "0", json.Number("9007199254740992"), json.Number("1.0")} {
		if _, ok := replayCommandSecondsObserved(v, providerHistoryExecutionUnitMeter()); ok {
			t.Fatal("invalid seconds", v)
		}
	}
	if _, ok := replayCommandUintObserved("1", nil); ok {
		t.Fatal("missing number observer")
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{2: 1, 10: 402})
	if _, ok := replayCommandUintObserved("1", short); ok || !short.failed {
		t.Fatal("decimal encode preflight requires actual2+upper401")
	}
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 1, 10: 403})
	if n, ok := replayCommandUintObserved("1", m); !ok || n != 1 || m.actual != (candidateWorkVector{2: 1, 10: 7}) {
		t.Fatal("decimal encoder upper recorded as actual", n, ok, m.actual)
	}
}

func TestProviderHistoryCommandAddressNameAndURIKeepExistingGrammar(t *testing.T) {
	for _, text := range []string{replayTestWallet, replayTestOperator, SepoliaRegistryAddress} {
		got, ok := replayCommandAddressObserved(text, providerHistoryExecutionUnitMeter())
		if !ok || got != common.HexToAddress(text) {
			t.Fatal("canonical address", text)
		}
	}
	for _, text := range []string{"", "0X" + replayTestWallet[2:], "invalid"} {
		if _, ok := replayCommandAddressObserved(text, providerHistoryExecutionUnitMeter()); ok {
			t.Fatal("noncanonical address")
		}
	}
	for _, text := range []string{"a", "A2A-0_", "", "bad.name", strings.Repeat("a", 33)} {
		if got := replayCommandNameObserved(text, providerHistoryExecutionUnitMeter()); got != replayName(text) {
			t.Fatal("existing name grammar changed")
		}
	}
	for _, text := range []string{"https://example.invalid/api", "HTTPS://[::1]:443/a?x=%20", "https://[v1.x]/", "http://example.invalid", "https://x/%ZZ", strings.Repeat("a", 2049)} {
		if got := replayCommandURIObserved(text, providerHistoryExecutionUnitMeter()); got != replayURI(text) {
			t.Fatal("existing HTTPS/IP grammar changed", text)
		}
	}
	if _, ok := replayCommandAddressObserved(replayTestWallet, nil); ok || replayCommandNameObserved("a", nil) || replayCommandURIObserved("https://x", nil) {
		t.Fatal("missing scalar original observer")
	}
}

func TestProviderHistoryCommandArgumentsAndGrantScopesStayClosed(t *testing.T) {
	for _, scope := range []string{"activate", "add-key", "revoke-key", "update-services", "deactivate", "unknown"} {
		args := map[string]any{"operator": replayTestOperator, "scope": scope}
		if replayCommandGrantObserved(args, providerHistoryExecutionUnitMeter()) != contentGrant(args) {
			t.Fatal("existing scope grammar changed", scope)
		}
	}
	for _, op := range []string{"create", "activate", "deactivate", "add-key", "revoke-key", "update-services", "authorize-operator", "revoke-operator"} {
		var args map[string]any
		switch op {
		case "create":
			args = map[string]any{"controller": nil, "keys": nil, "services": nil, "salt": nil}
		case "activate", "deactivate":
			args = map[string]any{}
		case "add-key":
			args = map[string]any{"key": nil}
		case "revoke-key":
			args = map[string]any{"name": nil}
		case "update-services":
			args = map[string]any{"services": nil}
		default:
			args = map[string]any{"operator": nil, "scope": nil}
		}
		want, accepted := contentArguments(op, args)
		got, ok := replayCommandArgumentsObserved(op, args, providerHistoryExecutionUnitMeter())
		if !accepted || !ok || !reflect.DeepEqual(got, want) {
			t.Fatal("adopted argument fields", op)
		}
	}
	for _, op := range []string{"unknown", "activate"} {
		if got, ok := replayCommandArgumentsObserved(op, map[string]any{"extra": 0}, providerHistoryExecutionUnitMeter()); ok || got != nil {
			t.Fatal("unknown operation or field")
		}
	}
	if got, ok := replayCommandArgumentsObserved("activate", map[string]any{}, nil); ok || got != nil || replayCommandGrantObserved(nil, providerHistoryExecutionUnitMeter()) {
		t.Fatal("missing argument observer or operator")
	}
	if replayCommandObjectObserved(map[string]any{}, make([]string, 9), providerHistoryExecutionUnitMeter()) != nil || replayCommandObjectObserved(map[string]any{"x": 0}, []string{"y"}, providerHistoryExecutionUnitMeter()) != nil {
		t.Fatal("closed object field list")
	}
}
