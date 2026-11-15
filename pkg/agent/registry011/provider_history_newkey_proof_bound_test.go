package registry011

import (
	"encoding/base64"
	"reflect"
	"strings"
	"testing"
)

func TestProviderHistoryNewKeyProofBoundsMatchHandwrittenEventRows(t *testing.T) {
	for _, tc := range []struct{ added, retained, decoded, units64 uint64 }{
		{0, 0, 0, 133}, {1, 1, 486, 122994}, {2, 2, 972, 247107}, {128, 128, 62208, 25902597},
	} {
		// Independent event sums use the ordinary104-byte key descriptor.
		// Only actual declared copies change with the compiled target.
		copies := 2*tc.added + tc.added*tc.retained
		want := tc.units64 - copies*104 + copies*uint64(reflect.TypeFor[replayKey]().Size())
		got, ok := providerHistoryNewKeyProofUpper(tc.added, tc.retained)
		if !ok || got != (candidateWorkVector{2: tc.decoded, 7: tc.added, 10: want}) {
			t.Fatal("new-key envelope differs from independent event rows", tc, got, want, ok)
		}
	}
}

func providerHistoryProofCostParsedKeys(t *testing.T, command replayCommand) []replayKey {
	t.Helper()
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal("normal adopted ABI fixture", err)
	}
	parsed, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, command.raw, contract, providerHistoryStateUnitMeter())
	if !ok {
		t.Fatal("original observed key parser precursor")
	}
	owned, ok := replayStateKeysCloneObserved(parsed.keys, providerHistoryStateUnitMeter())
	if !ok {
		t.Fatal("original bounded owned-key precursor")
	}
	return owned
}

func TestProviderHistoryNewKeyProofBoundsFundOriginalSigningAndKEMRoles(t *testing.T) {
	keys := providerHistoryProofCostParsedKeys(t, replayTestCreate(t))
	for _, tc := range []struct {
		all, added []replayKey
		proofs     uint64
	}{
		{nil, nil, 0}, {keys, nil, 0}, {keys[:1], keys[:1], 1}, {keys, keys, 2}, {keys, keys[1:], 1},
	} {
		upper, ok := providerHistoryNewKeyProofUpper(uint64(len(tc.added)), uint64(len(tc.all)))
		if !ok || !replayVerifyNewKeys(replayTestRegistry, replayTestAgent, tc.all, tc.added, 0) {
			t.Fatal("normal original proof precursor")
		}
		meter := providerHistoryContentUnitMeter(upper)
		if !replayVerifyNewKeysObserved(replayTestRegistry, replayTestAgent, tc.all, tc.added, 0, meter) || meter.failed || meter.actual[7] != tc.proofs {
			t.Fatal("prospective envelope did not fund the actual Ed/KEM path", meter.actual, tc.proofs)
		}
		for coordinate, actual := range meter.actual {
			if actual > upper[coordinate] {
				t.Fatal("actual proof exceeded its prospective coordinate", coordinate)
			}
		}
	}
}

func TestProviderHistoryNewKeyProofBoundsIncludeLateSignerSearch(t *testing.T) {
	// Normal RFC8032 public-sample credentials only. Sorted keys place the
	// KEM first and its32-character signing name last; no peer is contacted.
	signer := replayTestKey(t, strings.Repeat("z", 32), false)
	command := replayTestCommand(t, "create", replayTestWallet, 0, map[string]any{
		"controller": replayTestWallet, "keys": []replayKey{replayTestKEM(t, signer), signer},
		"services": []replayService{}, "salt": base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
	})
	keys := providerHistoryProofCostParsedKeys(t, command)
	upper, ok := providerHistoryNewKeyProofUpper(1, 2)
	if !ok || len(keys) != 2 || len(keys[1].Name) != 32 || keys[0].Alg != "x25519" || keys[1].Alg != "ed25519" {
		t.Fatal("normal late-signer fixture precursor")
	}
	meter := providerHistoryContentUnitMeter(upper)
	if !replayVerifyNewKeysObserved(replayTestRegistry, replayTestAgent, keys, keys[:1], 0, meter) || meter.failed || meter.actual[7] != 1 {
		t.Fatal("late retained signer was not funded", meter.actual)
	}
	for coordinate, actual := range meter.actual {
		if actual > upper[coordinate] {
			t.Fatal("late signer exceeded its prospective coordinate", coordinate)
		}
	}
}

func TestProviderHistoryNewKeyProofBoundsKeepIndependentQuotasAndStickyRefusal(t *testing.T) {
	keys := providerHistoryProofCostParsedKeys(t, replayTestCreate(t))
	upper, ok := providerHistoryNewKeyProofUpper(2, 2)
	if !ok {
		t.Fatal("normal prospective two-proof allowance")
	}
	normal := providerHistoryContentUnitMeter(upper)
	if !replayVerifyNewKeysObserved(replayTestRegistry, replayTestAgent, keys, keys, 0, normal) || normal.failed || normal.actual[7] != 2 {
		t.Fatal("normal original two-proof precursor")
	}
	for _, coordinate := range []int{2, 7, 10} {
		// Observed actual is used only to select a UNIT shortage boundary;
		// it is never substituted for the prospective production allowance.
		short := upper
		short[coordinate] = normal.actual[coordinate] - 1
		meter := providerHistoryContentUnitMeter(short)
		if replayVerifyNewKeysObserved(replayTestRegistry, replayTestAgent, keys, keys, 0, meter) || !meter.failed || meter.actual[7] != 1 || meter.actual[2] == 0 || meter.actual[10] == 0 {
			t.Fatal("independent quota did not stop the second actual proof", coordinate, meter.actual)
		}
		for i, actual := range meter.actual {
			if actual > short[i] {
				t.Fatal("rejected event consumed an unadmitted coordinate", coordinate, i)
			}
		}
		before := meter.actual
		if replayVerifyNewKeysObserved(replayTestRegistry, replayTestAgent, keys, keys, 0, meter) || !meter.failed || meter.actual != before {
			t.Fatal("same failed proof observer revived on retry", coordinate)
		}
	}
}

func TestProviderHistoryNewKeyProofBoundsRetainOptionalCountsAndRejectInvalidDomain(t *testing.T) {
	for _, counts := range [][2]uint64{{0, 128}, {128, 0}, {1, 128}, {128, 1}} {
		upper, ok := providerHistoryNewKeyProofUpper(counts[0], counts[1])
		if !ok || upper[2] != 486*counts[0] || upper[7] != counts[0] || upper[10] < 133 {
			t.Fatal("optional cardinality was inferred from successful proofs", counts, upper, ok)
		}
	}
	for _, counts := range [][2]uint64{{129, 0}, {0, 129}, {^uint64(0), 128}, {128, ^uint64(0)}} {
		if upper, ok := providerHistoryNewKeyProofUpper(counts[0], counts[1]); ok || upper != (candidateWorkVector{}) {
			t.Fatal("out-of-domain counts returned partial proof work", counts, upper, ok)
		}
	}
}
