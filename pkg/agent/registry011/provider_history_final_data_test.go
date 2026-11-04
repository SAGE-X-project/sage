package registry011

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Copy-only DATA UNIT subject. No original store, clock, owner, selection or
// native primitive is constructed by this value or its synthetic observer.
func historyFinalDataInput() providerHistoryFinalData {
	hash := admissionTestDigest("1")
	return providerHistoryFinalData{
		program: hash, costs: hash, profile: hash, manifest: hash, model: hash, commit: hash, collection: hash,
		selected: FinalizedHeader{hash, admissionTestDigest("2"), sepoliaDeploymentBlock, 1700000000},
		clockID:  "clock", epoch: admissionTestDigest("3"), beginWall: 1700000000000, beginMonotonic: 7, beginUncertainty: 1,
		groups: 4, sequence: 16, files: 21, bytes: 100, entries: 21, names: 200,
	}
}

func TestProviderHistoryFinalDataCopyUsesExactlyItsFundedValueEnvelope(t *testing.T) {
	in := historyFinalDataInput()
	upper, ok := providerHistoryFinalDataCopyUpper(5)
	// Eleven independently owned strings: ten66-byte hashes and five-byte
	// clock ID. The eleven sentinels and one struct copy stay distinct.
	want := candidateWorkVector{10: uint64(reflect.TypeFor[providerHistoryFinalData]().Size()) + 677}
	if !ok || upper != want {
		t.Fatal("detached DATA copy omitted a field, sentinel or fixed value struct", ok, upper)
	}
	m := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{work: upper, execution: upper})
	got, ok := providerHistoryFinalDataCopyObserved(in, m)
	if !ok || got != in || m.actual != want || m.failed || m.closed || m.clocks != 0 || m.authority {
		t.Fatal("copy changed values, borrowed a coordinate or acquired native authority", ok, m.actual)
	}
	if json.Unmarshal([]byte(`{}`), &providerHistoryFinalData{}) == nil {
		t.Fatal("serialized DATA constructed a private verification result")
	}
	// The returned schema can have scalar fields, fixed arrays and structs,
	// but no owner/store/model/catalog pointer or mutable collection/callback.
	var check func(reflect.Type)
	check = func(typ reflect.Type) {
		switch typ.Kind() {
		case reflect.Struct:
			for i := 0; i < typ.NumField(); i++ {
				check(typ.Field(i).Type)
			}
		case reflect.Array:
			check(typ.Elem())
		case reflect.String, reflect.Uint64, reflect.Int64:
		default:
			t.Fatal("result schema contains a mutable or executable alias", typ)
		}
	}
	check(reflect.TypeFor[providerHistoryFinalData]())
}

func TestProviderHistoryFinalDataCopyRejectsIncompleteOrUnfundedInputs(t *testing.T) {
	for _, name := range []string{"clock-id", "groups-zero", "groups-high", "header", "program", "epoch", "nil-observer", "closed-observer", "insufficient-work"} {
		t.Run(name, func(t *testing.T) {
			in := historyFinalDataInput()
			upper, ok := providerHistoryFinalDataCopyUpper(uint64(len(in.clockID)))
			if !ok {
				t.Fatal("normal copy precursor")
			}
			if name == "insufficient-work" {
				upper[10]-- // Separate numeric UNIT selection, before observer creation.
			}
			m := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{work: upper, execution: upper})
			switch name {
			case "clock-id":
				in.clockID = ""
			case "groups-zero":
				in.groups = 0
			case "groups-high":
				in.groups = (1<<20)/3 + 1
			case "header":
				in.selected = FinalizedHeader{}
			case "program":
				in.program = "unselected"
			case "epoch":
				in.epoch = ""
			case "nil-observer":
				m = nil
			case "closed-observer":
				m.closed = true // Synthetic DATA observer; no live role is patched.
			}
			if got, ok := providerHistoryFinalDataCopyObserved(in, m); ok || got != (providerHistoryFinalData{}) {
				t.Fatal("incomplete or underfunded input published partial copied fields", ok)
			}
			if m != nil && (m.actual != (candidateWorkVector{}) || m.clocks != 0 || m.authority) {
				t.Fatal("rejected copy consumed partial work or gained native permission")
			}
		})
	}
}

func TestProviderHistoryFinalDataCopyUpperRefusesEmptyAndOverflow(t *testing.T) {
	for _, n := range []uint64{0, ^uint64(0)} {
		if got, ok := providerHistoryFinalDataCopyUpper(n); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid clock-ID byte maximum published partial copy work", n, ok)
		}
	}
}

func TestProviderHistoryFinalDataPreparationFundsTheIndependentSealedValue(t *testing.T) {
	size := uint64(reflect.TypeFor[providerHistoryFinalData]().Size())
	got, ok := providerHistoryFinalDataPreparationUpper(5)
	if !ok || got != (candidateWorkVector{10: 2*size + 678}) {
		t.Fatal("complete preparation omitted its independent fixed DATA copy", ok, got)
	}
	for _, n := range []uint64{0, ^uint64(0), ^uint64(0) - (size + 672)} {
		if n != 0 && n != ^uint64(0) {
			copy, ok := providerHistoryFinalDataCopyUpper(n)
			if !ok || copy[10] != ^uint64(0) {
				t.Fatal("copy-only upper must fit before the separate snapshot overflows")
			}
		}
		if value, ok := providerHistoryFinalDataPreparationUpper(n); ok || value != (candidateWorkVector{}) {
			t.Fatal("incomplete preparation budget published a partial row", n, ok)
		}
	}
}

// Binding-only UNIT values: no store, clock, meter, selection or native
// resource. A matching pair proves only a comparison, never verification.
func historyFinalDataBindingUnit() (*providerHistoryFinalOwner, *providerHistoryFinalResult) {
	f := &providerHistoryFinalOwner{}
	f.owner = f
	out := &providerHistoryFinalResult{final: f, data: historyFinalDataInput()}
	out.owner = out
	f.result, f.preparedData = out, out.data
	return f, out
}

func TestProviderHistoryFinalDataBindingRejectsOtherResults(t *testing.T) {
	f, out := historyFinalDataBindingUnit()
	if !f.selectedDataMatchesLocked(out) {
		t.Fatal("complete binding-only UNIT precursor")
	}
	copy := *out
	copy.owner = &copy
	if f.selectedDataMatchesLocked(&copy) || !f.selectedDataMatchesLocked(out) {
		t.Fatal("self-owned copy replaced the sole issued result or consumed its binding")
	}
	other := *f
	other.owner = &other
	if other.selectedDataMatchesLocked(out) || f.selectedDataMatchesLocked(nil) || (*providerHistoryFinalOwner)(nil).selectedDataMatchesLocked(out) {
		t.Fatal("another final owner or absent result matched the original binding")
	}
}

func TestProviderHistoryFinalDataBindingRejectsChangedValues(t *testing.T) {
	for _, name := range []string{"sequence", "files", "bytes", "entries", "names", "retention", "collection", "work", "end-clock", "selected-header"} {
		t.Run(name, func(t *testing.T) {
			f, out := historyFinalDataBindingUnit()
			if !f.selectedDataMatchesLocked(out) {
				t.Fatal("fresh complete binding-only UNIT precursor")
			}
			switch name {
			case "sequence":
				out.data.sequence++
			case "files":
				out.data.files++
			case "bytes":
				out.data.bytes++
			case "entries":
				out.data.entries++
			case "names":
				out.data.names++
			case "retention":
				out.data.retention = providerHistoryStorageCharge{physical: 1}
			case "collection":
				out.data.collection = admissionTestDigest("4")
			case "work":
				out.data.work[10]++
			case "end-clock":
				out.data.endMonotonic++
			case "selected-header":
				out.data.selected.Number++
			}
			if f.selectedDataMatchesLocked(out) {
				t.Fatal("changed DATA matched the original independent sealed value")
			}
		})
	}
}

func TestProviderHistoryFinalDataReleaseRejectsUnselectedResults(t *testing.T) {
	for _, out := range []*providerHistoryFinalResult{nil, {}, {final: &providerHistoryFinalOwner{}}} {
		if got, err := out.closeAndTakeDataCandidate(nil); err == nil || got != (providerHistoryFinalData{}) {
			t.Fatal("unselected result produced detached verified DATA", err)
		}
	}
}
