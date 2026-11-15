//go:build darwin && registry_native_limits

package registry011

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

// Cancellation happens through a real CancelFunc at a selected access boundary.
// This is a small DATA-reader unit, without execution authority or host bypass.
type nativeBoundaryCancelContext struct {
	context.Context
	cancel context.CancelFunc
	calls  atomic.Int64
	at     atomic.Int64
}

func (c *nativeBoundaryCancelContext) Err() error {
	call := c.calls.Add(1)
	if at := c.at.Load(); at > 0 && call >= at {
		c.cancel()
	}
	return c.Context.Err()
}

type nativeBoundaryBuffer struct {
	region   *compilermemory.Region
	span     compilermemory.Span
	document compilermemory.JSONDocument
	action   *nativeBoundaryCancelContext
	cleanup  context.Context
}

func nativeBoundaryStorage(t *testing.T, raw []byte, capacity int) nativeBoundaryBuffer {
	t.Helper()
	action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	cleanup, stop := context.WithTimeout(context.Background(), 35*time.Second)
	t.Cleanup(stop)
	control := &nativeBoundaryCancelContext{Context: action, cancel: cancel}
	region, err := compilermemory.Acquire(control, cleanup, max(capacity, 1))
	if region != nil {
		t.Cleanup(func() {
			if err := region.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	if err != nil {
		t.Fatalf("bounded native storage: %v", err)
	}
	span, err := region.Store(raw)
	if err != nil {
		t.Fatal(err)
	}
	return nativeBoundaryBuffer{region: region, span: span, action: control, cleanup: cleanup}
}

func nativeBoundaryDocument(t *testing.T, raw []byte) nativeBoundaryBuffer {
	t.Helper()
	limits := compilermemory.JSONValueDepthLimits{MaximumValueDepth: 64, MaximumNodes: len(raw)}
	capacity, err := compilermemory.JSONValueCapacity(len(raw), limits)
	if err != nil {
		t.Fatal(err)
	}
	buffer := nativeBoundaryStorage(t, raw, capacity)
	buffer.document, err = buffer.region.ParseJSONAtValueDepth(buffer.span, limits)
	if err != nil {
		t.Fatal(err)
	}
	return buffer
}

func nativeBoundaryPolicyReader(t *testing.T, raw string) (*nativeProviderPolicyReader, *nativeBoundaryCancelContext) {
	t.Helper()
	buffer := nativeBoundaryDocument(t, []byte(raw))
	numbers, err := compilermemory.Acquire(buffer.action, buffer.cleanup, max(len(raw), 1))
	if numbers != nil {
		t.Cleanup(func() {
			if err := numbers.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	if err != nil {
		t.Fatal(err)
	}
	reader, err := newNativeProviderPolicyReader(buffer.region, numbers, buffer.document, buffer.span, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	return reader, buffer.action
}

func TestProviderHistoryNativeSelectionRejectsInvalidTypedBoundaries(t *testing.T) {
	for _, item := range []struct {
		name, raw string
		schema    reflect.Type
		depth     int
		cause     error
	}{
		{"missing-schema", "0", nil, 0, ErrConfiguration},
		{"negative-depth", "0", reflect.TypeFor[int](), -1, ErrConfiguration},
		{"excessive-depth", "0", reflect.TypeFor[int](), 65, ErrConfiguration},
		{"unsupported-pointer", "0", reflect.TypeFor[*int](), 0, ErrConfiguration},
		{"wrong-object", "[]", reflect.TypeFor[struct{ A int }](), 0, ErrConfiguration},
		{"missing-member", "{}", reflect.TypeFor[struct{ A int }](), 0, ErrConfiguration},
		{"extra-member", "{\"A\":1,\"B\":2}", reflect.TypeFor[struct{ A int }](), 0, ErrConfiguration},
		{"wrong-array", "{}", reflect.TypeFor[[1]int](), 0, ErrConfiguration},
		{"wrong-array-length", "[]", reflect.TypeFor[[1]int](), 0, ErrConfiguration},
		{"wrong-array-element", "[true]", reflect.TypeFor[[1]int](), 0, compilermemory.ErrJSONType},
	} {
		t.Run(item.name, func(t *testing.T) {
			buffer := nativeBoundaryDocument(t, []byte(item.raw))
			if err := validateNativeHistorySelectionValue(buffer.document.Root(), item.schema, item.depth); !errors.Is(err, item.cause) {
				t.Fatalf("invalid typed DATA accepted: %v", err)
			}
		})
	}
	t.Run("ignored-field", func(t *testing.T) {
		schema := reflect.StructOf([]reflect.StructField{{
			Name: "Ignored", Type: reflect.TypeFor[int](), Tag: reflect.StructTag("json:\"-\""),
		}})
		buffer := nativeBoundaryDocument(t, []byte("{}"))
		if err := validateNativeHistorySelectionValue(buffer.document.Root(), schema, 0); err != nil {
			t.Fatal("ignored metadata changed the closed wire shape", err)
		}
	})
	raw, _, _ := nativeHistoryTestRaw(t)
	for _, name := range []string{"empty", "input-limit", "missing-maximum", "missing-policy", "empty-policy", "policy-too-long", "invalid-policy-json", "policy-acquisition", "number-acquisition"} {
		t.Run(name, func(t *testing.T) {
			session, state, _ := nativeHistoryTestSession(t, len(raw)+4096)
			var object map[string]json.RawMessage
			if err := json.Unmarshal(raw, &object); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "missing-maximum":
				delete(object, "maximum_policy_bytes")
			case "missing-policy":
				delete(object, "policy")
			case "empty-policy":
				object["policy"] = json.RawMessage("\"\"")
			case "policy-too-long":
				object["maximum_policy_bytes"] = json.RawMessage("1")
			case "invalid-policy-json":
				object["policy"] = json.RawMessage("\"{\"")
			}
			input, err := json.Marshal(object)
			if err != nil {
				t.Fatal(err)
			}
			if name == "empty" {
				input = nil
			} else if name == "input-limit" {
				session.maximumInput = len(input) - 1
			}
			marker := errors.New("simulated acquisition refusal")
			acquire, calls, failAt := state.acquire, 0, 0
			if name == "policy-acquisition" {
				failAt = 2
			} else if name == "number-acquisition" {
				failAt = 3
			}
			state.acquire = func(action, cleanup context.Context, capacity int) (*compilermemory.Region, error) {
				calls++
				if calls == failAt {
					return nil, marker
				}
				return acquire(action, cleanup, capacity)
			}
			selection, err := state.decodeOwnedSelection(session, input)
			if selection == nil || selection.validated || err == nil {
				t.Fatalf("invalid selection remained usable: %v", err)
			}
			if failAt != 0 {
				if calls != failAt || !errors.Is(err, marker) {
					t.Fatal("acquisition refusal was lost or retried", err)
				}
			} else if !errors.Is(err, ErrConfiguration) || errors.Is(err, compilermemory.ErrUnavailable) ||
				errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("invalid input was masked by another refusal", err)
			}
		})
	}
}

func TestProviderHistoryNativeSelectionConfigKeepsRefusalCauses(t *testing.T) {
	for _, raw := range []string{
		"{}",
		"{\"format\":" + strconv.Quote(historyPreparationInputFormat) + "}",
		"{\"format\":" + strconv.Quote(historyPreparationInputFormat) + ",\"capture\":[]}",
	} {
		t.Run("incomplete-header/"+strconv.Itoa(len(raw)), func(t *testing.T) {
			buffer := nativeBoundaryDocument(t, []byte(raw))
			if err := nativeHistorySelectionHeader(buffer.document.Root()); err != ErrConfiguration {
				t.Fatal("incomplete header was accepted", err)
			}
		})
	}
	t.Run("absent-selection", func(t *testing.T) {
		var absent *nativeHistoryOwnedSelection
		if _, err := nativeHistorySelectionPart[uint64](absent, "clock"); err != ErrConfiguration {
			t.Fatal("absent selection supplied a section", err)
		}
		if _, err := absent.bindingText("namespace", 66); err != ErrConfiguration {
			t.Fatal("absent selection supplied binding text", err)
		}
		if _, _, err := absent.transientConfig(); err != ErrConfiguration {
			t.Fatal("absent selection supplied config", err)
		}
	})
	t.Run("span-boundaries", func(t *testing.T) {
		if _, err := nativeHistorySpanBytes(nil, compilermemory.Span{}, 1); err != compilermemory.ErrConfiguration {
			t.Fatal("absent span supplied bytes", err)
		}
		buffer := nativeBoundaryStorage(t, []byte("data"), 4)
		if _, err := nativeHistorySpanBytes(buffer.region, buffer.span, 3); err != ErrConfiguration {
			t.Fatal("oversized span supplied bytes", err)
		}
		if err := buffer.region.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := nativeHistorySpanBytes(buffer.region, buffer.span, 4); !errors.Is(err, compilermemory.ErrClosed) {
			t.Fatal("span lost its original lifetime refusal", err)
		}
	})
	for _, expected := range []string{
		strings.Repeat("a", 4095),
		strings.Repeat("b", 4096),
		strings.Repeat("c", 4097),
		"x" + strings.Repeat("\u00e9", 4097),
	} {
		t.Run("text-span-round-trip/"+strconv.Itoa(len(expected)), func(t *testing.T) {
			input := []byte(expected)
			buffer := nativeBoundaryStorage(t, input, len(input))
			clear(input)
			text, err := nativeHistorySpanText(buffer.region, buffer.span, len(expected))
			if err != nil || text != expected {
				t.Fatal("native text changed its original bytes", err)
			}
			if err := buffer.region.Close(); err != nil {
				t.Fatal(err)
			}
			if text != expected {
				t.Fatal("Go text retained the closed native storage")
			}
		})
	}
	t.Run("text-span-boundaries", func(t *testing.T) {
		if text, err := nativeHistorySpanText(nil, compilermemory.Span{}, 1); !errors.Is(err, compilermemory.ErrConfiguration) || text != "" {
			t.Fatal("absent owner supplied text", err)
		}
		buffer := nativeBoundaryStorage(t, []byte("data"), 4)
		for _, maximum := range []int{-1, 0, 3} {
			if text, err := nativeHistorySpanText(buffer.region, buffer.span, maximum); !errors.Is(err, ErrConfiguration) || text != "" {
				t.Fatal("out-of-bound span supplied text", err)
			}
		}
		foreign := nativeBoundaryStorage(t, []byte("else"), 4)
		if text, err := nativeHistorySpanText(foreign.region, buffer.span, 4); err == nil || text != "" {
			t.Fatal("foreign owner supplied text", err)
		}
		if err := buffer.region.Close(); err != nil {
			t.Fatal(err)
		}
		if text, err := nativeHistorySpanText(buffer.region, buffer.span, 4); !errors.Is(err, compilermemory.ErrClosed) || text != "" {
			t.Fatal("closed owner supplied text or lost its cause", err)
		}
	})
	textInput := strings.Repeat("bounded original policy ", 400)
	for boundary := int64(1); boundary <= 32; boundary++ {
		t.Run("text-span-cancel/"+strconv.FormatInt(boundary, 10), func(t *testing.T) {
			buffer := nativeBoundaryStorage(t, []byte(textInput), len(textInput))
			buffer.action.calls.Store(0)
			buffer.action.at.Store(boundary)
			text, err := nativeHistorySpanText(buffer.region, buffer.span, len(textInput))
			if buffer.action.Context.Err() != nil {
				if !errors.Is(err, context.Canceled) || text != "" {
					t.Fatal("canceled read supplied partial native text", err)
				}
			} else if err != nil || text != textInput {
				t.Fatal("live native text changed before cancellation", err)
			}
		})
	}
	raw, _, _ := nativeHistoryTestRaw(t)
	for _, field := range []string{"namespace", "source_id", "generation", "store", "capture", "collection", "bounds", "clock"} {
		t.Run("invalid-section/"+field, func(t *testing.T) {
			session, state, _ := nativeHistoryTestSession(t, len(raw))
			selection, err := state.decodeOwnedSelection(session, raw)
			if err != nil {
				t.Fatal(err)
			}
			var object map[string]json.RawMessage
			if err := json.Unmarshal(raw, &object); err != nil {
				t.Fatal(err)
			}
			cause := ErrConfiguration
			switch field {
			case "namespace", "source_id", "generation":
				object[field] = json.RawMessage("\"\"")
				cause = ErrProviderPolicyCandidate
			default:
				object[field] = json.RawMessage("false")
			}
			changed, err := json.Marshal(object)
			if err != nil {
				t.Fatal(err)
			}
			// A private DATA-view fault, not a compiled plan or permission.
			buffer := nativeBoundaryDocument(t, changed)
			selection.document = buffer.document
			config, clock, err := selection.transientConfig()
			if !errors.Is(err, cause) || config != (providerCustodyConfig{}) || clock != (historyPreparationClock{}) {
				t.Fatal("invalid section produced a partial usable config", err)
			}
		})
	}
	for _, name := range []string{"missing-policy", "wrong-owner", "closed-policy", "missing-subtrees", "wrong-binding-type", "wrong-section-type"} {
		t.Run(name, func(t *testing.T) {
			session, state, _ := nativeHistoryTestSession(t, len(raw))
			selection, err := state.decodeOwnedSelection(session, raw)
			if err != nil {
				t.Fatal(err)
			}
			cause := ErrConfiguration
			switch name {
			case "missing-policy":
				selection.policy = nil
			case "wrong-owner":
				selection.policy.source = state.root.region
			case "closed-policy":
				if err := state.policy.region.Close(); err != nil {
					t.Fatal(err)
				}
				cause = compilermemory.ErrClosed
			case "missing-subtrees":
				buffer := nativeBoundaryDocument(t, []byte("{}"))
				selection.policy.root = buffer.document.Root()
			case "wrong-binding-type":
				cause = compilermemory.ErrJSONType
				_, err = selection.bindingText("clock", 66)
			case "wrong-section-type":
				_, err = nativeHistorySelectionPart[uint64](selection, "namespace")
			}
			if name != "wrong-binding-type" && name != "wrong-section-type" {
				_, err = selection.transientPolicy()
			}
			if !errors.Is(err, cause) {
				t.Fatal("DATA-view refusal lost its original cause", err)
			}
		})
	}
}

func TestProviderHistoryNativeSelectionRetainedDataBoundaries(t *testing.T) {
	raw, _, expectedManifest := nativeHistoryTestRaw(t)
	session, state, _ := nativeHistoryTestSession(t, len(raw))
	if err := session.CompileOwned(raw); err != nil {
		t.Fatal("original retained-plan precondition", err)
	}
	selection, plan := state.selection, session.plan
	length, err := selection.manifestOwner.SpanLength(selection.manifestSpan)
	if err != nil {
		t.Fatal(err)
	}
	record := make([]byte, length)
	if err := selection.manifestOwner.Read(selection.manifestSpan, record); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"empty", "short-header", "header-only", "magic", "digest-format", "length", "body-digest"} {
		t.Run("record/"+name, func(t *testing.T) {
			data := append([]byte(nil), record...)
			switch name {
			case "empty":
				data = nil
			case "short-header":
				data = data[:nativeHistoryManifestHeaderBytes-1]
			case "header-only":
				data = data[:nativeHistoryManifestHeaderBytes]
			case "magic":
				data[0] ^= 1
			case "digest-format":
				data[8] = 'X'
			case "length":
				binary.LittleEndian.PutUint64(data[8+nativeHistoryRegistryDigestBytes:], 0)
			case "body-digest":
				data[len(data)-1] ^= 1
			}
			buffer := nativeBoundaryStorage(t, data, len(data))
			// Only a malformed DATA record is supplied; no plan or authority
			// is reconstructed from this detached test buffer.
			probe := &nativeHistoryOwnedSelection{manifestOwner: buffer.region, manifestSpan: buffer.span}
			digest, counts, err := probe.retainedManifest()
			if err != ErrConfiguration || digest != "" || counts != (providerHistoryCollectionCounts{}) {
				t.Fatal("malformed retained DATA produced usable output", err)
			}
		})
	}
	for _, name := range []string{"root-kind", "missing-policy", "policy-kind", "result-size", "foreign-result", "group-count"} {
		t.Run("binding/"+name, func(t *testing.T) {
			document, policy, policyRoot := selection.document, selection.policy, selection.policy.root
			resultSpan, stateSpan := selection.resultSpan, state.resultSpan
			groups := plan.source.collection.groups
			t.Cleanup(func() {
				selection.document, selection.policy = document, policy
				policy.root = policyRoot
				selection.resultSpan, state.resultSpan = resultSpan, stateSpan
				plan.source.collection.groups = groups
			})
			cause := ErrConfiguration
			switch name {
			case "root-kind":
				buffer := nativeBoundaryDocument(t, []byte("true"))
				selection.document = buffer.document
			case "missing-policy":
				selection.policy = nil
			case "policy-kind":
				buffer := nativeBoundaryDocument(t, []byte("true"))
				selection.policy.root = buffer.document.Root()
			case "result-size":
				short, err := state.result.region.Slice(state.resultSpan, 0, 1)
				if err != nil {
					t.Fatal(err)
				}
				selection.resultSpan, state.resultSpan = short, short
			case "foreign-result":
				buffer := nativeBoundaryStorage(t, []byte("x"), 1)
				selection.resultSpan, state.resultSpan = buffer.span, buffer.span
				cause = compilermemory.ErrRange
			case "group-count":
				plan.source.collection.groups++
			}
			if digest, err := selection.checkedRetainedManifest(plan); !errors.Is(err, cause) || digest != "" {
				t.Fatal("changed retained binding was accepted", err)
			}
		})
	}
	t.Run("invalid-promotion", func(t *testing.T) {
		var absent *nativeHistoryOwnedSelection
		if err := absent.retainCompiledPlan(nil); err != ErrConfiguration {
			t.Fatal("absent selection promoted a plan", err)
		}
		if err := selection.retainCompiledPlan(nil); err != ErrConfiguration {
			t.Fatal("retained selection promoted another plan", err)
		}
		if err := selection.storeCompiledManifest(&providerHistoryCollectionManifest{}); err != ErrConfiguration {
			t.Fatal("unvalidated manifest was stored", err)
		}
	})
	for _, operation := range []string{"manifest", "digest", "bytes"} {
		for boundary := int64(1); boundary <= 24; boundary++ {
			t.Run(operation+"/cancel-"+strconv.FormatInt(boundary, 10), func(t *testing.T) {
				data := record
				if operation == "digest" {
					data = record[nativeHistoryManifestHeaderBytes:]
				}
				buffer := nativeBoundaryStorage(t, data, len(data))
				buffer.action.calls.Store(0)
				buffer.action.at.Store(boundary)
				var digest string
				var output []byte
				var err error
				switch operation {
				case "manifest":
					probe := &nativeHistoryOwnedSelection{manifestOwner: buffer.region, manifestSpan: buffer.span}
					digest, _, err = probe.retainedManifest()
				case "digest":
					digest, err = nativeHistorySpanDigest(buffer.region, buffer.span, "sage-provider-history-collection-manifest-v1")
				case "bytes":
					output, err = nativeHistorySpanBytes(buffer.region, buffer.span, len(data))
				}
				if buffer.action.Context.Err() != nil {
					if !errors.Is(err, context.Canceled) || digest != "" || output != nil {
						t.Fatal("canceled native DATA escaped as usable output", err)
					}
				} else if err != nil || (operation == "bytes" && string(output) != string(data)) ||
					(operation != "bytes" && digest != expectedManifest) {
					t.Fatal("positive native DATA changed before cancellation", err)
				}
			})
		}
	}
}

func TestProviderPolicyNativeBoundariesKeepOriginalRefusals(t *testing.T) {
	t.Run("constructor", func(t *testing.T) {
		reader, _ := nativeBoundaryPolicyReader(t, "{\"a\":\"1\"}")
		buffer := nativeBoundaryDocument(t, []byte("{\"a\":\"1\"}"))
		for _, item := range []struct {
			name     string
			source   *compilermemory.Region
			numbers  *compilermemory.Region
			document compilermemory.JSONDocument
			span     compilermemory.Span
			length   int
			cause    error
		}{
			{"nil-source", nil, reader.numbers, buffer.document, buffer.span, 9, ErrConfiguration},
			{"nil-numbers", buffer.region, nil, buffer.document, buffer.span, 9, ErrConfiguration},
			{"same-owner", buffer.region, buffer.region, buffer.document, buffer.span, 9, ErrConfiguration},
			{"zero-length", buffer.region, reader.numbers, buffer.document, buffer.span, 0, ErrConfiguration},
			{"oversized-length", buffer.region, reader.numbers, buffer.document, buffer.span, 65537, ErrConfiguration},
			{"missing-document", buffer.region, reader.numbers, compilermemory.JSONDocument{}, buffer.span, 9, compilermemory.ErrConfiguration},
			{"foreign-document", reader.source, reader.numbers, buffer.document, buffer.span, 9, compilermemory.ErrRange},
			{"partial-input", buffer.region, reader.numbers, buffer.document, buffer.span, 8, ErrProviderPolicyCandidate},
		} {
			t.Run(item.name, func(t *testing.T) {
				value, err := newNativeProviderPolicyReader(item.source, item.numbers, item.document, item.span, item.length)
				if value != nil || !errors.Is(err, item.cause) {
					t.Fatalf("invalid policy ownership accepted: %v", err)
				}
			})
		}
	})
	t.Run("closed-schema", func(t *testing.T) {
		reader, _ := nativeBoundaryPolicyReader(t, "{\"a\":\"1\",\"b\":\"2\"}")
		value := nativeProviderPolicyValue{reader: reader, value: reader.root, valid: true}
		if value.object("a", "missing") || reader.err != nil {
			t.Fatal("an absent field satisfied the closed schema")
		}
		if value.object("a", "a") || !errors.Is(reader.err, ErrConfiguration) {
			t.Fatal("duplicate schema names were accepted", reader.err)
		}
		parts, names := make([]string, 15), make([]string, 15)
		for index := range parts {
			names[index] = "field" + strconv.Itoa(index)
			parts[index] = strconv.Quote(names[index]) + ":0"
		}
		reader, _ = nativeBoundaryPolicyReader(t, "{"+strings.Join(parts, ",")+"}")
		value = nativeProviderPolicyValue{reader: reader, value: reader.root, valid: true}
		if value.object(names...) || !errors.Is(reader.err, ErrConfiguration) {
			t.Fatal("oversized schema was accepted", reader.err)
		}
	})
	t.Run("wrong-kinds-and-bounds", func(t *testing.T) {
		reader, _ := nativeBoundaryPolicyReader(t, "\"value\"")
		value := nativeProviderPolicyValue{reader: reader, value: reader.root, valid: true}
		if value.object() || value.member("a").equals("1") {
			t.Fatal("scalar supplied an object field")
		}
		if _, ok := value.arrayLength(); ok || value.element(0).equals("1") {
			t.Fatal("scalar supplied an array element")
		}
		if _, ok := value.text(make([]byte, 2)); ok || reader.err != nil {
			t.Fatal("short destination accepted decoded text")
		}
		reader, _ = nativeBoundaryPolicyReader(t, "[\"1\"]")
		value = nativeProviderPolicyValue{reader: reader, value: reader.root, valid: true}
		if value.equals("1") {
			t.Fatal("array satisfied a text comparison")
		}
		if _, ok := value.text(make([]byte, 8)); ok {
			t.Fatal("array supplied decoded text")
		}
		if value.element(-1).equals("1") || value.element(1).equals("1") {
			t.Fatal("out-of-range array element was accepted")
		}
		for _, raw := range []string{"\"\"", "\"123456789012345678901\""} {
			reader, _ = nativeBoundaryPolicyReader(t, raw)
			value = nativeProviderPolicyValue{reader: reader, value: reader.root, valid: true}
			if _, ok := value.decimal(false); ok || reader.err != nil {
				t.Fatal("invalid decimal width was accepted")
			}
		}
	})
	for _, item := range []struct {
		name, raw string
		read      func(*nativeProviderPolicyReader) (bool, error)
	}{
		{"object", "{\"a\":\"1\",\"b\":\"2\"}", func(r *nativeProviderPolicyReader) (bool, error) {
			ok := (nativeProviderPolicyValue{reader: r, value: r.root, valid: true}).object("a", "b")
			return ok, r.err
		}},
		{"member", "{\"a\":\"1\"}", func(r *nativeProviderPolicyReader) (bool, error) {
			ok := (nativeProviderPolicyValue{reader: r, value: r.root, valid: true}).member("a").equals("1")
			return ok, r.err
		}},
		{"array", "[\"1\",\"2\"]", func(r *nativeProviderPolicyReader) (bool, error) {
			n, ok := (nativeProviderPolicyValue{reader: r, value: r.root, valid: true}).arrayLength()
			return ok && n == 2, r.err
		}},
		{"element", "[\"1\"]", func(r *nativeProviderPolicyReader) (bool, error) {
			ok := (nativeProviderPolicyValue{reader: r, value: r.root, valid: true}).element(0).equals("1")
			return ok, r.err
		}},
		{"equals", "\"value\"", func(r *nativeProviderPolicyReader) (bool, error) {
			ok := (nativeProviderPolicyValue{reader: r, value: r.root, valid: true}).equals("value")
			return ok, r.err
		}},
		{"text", "\"value\"", func(r *nativeProviderPolicyReader) (bool, error) {
			var data [5]byte
			n, ok := (nativeProviderPolicyValue{reader: r, value: r.root, valid: true}).text(data[:])
			return ok && n == 5 && string(data[:]) == "value", r.err
		}},
		{"decimal", "\"12\"", func(r *nativeProviderPolicyReader) (bool, error) {
			n, ok := (nativeProviderPolicyValue{reader: r, value: r.root, valid: true}).decimal(true)
			return ok && n == 12, r.err
		}},
		{"canonical", "{\"a\":\"value\"}", func(r *nativeProviderPolicyReader) (bool, error) {
			err := r.canonicalText()
			return err == nil, err
		}},
	} {
		for boundary := int64(1); boundary <= 32; boundary++ {
			t.Run(item.name+"/cancel-"+strconv.FormatInt(boundary, 10), func(t *testing.T) {
				reader, control := nativeBoundaryPolicyReader(t, item.raw)
				control.calls.Store(0)
				control.at.Store(boundary)
				ok, err := item.read(reader)
				if control.Context.Err() != nil {
					if ok || !errors.Is(err, context.Canceled) || !errors.Is(err, ErrProviderPolicyCandidate) {
						t.Fatalf("native read lost cancellation: ok=%v, error=%v", ok, err)
					}
				} else if !ok || err != nil {
					t.Fatalf("positive read failed before cancellation: ok=%v, error=%v", ok, err)
				}
			})
		}
	}
}
