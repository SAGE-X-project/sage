package registry011

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

func TestProviderHistoryEventHashAndDigestHaveIndependentConcreteOracles(t *testing.T) {
	raw := []byte("abc")
	want, err := hex.DecodeString("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad")
	if err != nil {
		t.Fatal(err)
	}
	m := providerHistoryContentUnitMeter(candidateWorkVector{10: 4})
	got, ok := replaySHA256Observed(raw, m)
	if !ok || !bytes.Equal(got[:], want) || m.actual != (candidateWorkVector{10: 4}) {
		t.Fatal("SHA256 abc/hash-input4", got, m.actual)
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{10: 3})
	if got, ok := replaySHA256Observed(raw, short); ok || got != ([32]byte{}) || !short.failed || short.actual != (candidateWorkVector{}) {
		t.Fatal("unfunded hash invocation")
	}
	before := short.actual
	if _, ok := replaySHA256Observed(raw, short); ok || short.actual != before {
		t.Fatal("failed hash observer resumed")
	}
	m = providerHistoryContentUnitMeter(candidateWorkVector{10: 8})
	textHash, ok := replayTextSHA256Observed("abc", m)
	if !ok || textHash != got || m.actual != (candidateWorkVector{10: 8}) {
		t.Fatal("text copy4/hash4", m.actual)
	}
	const digest = "ungWv48Bz-pBQUDeXa4iI7ADYaOWF3qctBD_YfIAFa0"
	m = providerHistoryContentUnitMeter(candidateWorkVector{10: 1854})
	if got, ok := replayDigestObserved(raw, m); !ok || got != digest || m.actual != (candidateWorkVector{10: 1854}) {
		t.Fatal("hash4/base64actual1850", got, m.actual)
	}
	short = providerHistoryContentUnitMeter(candidateWorkVector{10: 1853})
	if got, ok := replayDigestObserved(raw, short); ok || got != "" || !short.failed || short.actual != (candidateWorkVector{10: 4}) {
		t.Fatal("late digest encode one-short", got, short.actual)
	}
	if got, ok := replayDigestObserved(raw, nil); !ok || got != digest {
		t.Fatal("legacy digest bytes")
	}
}

func TestProviderHistoryEventTopicsAndBoundedHashFailuresKeepZeroResults(t *testing.T) {
	want := "0000000000000000000000000000000000000000000000000000000000001000"
	got, ok := replayAddressTopicObserved(replayTestWallet, providerHistoryExecutionUnitMeter())
	if !ok || hex.EncodeToString(got[:]) != want {
		t.Fatal("indexed actor topic byte layout", got)
	}
	legacy, ok := replayAddressTopicObserved(replayTestWallet, nil)
	if !ok || legacy != got {
		t.Fatal("legacy indexed topic")
	}
	for _, actor := range []string{"", "invalid", strings.ToUpper(replayTestWallet)} {
		if got, ok := replayAddressTopicObserved(actor, providerHistoryExecutionUnitMeter()); ok || got != ([32]byte{}) {
			t.Fatal("invalid topic address")
		}
	}
	if got, ok := replayTextSHA256Observed(strings.Repeat("a", 262145), providerHistoryExecutionUnitMeter()); ok || got != ([32]byte{}) {
		t.Fatal("text hash source cap")
	}
	if got, ok := replaySHA256Observed(make([]byte, 262209), providerHistoryExecutionUnitMeter()); ok || got != ([32]byte{}) {
		t.Fatal("raw hash source/domain cap")
	}
	if got, ok := replayTextSHA256Observed("abc", providerHistoryContentUnitMeter(candidateWorkVector{})); ok || got != ([32]byte{}) {
		t.Fatal("unfunded text conversion")
	}
}

func TestProviderHistoryClaimDigestUsesActualCommandAndAdoptedDomain(t *testing.T) {
	c := replayTestCreate(t)
	want := replayClaim(replayTestRegistry, c)
	for _, m := range []*providerWorkMeter{nil, providerHistoryExecutionUnitMeter()} {
		got, ok := replayClaimObserved(replayTestRegistry, c, m)
		if !ok || got != want {
			t.Fatal("actual claim bytes/domain changed")
		}
	}
	for _, change := range []string{"operation", "agent", "registry", "controller", "fields", "callback"} {
		bad := c
		args := make(map[string]any, len(c.args))
		for key, value := range c.args {
			args[key] = value
		}
		bad.args = args
		registry := replayTestRegistry
		switch change {
		case "operation":
			bad.op = "activate"
		case "agent":
			bad.agent = ""
		case "registry":
			registry = "other"
		case "controller":
			bad.args["controller"] = replayTestOperator
		case "fields":
			delete(bad.args, "salt")
		case "callback":
			bad.args["keys"] = providerHistoryTupleCallback{}
		}
		if got, ok := replayClaimObserved(registry, bad, providerHistoryExecutionUnitMeter()); ok || got != ([32]byte{}) {
			t.Fatal("changed private claim prerequisite", change)
		}
	}
	m := providerHistoryContentUnitMeter(candidateWorkVector{})
	if got, ok := replayClaimObserved(replayTestRegistry, c, m); ok || got != ([32]byte{}) || !m.failed {
		t.Fatal("unfunded original claim observer")
	}
	if !reflect.DeepEqual(c.args["controller"], replayTestWallet) {
		t.Fatal("claim failure changed original command")
	}
}
