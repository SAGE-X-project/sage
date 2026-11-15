package registry011

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

func TestProviderHistoryStateSerializationBoundsMatchHandwrittenBuilderWalkerRows(t *testing.T) {
	for _, tc := range []struct {
		limits  providerHistoryStateSerializationLimits
		units64 uint64
	}{
		{providerHistoryStateSerializationLimits{Record: true}, 2474348400818},
		{providerHistoryStateSerializationLimits{Keys: 1, Services: 1, Record: true}, 2474801448976},
		{providerHistoryStateSerializationLimits{Keys: 128, Services: 16, Record: true}, 2517725493090},
		{providerHistoryStateSerializationLimits{}, 4904},
		{providerHistoryStateSerializationLimits{Grants: 1}, 2474216269832},
		{providerHistoryStateSerializationLimits{Grants: 128}, 2486216802600},
	} {
		l := tc.limits
		maps, texts, values := 2*l.Grants, 4*l.Grants, 8*l.Grants
		if l.Record {
			maps, texts, values = 2+4*l.Keys+2*l.Services, 12+16*l.Keys+6*l.Services, 12+20*l.Keys+10*l.Services
		}
		// Handwritten constants use map8/string16/any16. Only the explicit
		// copy bytes adjust for the compiled target; node/byte/event counts
		// and encode/decimal preflight constants remain independent literals.
		want := tc.units64 - maps*8 - texts*16 - values*16
		want += maps*uint64(reflect.TypeFor[map[string]any]().Size()) + texts*uint64(reflect.TypeFor[string]().Size()) + values*uint64(reflect.TypeFor[any]().Size())
		got, ok := providerHistoryStateSerializationUpper(l)
		decoded := uint64(0)
		if l.Record {
			decoded = 16 * l.Keys
		}
		if !ok || got != (candidateWorkVector{2: decoded, 10: want}) {
			t.Fatal("typed serialization differs from handwritten builder/walker rows", l, got, want, ok)
		}
	}
}

func TestProviderHistoryStateSerializationBoundsFundOriginalTypedStateBytes(t *testing.T) {
	for _, version := range []int{1, 3, 5, 8, 11} {
		state := providerHistoryStateUnitPrefix(t, version)
		expires := uint64(1<<53 - 1)
		state.record.Keys[0].Expires = &expires
		// The original generated arrays normalize nil slices to empty arrays.
		record := state.record
		record.Keys = append([]replayKey{}, record.Keys...)
		record.Services = append([]replayService{}, record.Services...)
		wantRecord, a := rfc8785.Marshal(record)
		wantGrants, b := rfc8785.Marshal(append([]replayGrant{}, state.grants...))
		if a != nil || b != nil {
			t.Fatal("independent canonical typed-data precursor", a, b)
		}
		for _, isRecord := range []bool{true, false} {
			l := providerHistoryStateSerializationLimits{Grants: 128}
			want := wantGrants
			if isRecord {
				l = providerHistoryStateSerializationLimits{Keys: 128, Services: 16, Record: true}
				want = wantRecord
			}
			upper, ok := providerHistoryStateSerializationUpper(l)
			if !ok {
				t.Fatal("prospective typed serialization upper")
			}
			meter := providerHistoryContentUnitMeter(upper)
			var raw []byte
			if isRecord {
				raw, ok = replayStateRecordBytesObserved(state.record, meter)
			} else {
				raw, ok = replayStateGrantsBytesObserved(state.grants, meter)
			}
			if !ok || meter.failed || !bytes.Equal(raw, want) {
				t.Fatal("original typed JSON differs or its actual preflight was unfunded", version, isRecord)
			}
			for coordinate, actual := range meter.actual {
				if actual > upper[coordinate] {
					t.Fatal("typed serialization exceeded scoped prospective row", version, isRecord, coordinate)
				}
			}
		}
	}
}

func TestProviderHistoryStateSerializationBoundsIncludeStringsBeyondCloneFieldLimits(t *testing.T) {
	// Ordinary serializer-only DATA, not a valid DID/credential or a Source.
	// These original primitives do not apply clone-specific field ceilings.
	recordUpper, a := providerHistoryStateSerializationUpper(providerHistoryStateSerializationLimits{Record: true})
	grantUpper, b := providerHistoryStateSerializationUpper(providerHistoryStateSerializationLimits{Grants: 1})
	if !a || !b {
		t.Fatal("prospective serializer-only rows")
	}
	recordMeter, grantMeter := providerHistoryContentUnitMeter(recordUpper), providerHistoryContentUnitMeter(grantUpper)
	record := replayRecord{ID: strings.Repeat("a", 4096)}
	grants := []replayGrant{{Operator: strings.Repeat("a", 4096), Scope: "unit"}}
	recordRaw, a := replayStateRecordBytesObserved(record, recordMeter)
	grantRaw, b := replayStateGrantsBytesObserved(grants, grantMeter)
	if !a || !b || recordMeter.failed || grantMeter.failed || len(recordRaw) <= 4096 || len(grantRaw) <= 4096 || recordMeter.actual[10] > recordUpper[10] || grantMeter.actual[10] > grantUpper[10] {
		t.Fatal("clone field assumptions leaked into successful serializer envelope")
	}
	// Safe in-memory DATA only. The output-cap refusal occurs after the
	// original walker/encode, and must not escape its prospective allowance.
	refused := providerHistoryContentUnitMeter(recordUpper)
	raw, ok := replayStateRecordBytesObserved(replayRecord{ID: strings.Repeat("a", 65536)}, refused)
	if ok || raw != nil || refused.actual[10] == 0 || refused.actual[10] > recordUpper[10] || refused.actual != (candidateWorkVector{10: refused.actual[10]}) {
		t.Fatal("late output-size refusal escaped the entered serialization envelope")
	}
}

func TestProviderHistoryStateSerializationBoundsKeepOriginalCanonicalPreflightRoom(t *testing.T) {
	baseline := providerHistoryContentUnitMeter(candidateWorkVector{10: 1 << 60})
	raw, ok := replayStateGrantsBytesObserved(nil, baseline)
	if !ok || string(raw) != "[]" || baseline.failed || baseline.actual != (candidateWorkVector{10: 8}) {
		t.Fatal("empty grant array original events precursor", baseline.actual)
	}
	actualOnly := providerHistoryContentUnitMeter(baseline.actual)
	if raw, ok := replayStateGrantsBytesObserved(nil, actualOnly); ok || raw != nil || !actualOnly.failed {
		t.Fatal("actual-only allowance incorrectly funded original encode70 preflight")
	}
	upper, ok := providerHistoryStateSerializationUpper(providerHistoryStateSerializationLimits{})
	if !ok || upper != (candidateWorkVector{10: 4904}) {
		t.Fatal("empty array prospective encode70 row", upper)
	}
	bounded := providerHistoryContentUnitMeter(upper)
	if raw, ok := replayStateGrantsBytesObserved(nil, bounded); !ok || string(raw) != "[]" || bounded.failed || bounded.actual != baseline.actual {
		t.Fatal("prospective quote did not fund unchanged canonical preflight")
	}
}

func TestProviderHistoryStateSerializationBoundsRefuseInvalidCardinalitiesAndKinds(t *testing.T) {
	for _, l := range []providerHistoryStateSerializationLimits{{Keys: 129, Record: true}, {Services: 17, Record: true}, {Grants: 129},
		{Keys: ^uint64(0), Record: true}, {Services: ^uint64(0), Record: true}, {Grants: ^uint64(0)},
		{Grants: 1, Record: true}, {Keys: 1}, {Services: 1}} {
		if got, ok := providerHistoryStateSerializationUpper(l); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid serializer plan returned a partial upper", l, got, ok)
		}
	}
}

func TestProviderHistoryStateSerializationBoundsFundExpiryDecodedBytes(t *testing.T) {
	// Ordinary typed serializer DATA, never a valid credential or Source.
	for _, count := range []int{1, 128} {
		expires := uint64(1<<53 - 1)
		record := replayRecord{Keys: make([]replayKey, count), Services: []replayService{}}
		for i := range record.Keys {
			record.Keys[i].Expires = &expires
		}
		want, err := rfc8785.Marshal(record)
		if err != nil {
			t.Fatal("independent canonical expiry-byte precursor", err)
		}
		upper, ok := providerHistoryStateSerializationUpper(providerHistoryStateSerializationLimits{Keys: uint64(count), Record: true})
		if !ok || upper[2] != 16*uint64(count) {
			t.Fatal("prospective expiry Number decoded-byte row", count, upper)
		}
		normal := providerHistoryContentUnitMeter(upper)
		if raw, ok := replayStateRecordBytesObserved(record, normal); !ok || !bytes.Equal(raw, want) || normal.failed || normal.actual[2] != 16*uint64(count) || normal.actual[10] > upper[10] {
			t.Fatal("normal original expiry decoding precursor", count, normal.actual)
		}
		short := upper
		short[2]--
		meter := providerHistoryContentUnitMeter(short)
		if raw, ok := replayStateRecordBytesObserved(record, meter); ok || raw != nil || !meter.failed || meter.actual[2] != 16*uint64(count-1) || meter.actual[10] == 0 || meter.actual[10] > upper[10] {
			t.Fatal("independent decoded-byte shortage was hidden by ample logical work", count, meter.actual)
		}
		before := meter.actual
		if raw, ok := replayStateRecordBytesObserved(record, meter); ok || raw != nil || !meter.failed || meter.actual != before {
			t.Fatal("refused original observer revived or changed actuals on retry", count, meter.actual)
		}
	}
}
