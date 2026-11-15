package registry011

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Fixed byte-stage UNIT subjects do not carry a transaction, permit, selected
// reader, signature authorization or native resource qualification. Keeping
// these stages private lets their own failure paths be checked independently
// of the larger incoming preflight, which remains in every original caller.
func TestProviderHistoryEncodedHashStageKeepsChargedPrefixOnRefusal(t *testing.T) {
	raw := []byte{0xc2, 1, 2}
	want := crypto.Keccak256Hash(raw)
	for _, tc := range []struct{ fuel, charged uint64 }{{0, 0}, {10, 10}, {14, 14}} {
		meter := providerReplayUnitMeter()
		meter.upper[10] = tc.fuel
		got, ok := nodeEncodedHashObserved(raw, meter)
		if tc.fuel == 14 {
			if !ok || got != want || meter.failed {
				t.Fatal("complete encoding and hash stage")
			}
		} else {
			if ok || got != (common.Hash{}) || !meter.failed {
				t.Fatal("unfunded encoding/hash exposed a partial value")
			}
			before := meter.actual
			if got, ok := nodeEncodedHashObserved(raw, meter); ok || got != (common.Hash{}) || meter.actual != before {
				t.Fatal("failed byte stage resumed its original observer")
			}
		}
		if meter.actual[10] != tc.charged || !bytes.Equal(raw, []byte{0xc2, 1, 2}) {
			t.Fatal("encoding/hash prefix or input changed", meter.actual)
		}
	}
}

func TestProviderHistorySigningPreimageStagesRejectPartialTypedValues(t *testing.T) {
	raw := []byte{0xc2, 1, 2}
	for _, kind := range []byte{types.LegacyTxType, types.DynamicFeeTxType} {
		wantBytes := bytes.Clone(raw)
		funding := []uint64{0, 10, 14}
		if kind != types.LegacyTxType {
			wantBytes = append([]byte{kind}, wantBytes...)
			funding = []uint64{0, 10, 15, 20}
		}
		for i, fuel := range funding {
			meter := providerReplayUnitMeter()
			meter.upper[10] = fuel
			got, err := nodeSigningPreimageHashObserved(raw, kind, meter)
			if i == len(funding)-1 {
				if err != nil || got != crypto.Keccak256Hash(wantBytes) || meter.failed {
					t.Fatal("normal typed signing-preimage stage", kind, err)
				}
			} else {
				if err != ErrReplayUnavailable || got != (common.Hash{}) || !meter.failed || meter.actual[10] != fuel {
					t.Fatal("unfunded signing stage exposed a partial hash or lost its charged prefix", kind, fuel, err, meter.actual)
				}
				charged := meter.actual
				if got, err := nodeSigningPreimageHashObserved(raw, kind, meter); err == nil || got != (common.Hash{}) || meter.actual != charged {
					t.Fatal("failed signing stage renewed its observer")
				}
			}
			if !bytes.Equal(raw, []byte{0xc2, 1, 2}) {
				t.Fatal("typed prefix changed the original untyped bytes")
			}
		}
	}
}

func TestProviderHistoryRecoveredPublicKeyStageReturnsNoUnfundedAddress(t *testing.T) {
	// The public secp256k1 generator point, without any private key or signing.
	public, err := hex.DecodeString("0479be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8")
	if err != nil || len(public) != 65 {
		t.Fatal("fixed public point")
	}
	digest := crypto.Keccak256Hash(public[1:])
	want := common.BytesToAddress(digest[12:])
	for _, fuel := range []uint64{0, 65, 86} {
		meter := providerReplayUnitMeter()
		meter.upper[10] = fuel
		got, err := nodeRecoveredPublicKeyAddressObserved(public, meter)
		if fuel == 86 {
			if err != nil || got != want || meter.failed || meter.actual[10] != 86 {
				t.Fatal("normal public-point address derivation", err)
			}
			continue
		}
		if err != ErrReplayUnavailable || got != (common.Address{}) || !meter.failed || meter.actual[10] != fuel {
			t.Fatal("unfunded public-key stage returned an address", fuel, err, meter.actual)
		}
		charged := meter.actual
		if got, err := nodeRecoveredPublicKeyAddressObserved(public, meter); err == nil || got != (common.Address{}) || meter.actual != charged {
			t.Fatal("failed address stage resumed")
		}
	}
	for _, invalid := range [][]byte{nil, public[:64], append([]byte{5}, public[1:]...)} {
		meter := providerReplayUnitMeter()
		if got, err := nodeRecoveredPublicKeyAddressObserved(invalid, meter); err != ErrReplay || got != (common.Address{}) || meter.actual != (candidateWorkVector{}) {
			t.Fatal("invalid public-point representation entered hashing", err)
		}
	}
}

func TestProviderHistoryCommandConsumerRejectsInvalidNestedArgumentsWithoutPartialOutput(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	steps := replayTestSteps(t)
	for _, tc := range []struct {
		name string
		step int
		edit func(map[string]any)
	}{
		{"create-key", 0, func(args map[string]any) { args["keys"] = []any{} }},
		{"create-service", 0, func(args map[string]any) {
			args["services"] = []any{map[string]any{"name": "api", "type": "A2A", "uri": "file:///unavailable"}}
		}},
		{"create-salt", 0, func(args map[string]any) { args["salt"] = "" }},
		{"added-key", 5, func(args map[string]any) { args["key"].(map[string]any)["alg"] = "unsupported-unit-algorithm" }},
		{"revoked-key-name", 8, func(args map[string]any) { args["name"] = "invalid.name" }},
		{"updated-service", 7, func(args map[string]any) {
			args["services"] = []any{map[string]any{"name": "api", "type": "A2A", "uri": "file:///unavailable"}}
		}},
		{"operator-scope", 1, func(args map[string]any) { args["scope"] = "unsupported-unit-scope" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := bytes.Clone(steps[tc.step].History.Command)
			if _, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, original, contract, providerHistoryExecutionUnitMeter()); !ok {
				t.Fatal("ordinary complete command precursor")
			}
			raw := contentEdit(t, original, func(object map[string]any) { tc.edit(object["arguments"].(map[string]any)) })
			if replayCommandJSONObserved(raw, providerHistoryExecutionUnitMeter()) == nil {
				t.Fatal("nested refusal did not reach the canonical command grammar")
			}
			got, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, raw, contract, providerHistoryExecutionUnitMeter())
			if ok || !reflect.DeepEqual(got, replayCommand{}) || !bytes.Equal(original, steps[tc.step].History.Command) {
				t.Fatal("nested argument refusal exposed a partial command or changed its original")
			}
			if _, accepted := replayParseCommand(replayTestRegistry, replayTestAgent, raw, contract); accepted {
				t.Fatal("the original strict parser accepted the invalid nested argument")
			}
		})
	}
}

func TestProviderHistorySemanticIndexRefusalsKeepOriginalMaps(t *testing.T) {
	stream := historySemanticIndexCostData(1)
	storage := 2*uint64(reflect.TypeFor[providerHistorySemanticIndexEntry]().Size()) + 1
	actor := uint64(len(replayTestWallet) + 1)
	var created string
	for key := range stream.created {
		created = key
	}
	before := replayTestClone(t, stream.created)
	for _, tc := range []struct {
		name   string
		claims bool
		fuel   uint64
	}{
		{"claim-actor-copy", true, storage + actor},
		{"claim-key-join", true, storage + 2*actor + 64*64 + 1},
		{"created-key-copy", false, storage + uint64(len(created)+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if providerHistorySemanticIndexRootObserved(stream, tc.claims, 1, providerReplayUnitMeter()) == "" {
				t.Fatal("ordinary one-entry index precursor")
			}
			meter := providerReplayUnitMeter()
			meter.upper[10] = tc.fuel
			if root := providerHistorySemanticIndexRootObserved(stream, tc.claims, 1, meter); root != "" || !meter.failed || meter.actual[10] != tc.fuel ||
				!reflect.DeepEqual(stream.created, before) || len(stream.claims) != 1 {
				t.Fatal("unfunded index construction exposed a root or changed original maps", tc.name, meter.actual)
			}
			charged := meter.actual
			if root := providerHistorySemanticIndexRootObserved(stream, tc.claims, 1, meter); root != "" || meter.actual != charged {
				t.Fatal("failed index construction resumed")
			}
		})
	}
}

func TestProviderHistoryStateApplyCloneRefusalPreservesUnpublishedState(t *testing.T) {
	command := replayTestCreate(t)
	// Fund exactly the initial subject/registry comparisons. The original
	// state-copy event cannot be funded; no private working mutation starts.
	comparisons := uint64(len(command.agent) + 1 + 2*len(replayTestRegistry) + 1)
	meter := providerReplayUnitMeter()
	meter.upper[10] = comparisons
	var state replayState
	out, ok := state.applyObserved(replayTestRegistry, command, 1, meter)
	if ok || !reflect.DeepEqual(out, HistoryContentStep{}) || !reflect.DeepEqual(state, replayState{}) || !meter.failed || meter.actual[10] != comparisons {
		t.Fatal("unfunded state clone exposed a partial history or changed state", meter.actual)
	}
	charged := meter.actual
	if _, ok := state.applyObserved(replayTestRegistry, command, 1, meter); ok || meter.actual != charged || !reflect.DeepEqual(state, replayState{}) {
		t.Fatal("failed state observer resumed or published a mutation")
	}
}

func TestProviderHistoryCommandConsumerRejectsZeroVersionWithoutPartialOutput(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	original := replayTestCommand(t, "activate", replayTestWallet, 1, map[string]any{})
	if _, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, original.raw, contract, providerHistoryExecutionUnitMeter()); !ok {
		t.Fatal("ordinary versioned command precursor")
	}
	raw := contentEdit(t, original.raw, func(object map[string]any) { object["expectedVersion"] = "0" })
	m := providerHistoryExecutionUnitMeter()
	got, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, raw, contract, m)
	if ok || !reflect.DeepEqual(got, replayCommand{}) || m.failed || m.actual[0] != 0 || m.actual[1] != 0 || m.clocks != 0 {
		t.Fatal("zero expected version produced partial command DATA or a live effect")
	}
}
