package registry011

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Only fresh arithmetic UNIT observers are used. An amount below the charged
// normal path cannot fund completion; its last successful event must remain
// charged, and an unavailable result must not expose a partially built value.
// The normal charged amount is not asserted to fund incoming preflight uppers.
func historyUnitConversionRefusalPrefixes(t *testing.T, run func(*providerWorkMeter) (bool, bool)) {
	t.Helper()
	baseline := providerReplayUnitMeter()
	completed, closed := run(baseline)
	if !completed || !closed || baseline.failed || baseline.actual[10] == 0 || baseline.actual[10] > 65536 {
		t.Fatal("bounded normal conversion precursor did not complete", baseline.actual)
	}
	for fuel := uint64(0); fuel < baseline.actual[10]; fuel++ {
		m := providerReplayUnitMeter()
		m.upper[10] = fuel
		ok, closed := run(m)
		if ok || !closed || !m.failed || m.actual[10] > fuel {
			t.Fatal("unfunded conversion exposed a result or exceeded its original prefix", fuel, m.actual)
		}
		prefix := m.actual
		if ok, closed := run(m); ok || !closed || m.actual != prefix {
			t.Fatal("refused conversion resumed its original observer", fuel, m.actual, prefix)
		}
	}
}

func TestProviderHistoryFrameLogConversionKeepsInputsAndRefusesPartialValues(t *testing.T) {
	input := ReplayFrameLog{
		Address: replayTestWallet,
		Topics:  []common.Hash{{1}, {2}},
		Data:    []byte("ordinary log data"),
	}
	before := replayTestClone(t, input)
	want := &types.Log{Address: common.HexToAddress(input.Address), Topics: input.Topics, Data: input.Data}
	historyUnitConversionRefusalPrefixes(t, func(m *providerWorkMeter) (bool, bool) {
		out, err := replayFrameLogObserved(&input, m)
		if !reflect.DeepEqual(input, before) || err == nil && !reflect.DeepEqual(out, want) {
			t.Fatal("log conversion changed its original fields or normal value", err)
		}
		return err == nil, err == nil || out == nil
	})
	out, err := replayFrameLogObserved(&input, providerReplayUnitMeter())
	if err != nil || out == nil {
		t.Fatal("normal owned log precursor", err)
	}
	out.Topics[0][0] ^= 1
	out.Data[0] ^= 1
	if !reflect.DeepEqual(input, before) {
		t.Fatal("returned log storage aliases its original input")
	}
}

func TestProviderHistoryObservedLogGrowthKeepsCommittedRowsOnRefusal(t *testing.T) {
	size := uint64(reflect.TypeFor[replayLog]().Size())
	first := replayLog{log: &types.Log{Data: []byte("retained")}, eligible: true}
	second := replayLog{log: &types.Log{Data: []byte("next")}}
	for _, tc := range []struct {
		name string
		fuel uint64
		work uint64
	}{
		{"before-new-storage", 2 * size, 0},
		{"before-old-row-copy", 2*size + 1, 2*size + 1},
		{"before-new-row-copy", 3*size + 2, 3*size + 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := providerReplayUnitMeter()
			m.upper[10] = tc.fuel
			original := []replayLog{first}
			out := replayObservedLogs{rows: original, maximum: 2, meter: m}
			if out.append(second) || !m.failed || m.actual[10] != tc.work || len(out.rows) != 1 ||
				!reflect.DeepEqual(out.rows[0], first) || !reflect.DeepEqual(original, []replayLog{first}) {
				t.Fatal("refused growth lost or published a row", m.actual, out.rows)
			}
			prefix, rows := m.actual, append([]replayLog(nil), out.rows...)
			if out.append(second) || m.actual != prefix || !reflect.DeepEqual(out.rows, rows) {
				t.Fatal("refused log growth resumed or changed retained rows")
			}
		})
	}
	m := providerReplayUnitMeter()
	m.upper[10] = 4*size + 3
	out := replayObservedLogs{rows: []replayLog{first}, maximum: 2, meter: m}
	if !out.append(second) || m.failed || m.actual[10] != 4*size+3 || !reflect.DeepEqual(out.rows, []replayLog{first, second}) {
		t.Fatal("exact normal growth did not retain ordered rows", m.actual)
	}
}

func TestProviderHistoryCommandConversionsRetainNoUnfundedPartialTuple(t *testing.T) {
	create := replayTestCreate(t)
	expires := uint64(1000)
	keys := replayTestClone(t, create.keys)
	keys[0].Expires = &expires
	before := replayTestClone(t, keys)
	services := []replayService{{Name: "api", Type: "A2A", URI: "https://example.invalid/api"}}
	for _, tc := range []struct {
		name string
		run  func(*providerWorkMeter) (bool, bool)
	}{
		{"keys", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayCommandABIKeysObserved(keys, m)
			if !reflect.DeepEqual(keys, before) {
				t.Fatal("key tuple conversion changed its original input")
			}
			if ok {
				if len(out) != len(keys) || !out[0].HasExpiry || out[0].Expires != expires {
					t.Fatal("normal tuple lost its expiry or keys")
				}
				for i, key := range keys {
					material, a := replayBinary(key.Key, 32)
					proof, b := replayBinary(key.Proof.Value, 64)
					if !a || !b || out[i].Name != key.Name || out[i].Alg != key.Alg || out[i].Signer != key.Proof.Signer ||
						!bytes.Equal(out[i].Key, material) || !bytes.Equal(out[i].Proof, proof) {
						t.Fatal("normal tuple changed key material or proof", i)
					}
				}
			}
			return ok, ok || out == nil
		}},
		{"services", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayCommandABIServicesObserved(services, m)
			if ok && !reflect.DeepEqual(out, []replayABIService{{Name: "api", ServiceType: "A2A", URI: "https://example.invalid/api"}}) {
				t.Fatal("normal service tuple changed its fields")
			}
			return ok, ok || out == nil
		}},
		{"challenge", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayChallengeObserved(replayTestRegistry, replayTestAgent, keys[0], m)
			if ok && !bytes.Equal(out, replayChallenge(replayTestRegistry, replayTestAgent, keys[0])) {
				t.Fatal("normal challenge changed its adopted framing")
			}
			return ok, ok || out == nil
		}},
		{"address-topic", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayAddressTopicObserved(replayTestWallet, m)
			want := common.BytesToHash(common.HexToAddress(replayTestWallet).Bytes())
			if ok && out != want {
				t.Fatal("normal address topic changed its padding")
			}
			return ok, ok || out == (common.Hash{})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) { historyUnitConversionRefusalPrefixes(t, tc.run) })
	}
}

func TestProviderHistoryGeneratedArgumentsRejectWrongDecodedTypes(t *testing.T) {
	// These are local tuple-conversion DATA values, without a sender, signed
	// request or dispatch. A wrong type must not become a partial command.
	for _, tc := range []struct {
		name, operation string
		values          []any
	}{
		{"unknown-operation", "unknown", nil},
		{"wrong-arity", "activate", []any{replayTestAgent}},
		{"add-key-tuple", "add-key", []any{replayTestAgent, uint64(1), true}},
		{"revoke-key-name", "revoke-key", []any{replayTestAgent, uint64(1), true}},
		{"service-tuple", "update-services", []any{replayTestAgent, uint64(1), true}},
		{"authorize-address", "authorize-operator", []any{replayTestAgent, uint64(1), replayTestOperator, "activate"}},
		{"authorize-scope", "authorize-operator", []any{replayTestAgent, uint64(1), common.HexToAddress(replayTestOperator), true}},
		{"revoke-address", "revoke-operator", []any{replayTestAgent, uint64(1), replayTestOperator, "activate"}},
		{"revoke-scope", "revoke-operator", []any{replayTestAgent, uint64(1), common.HexToAddress(replayTestOperator), true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := append([]any(nil), tc.values...)
			m := providerReplayUnitMeter()
			if out, ok := replayGeneratedArgumentsObserved(tc.operation, tc.values, m); ok || out != nil ||
				!reflect.DeepEqual(tc.values, before) || m.actual[7] != 0 {
				t.Fatal("wrong decoded type produced a partial tuple or changed its input", m.actual)
			}
		})
	}
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	command := replayTestCommand(t, "activate", replayTestWallet, 1, map[string]any{})
	values := []any{replayTestAgent, "1"}
	if out, ok := replayDecodedCommandObserved(replayTestRegistry, command.actor, "activate", values, command.calldata, contract, providerReplayUnitMeter()); ok || !reflect.DeepEqual(out, replayCommand{}) || values[1] != "1" {
		t.Fatal("string version became a typed command")
	}
}
