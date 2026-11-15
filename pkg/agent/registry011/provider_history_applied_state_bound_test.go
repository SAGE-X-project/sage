package registry011

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// Only declared copy arguments adjust from the independent64-bit literals.
// These are logical event arguments, not measured heap/backing allocations.
func historyAppliedSerializerLogicalUnits(l providerHistoryStateSerializationLimits, ordinary uint64) uint64 {
	maps, texts, values := 2*l.Grants, 4*l.Grants, 8*l.Grants
	if l.Record {
		maps, texts, values = 2+4*l.Keys+2*l.Services, 12+16*l.Keys+6*l.Services, 12+20*l.Keys+10*l.Services
	}
	return ordinary - maps*8 - texts*16 - values*16 + maps*uint64(reflect.TypeFor[map[string]any]().Size()) +
		texts*uint64(reflect.TypeFor[string]().Size()) + values*uint64(reflect.TypeFor[any]().Size())
}

func TestProviderHistoryAppliedSerializerPartsMatchIndependentRows(t *testing.T) {
	for _, tc := range []struct {
		limits                                    providerHistoryStateSerializationLimits
		normal, envelope, maximum, bytes, decoded uint64
	}{
		{providerHistoryStateSerializationLimits{Record: true}, 3842700, 3842700, 1960, 1960, 0},
		{providerHistoryStateSerializationLimits{Keys: 1, Services: 1, Record: true}, 313227052, 313227052, 17698, 17698, 16},
		{providerHistoryStateSerializationLimits{Keys: 128, Services: 16, Record: true}, 4295314140, 317600220444, 563560, 65536, 2048},
		{providerHistoryStateSerializationLimits{}, 8, 4904, 70, 2, 0},
		{providerHistoryStateSerializationLimits{Grants: 1}, 7615, 277066, 526, 85, 0},
		{providerHistoryStateSerializationLimits{Grants: 128}, 115676421, 3415049256, 58438, 10753, 0},
	} {
		got, ok := providerHistoryAppliedStateSerializationUpper(tc.limits)
		want := providerHistoryAppliedStateSerializationCost{
			normal:   candidateWorkVector{2: tc.decoded, 10: historyAppliedSerializerLogicalUnits(tc.limits, tc.normal)},
			envelope: candidateWorkVector{2: tc.decoded, 10: historyAppliedSerializerLogicalUnits(tc.limits, tc.envelope)},
			maximum:  tc.maximum, normalBytes: tc.bytes,
		}
		if !ok || got != want {
			t.Fatal("applied serializer parts differ from independent rows", tc, got, want, ok)
		}
	}
	for _, l := range []providerHistoryStateSerializationLimits{{Keys: 129, Record: true}, {Services: 17, Record: true}, {Grants: 129},
		{Keys: ^uint64(0), Record: true}, {Grants: 1, Record: true}, {Keys: 1}, {Services: 1}} {
		if got, ok := providerHistoryAppliedStateSerializationUpper(l); ok || got != (providerHistoryAppliedStateSerializationCost{}) {
			t.Fatal("invalid applied serializer DATA exposed a partial row", l, got, ok)
		}
	}
}

func TestProviderHistoryAppliedSerializerPartsPreserveObservedMutationPath(t *testing.T) {
	recordCost, a := providerHistoryAppliedStateSerializationUpper(providerHistoryStateSerializationLimits{Keys: 128, Services: 16, Record: true})
	grantCost, b := providerHistoryAppliedStateSerializationUpper(providerHistoryStateSerializationLimits{Grants: 128})
	contract, _, err := replayABIs()
	if !a || !b || err != nil {
		t.Fatal("prospective complete typed path precursors", err)
	}
	var state replayState
	original := providerHistoryStateUnitMeter()
	for i, step := range replayTestSteps(t) {
		command, parsed := replayParseCommand(replayTestRegistry, replayTestAgent, step.History.Command, contract)
		if !parsed {
			t.Fatal("normal original command precursor")
		}
		got, applied := state.applyObserved(replayTestRegistry, command, uint64(i+1), original)
		if !applied || original.failed || !reflect.DeepEqual(got, step) {
			t.Fatal("original observed state path", i, command.op)
		}
		before, actual := providerHistoryStateUnitSnapshot(t, state), original.actual
		for _, grant := range state.grants {
			if !rpcAddress(grant.Operator) || len(grant.Scope) > 15 || !replayCommandGrantObserved(map[string]any{"operator": grant.Operator, "scope": grant.Scope}, providerHistoryStateUnitMeter()) {
				t.Fatal("actual applied grant lost its original fixed schema")
			}
		}
		recordMeter, grantMeter := providerHistoryContentUnitMeter(recordCost.envelope), providerHistoryContentUnitMeter(grantCost.envelope)
		record, recordOK := replayStateRecordBytesObserved(state.record, recordMeter)
		grants, grantsOK := replayStateGrantsBytesObserved(state.grants, grantMeter)
		if !recordOK || !grantsOK || recordMeter.failed || grantMeter.failed || !bytes.Equal(record, step.Record) || !bytes.Equal(grants, step.Grants) ||
			!reflect.DeepEqual(state, before) || original.actual != actual {
			t.Fatal("parts changed original bytes, state or observer", i)
		}
		for c := range recordCost.normal {
			if recordMeter.actual[c] > recordCost.normal[c] || grantMeter.actual[c] > grantCost.normal[c] {
				t.Fatal("successful original serialization escaped its entered-only part", i, c)
			}
		}
	}
	// An entered-only row is never silently promoted to standalone admission.
	empty, ok := providerHistoryAppliedStateSerializationUpper(providerHistoryStateSerializationLimits{})
	if !ok {
		t.Fatal("empty grant parts")
	}
	short := providerHistoryContentUnitMeter(empty.normal)
	if raw, ok := replayStateGrantsBytesObserved(nil, short); ok || raw != nil || !short.failed {
		t.Fatal("entered-only8 funded the original encode70 preflight")
	}
	complete := providerHistoryContentUnitMeter(empty.envelope)
	if raw, ok := replayStateGrantsBytesObserved(nil, complete); !ok || string(raw) != "[]" || complete.failed || complete.actual[10] != 8 {
		t.Fatal("unchanged original empty-array preflight or actual")
	}
}

func TestProviderHistorySemanticIndexActualPartsKeepStandalonePreflights(t *testing.T) {
	for _, tc := range []struct {
		claims          bool
		entries, rounds uint64
		standalone64    uint64
	}{
		{true, 0, 0, 67109036}, {false, 0, 0, 67109036},
		{true, 1, 0, 67439360}, {false, 1, 0, 67516135},
		{true, 3, 2, 68102024}, {false, 3, 2, 68331661},
		{true, 17, 5, 72749309}, {false, 17, 5, 74045798},
	} {
		copies, headers := (3+tc.rounds)*tc.entries, 2*tc.rounds
		standalone := tc.standalone64 - copies*32 - headers*24 + copies*uint64(reflect.TypeFor[providerHistorySemanticIndexEntry]().Size()) +
			headers*uint64(reflect.TypeFor[[]providerHistorySemanticIndexEntry]().Size())
		want := candidateWorkVector{10: standalone - 67096320}
		got, a := providerHistorySemanticIndexActualUpper(tc.entries, tc.claims)
		old, b := providerHistorySemanticIndexUpper(tc.entries, tc.claims)
		if !a || !b || got != want || old != (candidateWorkVector{10: standalone}) {
			t.Fatal("actual split changed original standalone row or fixed schema padding", tc, got, old)
		}
		data := historySemanticIndexCostData(int(tc.entries))
		meter := providerHistoryContentUnitMeter(old)
		if root := providerHistorySemanticIndexRootObserved(data, tc.claims, max(1, int(tc.entries)), meter); root == "" || meter.failed || meter.actual[10] > got[10] {
			t.Fatal("original entered index work escaped independent actual DATA", tc, meter.actual)
		}
	}
	for _, claims := range []bool{false, true} {
		for _, entries := range []uint64{1<<20 + 1, ^uint64(0)} {
			if got, ok := providerHistorySemanticIndexActualUpper(entries, claims); ok || got != (candidateWorkVector{}) {
				t.Fatal("invalid index actual DATA became a partial row", entries, claims)
			}
		}
	}
	actualOnly, ok := providerHistorySemanticIndexActualUpper(0, true)
	if !ok {
		t.Fatal("normal empty index actual DATA")
	}
	meter := providerHistoryContentUnitMeter(actualOnly)
	if root := providerHistorySemanticIndexRootObserved(historySemanticIndexCostData(0), true, 1, meter); root != "" || !meter.failed {
		t.Fatal("actual-only index DATA supplied missing original encode8192 headroom")
	}
}

func TestProviderHistoryAppliedStreamPartsMatchLiteralRowsAndMetadataSchema(t *testing.T) {
	recordLimits := providerHistoryStateSerializationLimits{Keys: 128, Services: 16, Record: true}
	grantLimits := providerHistoryStateSerializationLimits{Grants: 128}
	normalState := historyAppliedSerializerLogicalUnits(recordLimits, 4295314140) + historyAppliedSerializerLogicalUnits(grantLimits, 115676421)
	envelopeState := historyAppliedSerializerLogicalUnits(recordLimits, 317600220444) + historyAppliedSerializerLogicalUnits(grantLimits, 3415049256)
	for _, tc := range []struct{ entries, rounds, normal, envelope uint64 }{
		{0, 0, 4411579683, 321216674517}, {1, 0, 4412317106, 321217411940},
		{3, 2, 4413795296, 321218890130}, {17, 5, 4424156718, 321229251552},
	} {
		copies, headers := 2*(3+tc.rounds)*tc.entries, 4*tc.rounds
		// Remove the ordinary state components from each full literal BEFORE
		// adjusting index copies. Every subtraction has a positive remainder;
		// no unsigned wrap is used to represent a negative portability delta.
		normal := normalState + (tc.normal - 4410990561) - copies*32 - headers*24 + copies*uint64(reflect.TypeFor[providerHistorySemanticIndexEntry]().Size()) +
			headers*uint64(reflect.TypeFor[[]providerHistorySemanticIndexEntry]().Size())
		envelope := envelopeState + (tc.envelope - 321015269700) - copies*32 - headers*24 + copies*uint64(reflect.TypeFor[providerHistorySemanticIndexEntry]().Size()) +
			headers*uint64(reflect.TypeFor[[]providerHistorySemanticIndexEntry]().Size())
		got, ok := providerHistoryAppliedStreamRootUpper(tc.entries, tc.entries)
		want := providerHistoryAppliedStreamRootCost{candidateWorkVector{2: 2048, 10: normal}, candidateWorkVector{2: 2048, 10: envelope}}
		if !ok || got != want {
			t.Fatal("applied stream original closure differs from independent literals", tc, got, want, ok)
		}
	}
	for _, invalid := range [][2]uint64{{1<<20 + 1, 0}, {0, 1<<20 + 1}, {^uint64(0), 0}, {0, ^uint64(0)}} {
		if got, ok := providerHistoryAppliedStreamRootUpper(invalid[0], invalid[1]); ok || got != (providerHistoryAppliedStreamRootCost{}) {
			t.Fatal("invalid stream component returned a partial closure", invalid, got, ok)
		}
	}
	// Fixed scalar DATA only. All generated hash/decimal fields use their
	// full text widths; this does not authenticate a selected chain header.
	value := map[string]any{"version": "18446744073709551615", "admitted": "18446744073709551615",
		"record": admissionTestDigest("a"), "grants": admissionTestDigest("b"), "claims": admissionTestDigest("c"), "created": admissionTestDigest("d"),
		"selected": providerHistoryCollectionHeaderObject(FinalizedHeader{admissionTestDigest("e"), admissionTestDigest("f"), ^uint64(0), ^uint64(0)})}
	meter := providerHistorySemanticUnitMeter()
	if root := providerHistoryCollectionValueDigestObserved("semantic-state", value, meter); root == "" || meter.failed ||
		len(providerReplayCanonical(value, candidateLedgerMetadataLimit, nil)) != 697 || meter.actual != (candidateWorkVector{10: 486565}) {
		t.Fatal("closed semantic-state schema/event upper changed", meter.actual)
	}
}

func TestProviderHistoryAppliedStreamPartsPreserveOriginalFoldObservers(t *testing.T) {
	for _, mode := range []string{"no-window", "pending", "queued", "confirmed-and-working"} {
		f, original, _ := historySemanticSnapshotCostFixture(t, mode)
		config, beforeWork := f.selection.config, original.actual
		parts, ok := providerHistoryAppliedStreamRootUpper(uint64(config.MaxClaims), uint64(config.MaxIdentities))
		if !ok {
			t.Fatal("original prospective stream component counts")
		}
		streams := []replayStream{f.confirmed}
		if f.window != nil {
			streams = append(streams, replayStream{state: f.window.run.state, claims: f.window.run.claims, created: f.window.run.created})
		}
		for _, stream := range streams {
			before := providerHistoryStateUnitSnapshot(t, stream.state)
			want := providerHistorySemanticStreamRootObserved(stream, config, providerHistorySemanticUnitMeter())
			meter := providerHistoryContentUnitMeter(parts.envelope)
			root := providerHistorySemanticStreamRootObserved(stream, config, meter)
			if want == "" || root != want || meter.failed || !reflect.DeepEqual(before, stream.state) || original.actual != beforeWork || f.observer != original || !f.valid() {
				t.Fatal("read-only composition changed original state/root/observer", mode)
			}
			for c := range parts.normal {
				if meter.actual[c] > parts.normal[c] {
					t.Fatal("normal original stream escaped entered-only row", mode, c, meter.actual[c], parts.normal[c])
				}
			}
		}
	}
}

func TestProviderHistoryAppliedStreamPartsCoverDeferredSizeRefusal(t *testing.T) {
	// Bounded in-memory serializer DATA only, outside a credential/Source or
	// network path. Clone text/count ceilings hold, but six-byte JSON escapes
	// make the original record output cap refuse AFTER encode(actual).
	state := replayState{version: 1, admitted: 1, record: replayRecord{ID: "unit", State: "active", Version: "1",
		Services: make([]replayService, 16)}, grants: []replayGrant{{Operator: replayTestWallet, Scope: "update-services"}}}
	for i := range state.record.Services {
		state.record.Services[i] = replayService{Name: "unit", Type: "unit", URI: strings.Repeat("\x01", 2048)}
	}
	if _, ok := replayStateCloneObserved(state, providerHistoryStateUnitMeter()); !ok {
		t.Fatal("typed clone ceiling precursor")
	}
	stream := replayStream{state: state}
	before := providerHistoryStateUnitSnapshot(t, state)
	parts, ok := providerHistoryAppliedStreamRootUpper(0, 0)
	if !ok {
		t.Fatal("complete deferred stream closure precursor")
	}
	bytesMeter := providerHistoryContentUnitMeter(parts.envelope)
	record, grants, encoded := stream.stateBytesObserved(bytesMeter)
	if encoded || record != nil || len(grants) == 0 || bytesMeter.failed || bytesMeter.actual[10] == 0 || !reflect.DeepEqual(before, stream.state) {
		t.Fatal("late size refusal did not retain the second serializer's entered work", bytesMeter.actual)
	}
	// Expand the deferred original calls independently of the quoted formula.
	// A false record result does not short-circuit these siblings or poison
	// this inert read-only DATA meter; the owning fold's refusal is separate.
	deferred := providerHistoryContentUnitMeter(parts.envelope)
	if _, ok := providerHistorySemanticDecimalObserved(1, deferred); !ok {
		t.Fatal("deferred version")
	}
	if _, ok := providerHistorySemanticDecimalObserved(1, deferred); !ok ||
		providerHistorySemanticIndexRootObserved(stream, true, 1, deferred) == "" || providerHistorySemanticIndexRootObserved(stream, false, 1, deferred) == "" {
		t.Fatal("deferred admitted and index roots")
	}
	meter := providerHistoryContentUnitMeter(parts.envelope)
	if root := providerHistorySemanticStreamRootObserved(stream, JournalConfig{MaxClaims: 1, MaxIdentities: 1}, meter); root != "" || meter.failed ||
		!reflect.DeepEqual(before, stream.state) || meter.actual[10] <= bytesMeter.actual[10] {
		t.Fatal("failed stream returned a root, skipped deferred work or changed DATA", meter.actual)
	}
	for c, actual := range meter.actual {
		if actual != bytesMeter.actual[c]+deferred.actual[c] || actual > parts.envelope[c] {
			t.Fatal("deferred original caller closure escaped its envelope", c, actual)
		}
	}
	// Separate ordinary typed DATA isolates the real decoded-byte guard.
	expires := uint64(1<<53 - 1)
	recordData := replayRecord{Keys: make([]replayKey, 128)}
	for i := range recordData.Keys {
		recordData.Keys[i].Expires = &expires
	}
	serializer, ok := providerHistoryAppliedStateSerializationUpper(providerHistoryStateSerializationLimits{Keys: 128, Record: true})
	if !ok {
		t.Fatal("prospective typed decoded row")
	}
	short := serializer.envelope
	short[2]--
	refused := providerHistoryContentUnitMeter(short)
	if raw, ok := replayStateRecordBytesObserved(recordData, refused); ok || raw != nil || !refused.failed || refused.actual[2] != 16*127 || refused.actual[10] == 0 {
		t.Fatal("independent decoded quota was replaced by ample logical work", refused.actual)
	}
	actual := refused.actual
	if raw, ok := replayStateRecordBytesObserved(recordData, refused); ok || raw != nil || refused.actual != actual || !refused.failed {
		t.Fatal("failed original observer revived or reset its consumed prefix")
	}
}
