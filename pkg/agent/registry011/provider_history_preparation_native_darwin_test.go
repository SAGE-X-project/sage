//go:build darwin && registry_native_limits

package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

func nativeHistoryTestSession(t *testing.T, maximumInput int) (*HistoryPreparationSession, *nativeHistoryPreparation, context.CancelFunc) {
	t.Helper()
	origin := time.Now()
	action, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
	cleanup, stopCleanup := context.WithDeadline(context.Background(), origin.Add(2*time.Minute))
	session, err := NewNativeHistoryPreparationSession(action, cleanup, origin, max(maximumInput, 1), 4096)
	if err != nil {
		cancel()
		stopCleanup()
		t.Fatal(err)
	}
	state := session.native.(*nativeHistoryPreparation)
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
		checkNativeHistoryReleased(t, session, state)
		cancel()
		stopCleanup()
	})
	return session, state, cancel
}

func checkNativeHistoryReleased(t *testing.T, session *HistoryPreparationSession, state *nativeHistoryPreparation) {
	t.Helper()
	if !session.closed || session.compiler != nil || session.plan != nil || session.inputSHA256 != "" ||
		state.resultPlan != nil || state.compiler != nil || state.owner != nil || state.cleanup != nil || state.acquire != nil ||
		state.resultSpan != (compilermemory.Span{}) {
		t.Fatal("closed native session retained its private plan, compiler or context")
	}
}

func nativeHistoryTestRaw(t *testing.T) ([]byte, ProviderCandidateIdentity, string) {
	t.Helper()
	input, identity, manifest := historyPreparationInputFromOriginalTestAPI(t)
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	return raw, identity, manifest
}

func TestProviderHistoryPreparationNativeUsesOriginalPlanAndRetainedResult(t *testing.T) {
	raw, expectedIdentity, expectedManifest := nativeHistoryTestRaw(t)
	original := bytes.Clone(raw)
	origin := time.Now()
	action, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
	defer cancel()
	cleanup, stopCleanup := context.WithDeadline(context.Background(), origin.Add(2*time.Minute))
	defer stopCleanup()
	legacy, err := NewHistoryPreparationSession(action, origin, len(raw), 4096)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = legacy.Close() }()
	native, err := NewNativeHistoryPreparationSession(action, cleanup, origin, len(raw), 4096)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = native.Close() }()
	compiler := native.compiler
	if err := legacy.CompileOwned(raw); err != nil {
		t.Fatal(err)
	}
	if err := native.CompileOwned(raw); err != nil {
		t.Fatal(err)
	}
	state := native.native.(*nativeHistoryPreparation)
	plan := native.plan
	if native.compiler != compiler || state.compiler != compiler || state.resultPlan != plan || plan == nil || plan.clock == nil {
		t.Fatal("native path replaced the original compiler, clock or plan")
	}
	if state.root.closed || state.policy.closed || state.manifest.closed || state.result.closed ||
		state.selection == nil || !state.selection.numbers.closed {
		t.Fatal("temporary and retained owner lifetimes were not separated")
	}
	if plan.retained != state.selection || state.selection.plan != plan || plan.config != (providerCustodyConfig{}) {
		t.Fatal("native plan did not retain its exact native selection exclusively")
	}
	if _, err := state.root.region.Store(nil); err != compilermemory.ErrConfiguration {
		t.Fatalf("retained input was not sealed: %v", err)
	}
	if err := state.selection.checkRetainedPlan(plan); err != nil {
		t.Fatalf("same original plan could not consume its retained input: %v", err)
	}
	if err := native.CompileOwned(raw); !errors.Is(err, ErrConfiguration) || native.plan != plan || state.result.closed {
		t.Fatal("losing duplicate destroyed the original retained selection")
	}
	for index := range raw {
		raw[index] = 0xff
	}
	legacyResult, err := legacy.FinishDiagnostic()
	if err != nil {
		t.Fatal(err)
	}
	nativeResult, err := native.FinishDiagnostic()
	if err != nil || !bytes.Equal(legacyResult, nativeResult) {
		t.Fatalf("native diagnostic differs from original fields: %s / %s / %v", legacyResult, nativeResult, err)
	}
	summary, err := DecodeHistoryPreparationSummary(action, nativeResult, original, 4096)
	digest := sha256.Sum256(original)
	if err != nil || summary.Identity != expectedIdentity || summary.ManifestSHA256 != expectedManifest ||
		summary.InputSHA256 != hex.EncodeToString(digest[:]) || summary.SourceReady {
		t.Fatalf("native diagnostic lost original assertions: %+v, %v", summary, err)
	}
	if !state.result.closed || !state.manifest.closed || !state.root.closed || !state.policy.closed ||
		native.plan != nil || native.compiler != nil || plan.retained != nil {
		t.Fatal("last native consumer retained its original resources")
	}
	checkNativeHistoryReleased(t, native, state)
	if err := native.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestProviderHistoryPreparationNativeResultRefusalNeverReadsGoFallback(t *testing.T) {
	for _, mode := range []string{"closed", "canceled", "short", "magic", "hash", "input-end", "identity-prefix", "identity-end", "manifest-end", "wrong-plan", "wrong-owner"} {
		t.Run(mode, func(t *testing.T) {
			raw, _, _ := nativeHistoryTestRaw(t)
			session, state, cancel := nativeHistoryTestSession(t, len(raw))
			if err := session.CompileOwned(raw); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "closed":
				if err := state.result.region.Close(); err != nil {
					t.Fatal(err)
				}
			case "canceled":
				cancel()
			case "wrong-plan":
				state.resultPlan = &providerHistorySourcePlan{}
			case "wrong-owner":
				state.owner = new(HistoryPreparationSession)
			default:
				var record [nativeHistoryResultBytes]byte
				if err := state.result.region.Read(state.resultSpan, record[:]); err != nil {
					t.Fatal(err)
				}
				if err := state.result.close(); err != nil {
					t.Fatal(err)
				}
				owner, err := compilermemory.Acquire(session.compiler.ctx, state.cleanup, len(record))
				if err != nil {
					if owner != nil {
						_ = owner.Close()
					}
					t.Fatal(err)
				}
				state.result = nativeHistoryOwner{region: owner}
				data := record[:]
				switch mode {
				case "short":
					data = data[:len(data)-1]
				case "magic":
					data[0] ^= 0xff
				case "hash":
					data[8] = 'G'
				case "input-end":
					data[8+nativeHistoryInputDigestBytes-1] = 'G'
				case "identity-prefix":
					data[8+nativeHistoryInputDigestBytes] = 'x'
				case "identity-end":
					data[8+nativeHistoryInputDigestBytes+nativeHistoryRegistryDigestBytes-1] = 'G'
				case "manifest-end":
					data[8+nativeHistoryInputDigestBytes+6*nativeHistoryRegistryDigestBytes-1] = 'G'
				}
				state.resultSpan, err = owner.Store(data)
				if err != nil {
					t.Fatal(err)
				}
				if err := owner.Seal(); err != nil {
					t.Fatal(err)
				}
			}
			if output, err := session.FinishDiagnostic(); output != nil || err == nil {
				t.Fatalf("native result refusal used a Go fallback: %s, %v", output, err)
			}
			checkNativeHistoryReleased(t, session, state)
		})
	}
}

func TestProviderHistoryPreparationNativeResultPreservesBothDigestContracts(t *testing.T) {
	raw, identity, expectedManifest := nativeHistoryTestRaw(t)
	session, state, _ := nativeHistoryTestSession(t, len(raw))
	if err := session.CompileOwned(raw); err != nil {
		t.Fatal(err)
	}
	var record [nativeHistoryResultBytes]byte
	if err := state.result.region.Read(state.resultSpan, record[:]); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	inputHash := hex.EncodeToString(digest[:])
	if len(record) != 588 || string(record[8:8+nativeHistoryInputDigestBytes]) != inputHash {
		t.Fatal("plain input SHA changed length, prefix or trailing digits")
	}
	plan := session.plan
	expected := [6]string{identity.Policy, identity.Instance, identity.Provider, identity.Namespace, identity.SourceID, expectedManifest}
	offset := 8 + nativeHistoryInputDigestBytes
	for _, value := range expected {
		if len(value) != 66 || string(record[offset:offset+nativeHistoryRegistryDigestBytes]) != value {
			t.Fatal("original Registry digest prefix or final digits were lost")
		}
		offset += nativeHistoryRegistryDigestBytes
	}
	// A private acquisition probe distinguishes digest validation from the
	// already occupied real result owner. Reaching it on invalid DATA is a
	// failure; the callback cannot create a physical Region or a result.
	acquisitions := 0
	stopped := errors.New("simulated acquisition boundary")
	config, _, err := state.selection.transientConfig()
	if err != nil || config.identity != identity || config.bounds.history.collection.digest != expectedManifest {
		t.Fatalf("writer DATA precondition differs from the original fixture: %v", err)
	}
	// The encoder's refusal units use a detached DATA input. They never mutate
	// the live native plan or reattach a complete Go config to its selection.
	writerPlan := &providerHistorySourcePlan{
		config: config, source: plan.source, clock: plan.clock, compilerDeadline: plan.compilerDeadline,
	}
	probe := &nativeHistoryPreparation{
		owner: session, compiler: session.compiler, cleanup: state.cleanup,
		cleanupDeadline: state.cleanupDeadline,
		acquire: func(context.Context, context.Context, int) (*compilermemory.Region, error) {
			acquisitions++
			return nil, stopped
		},
	}
	t.Cleanup(func() {
		if err := probe.closeAll(); err != nil {
			t.Error(err)
		}
	})
	for _, invalid := range []string{"", inputHash[:63], inputHash + "0", "0x" + inputHash, "G" + inputHash[1:], inputHash[:63] + "G"} {
		if err := probe.storeResult(session, writerPlan, invalid); !errors.Is(err, ErrConfiguration) || acquisitions != 0 {
			t.Fatal("writer accepted an invalid plain input digest", err)
		}
	}
	for _, invalid := range []string{identity.Policy[:64], identity.Policy[2:], "0X" + identity.Policy[2:], identity.Policy[:65] + "G"} {
		writerPlan.config.identity.Policy = invalid
		err := probe.storeResult(session, writerPlan, inputHash)
		writerPlan.config.identity.Policy = identity.Policy
		if !errors.Is(err, ErrConfiguration) || acquisitions != 0 {
			t.Fatal("writer accepted an invalid original Registry digest", err)
		}
	}
	if err := probe.storeResult(session, writerPlan, inputHash); !errors.Is(err, stopped) || acquisitions != 1 {
		t.Fatal("valid original digests did not reach the acquisition boundary", err)
	}
	output, err := session.FinishDiagnostic()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeHistoryPreparationSummary(context.Background(), output, raw, 4096); err != nil {
		t.Fatal("valid result changed after invalid writer attempts", err)
	}
	checkNativeHistoryReleased(t, session, state)
}

func TestProviderHistoryPreparationNativeCloseDropsPrivateReferences(t *testing.T) {
	for _, cleanupFailure := range []bool{false, true} {
		t.Run(strconv.FormatBool(cleanupFailure), func(t *testing.T) {
			raw, _, _ := nativeHistoryTestRaw(t)
			origin := time.Now()
			action, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
			cleanup, stopCleanup := context.WithDeadline(context.Background(), origin.Add(2*time.Minute))
			session, err := NewNativeHistoryPreparationSession(action, cleanup, origin, len(raw), 4096)
			if err != nil {
				cancel()
				stopCleanup()
				t.Fatal(err)
			}
			state := session.native.(*nativeHistoryPreparation)
			var simulated error
			t.Cleanup(func() {
				defer cancel()
				defer stopCleanup()
				if err := session.Close(); err != nil && !errors.Is(err, simulated) {
					t.Error(err)
				}
				checkNativeHistoryReleased(t, session, state)
			})
			if err := session.CompileOwned(raw); err != nil {
				t.Fatal(err)
			}
			if cleanupFailure {
				// The temporary Region is already closed. Inject only a cached
				// unit error; this is not evidence of a physical cleanup failure.
				simulated = errors.New("simulated cached cleanup refusal")
				state.root.closeErr = simulated
			}
			first := session.Close()
			checkNativeHistoryReleased(t, session, state)
			if second := session.Close(); second != first || !state.result.closed {
				t.Fatal("repeat Close lost its original cleanup result")
			}
			if cleanupFailure {
				if !errors.Is(first, simulated) || !errors.Is(first, ErrHistoryPreparationCleanup) {
					t.Fatal("reference release erased the cached cleanup refusal", first)
				}
			} else if first != nil {
				t.Fatal(first)
			}
		})
	}
}

func TestProviderHistoryPreparationNativeDecoderPreservesEveryRequiredField(t *testing.T) {
	raw, _, _ := nativeHistoryTestRaw(t)
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var tree any
	if err := decoder.Decode(&tree); err != nil {
		t.Fatal(err)
	}
	type member struct {
		path   string
		name   string
		parent map[string]any
	}
	var members []member
	var collect func(any, string)
	collect = func(value any, path string) {
		switch node := value.(type) {
		case map[string]any:
			names := make([]string, 0, len(node))
			for name := range node {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				members = append(members, member{path + "." + name, name, node})
				collect(node[name], path+"."+name)
			}
		case []any:
			for index, child := range node {
				collect(child, path+"["+strconv.Itoa(index)+"]")
			}
		}
	}
	collect(tree, "input")
	for _, field := range members {
		t.Run(field.path, func(t *testing.T) {
			saved := field.parent[field.name]
			delete(field.parent, field.name)
			invalid, err := json.Marshal(tree)
			field.parent[field.name] = saved
			if err != nil {
				t.Fatal(err)
			}
			session, state, _ := nativeHistoryTestSession(t, len(invalid))
			if output, digest, err := state.decodeInput(session, invalid); err == nil || !reflect.DeepEqual(output, historyPreparationInput{}) || digest != "" {
				t.Fatalf("missing field accepted or leaked partial data: %s, %v", field.path, err)
			}
		})
	}
}

func TestProviderHistoryPreparationNativeDecoderRefusesChangedWireShapes(t *testing.T) {
	raw, _, _ := nativeHistoryTestRaw(t)
	type mutation struct {
		name string
		edit func(map[string]any)
	}
	mutations := []mutation{
		{"unknown-root", func(tree map[string]any) { tree["extra"] = true }},
		{"case-varied", func(tree map[string]any) { tree["Format"] = tree["format"]; delete(tree, "format") }},
		{"policy-object", func(tree map[string]any) { tree["policy"] = map[string]any{} }},
		{"null-clock", func(tree map[string]any) { tree["clock"] = nil }},
		{"wrong-clock", func(tree map[string]any) { tree["clock"] = []any{} }},
		{"unsigned-negative", func(tree map[string]any) { tree["clock"].(map[string]any)["maximum"] = json.Number("-1") }},
		{"unsigned-overflow", func(tree map[string]any) {
			tree["clock"].(map[string]any)["maximum"] = json.Number("18446744073709551616")
		}},
		{"integer-exponent", func(tree map[string]any) { tree["clock"].(map[string]any)["maximum"] = json.Number("1e0") }},
	}
	for _, changed := range mutations {
		t.Run(changed.name, func(t *testing.T) {
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			var tree map[string]any
			if err := decoder.Decode(&tree); err != nil {
				t.Fatal(err)
			}
			changed.edit(tree)
			invalid, err := json.Marshal(tree)
			if err != nil {
				t.Fatal(err)
			}
			session, state, _ := nativeHistoryTestSession(t, len(invalid))
			if _, _, err := state.decodeInput(session, invalid); err == nil {
				t.Fatal("changed exact-schema wire shape accepted")
			}
		})
	}
	for _, suffix := range [][]byte{[]byte(" null"), {0xff}} {
		invalid := append(bytes.Clone(raw), suffix...)
		session, state, _ := nativeHistoryTestSession(t, len(invalid))
		if _, _, err := state.decodeInput(session, invalid); err == nil {
			t.Fatal("trailing or invalid UTF-8 input accepted")
		}
	}
	for _, name := range []string{"format", "\\u0066ormat"} {
		duplicate := append([]byte("{\""+name+"\":\"duplicate\","), raw[1:]...)
		session, state, _ := nativeHistoryTestSession(t, len(duplicate))
		if _, _, err := state.decodeInput(session, duplicate); err == nil {
			t.Fatal("ordinary or decoded duplicate member accepted")
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var tree map[string]any
	if err := decoder.Decode(&tree); err != nil {
		t.Fatal(err)
	}
	var checkArrays func(map[string]any, string)
	checkArrays = func(node map[string]any, path string) {
		names := make([]string, 0, len(node))
		for name := range node {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			switch value := node[name].(type) {
			case map[string]any:
				checkArrays(value, path+"."+name)
			case []any:
				if len(value) == 0 {
					t.Fatal("fixed-array fixture unexpectedly empty")
				}
				for _, long := range []bool{false, true} {
					t.Run(path+"."+name+"/"+strconv.FormatBool(long), func(t *testing.T) {
						changed := value[:len(value)-1]
						if long {
							changed = append(append([]any(nil), value...), value[0])
						}
						node[name] = changed
						invalid, err := json.Marshal(tree)
						node[name] = value
						if err != nil {
							t.Fatal(err)
						}
						session, state, _ := nativeHistoryTestSession(t, len(invalid))
						if _, _, err := state.decodeInput(session, invalid); err == nil {
							t.Fatal("fixed array length changed")
						}
					})
				}
			}
		}
	}
	checkArrays(tree, "input")
}

func TestProviderHistoryPreparationNativeDecoderMatchesOriginalPolicySyntax(t *testing.T) {
	raw, _, _ := nativeHistoryTestRaw(t)
	var baseline historyPreparationInput
	if err := json.Unmarshal(raw, &baseline); err != nil {
		t.Fatal(err)
	}
	for index, policy := range []string{
		baseline.Policy, "{}", "[]", "0", "true", "null",
		"{\"x\":0,\"x\":1}", "{\"x\":0,\"\\u0078\":1}", "0 0",
		strings.Repeat("[", 65) + strings.Repeat("]", 65),
		strings.Repeat("[", 66) + strings.Repeat("]", 66),
		strings.Repeat("[", 64) + "0" + strings.Repeat("]", 64),
		strings.Repeat("[", 65) + "0" + strings.Repeat("]", 65),
	} {
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			input := baseline
			input.Policy = policy
			input.MaximumPolicyBytes = 64 << 10
			encoded, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			session, state, _ := nativeHistoryTestSession(t, len(encoded))
			expected, legacyErr := decodeHistoryPreparationInput(session.compiler.ctx, encoded)
			actual, digest, nativeErr := state.decodeInput(session, encoded)
			if (nativeErr == nil) != (legacyErr == nil) {
				t.Fatalf("policy syntax boundary changed: %v / %v", nativeErr, legacyErr)
			}
			if nativeErr == nil && (!reflect.DeepEqual(actual, expected) || digest == "") {
				t.Fatal("native typed DTO differs from original input contract")
			}
		})
	}
	for _, invalid := range []struct {
		policy string
		limit  int
	}{
		{"", baseline.MaximumPolicyBytes}, {baseline.Policy, 0},
		{baseline.Policy, -1}, {baseline.Policy, 64<<10 + 1},
		{baseline.Policy, len(baseline.Policy) - 1},
		{strings.Repeat(" ", 64<<10+1), 64 << 10},
	} {
		input := baseline
		input.Policy, input.MaximumPolicyBytes = invalid.policy, invalid.limit
		encoded, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		session, state, _ := nativeHistoryTestSession(t, len(encoded))
		if _, _, err := state.decodeInput(session, encoded); err == nil || state.policy.region != nil {
			t.Fatalf("invalid policy acquired its syntax owner: %v", err)
		}
	}
}

func TestProviderHistoryPreparationNativeSimulatedAcquisitionFailuresRemainTerminal(t *testing.T) {
	raw, _, _ := nativeHistoryTestRaw(t)
	t.Run("owner-admission-refusals", func(t *testing.T) {
		session, state, _ := nativeHistoryTestSession(t, len(raw))
		otherSession, _, _ := nativeHistoryTestSession(t, len(raw))
		calls := 0
		state.acquire = func(context.Context, context.Context, int) (*compilermemory.Region, error) {
			calls++
			return nil, errors.New("unexpected native acquisition")
		}
		closedError := errors.New("retained close outcome")
		for _, tc := range []struct {
			name     string
			session  *HistoryPreparationSession
			owner    *nativeHistoryOwner
			capacity int
		}{
			{"foreign-session", otherSession, &nativeHistoryOwner{}, 1},
			{"missing-owner", session, nil, 1},
			{"retained-owner", session, &nativeHistoryOwner{region: &compilermemory.Region{}}, 1},
			{"closed-owner", session, &nativeHistoryOwner{closed: true, closeErr: closedError}, 1},
			{"zero-capacity", session, &nativeHistoryOwner{}, 0},
			{"negative-capacity", session, &nativeHistoryOwner{}, -1},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var before nativeHistoryOwner
				if tc.owner != nil {
					before = *tc.owner
				}
				if err := state.acquireOwner(tc.session, tc.owner, tc.capacity); !errors.Is(err, ErrConfiguration) || calls != 0 {
					t.Fatalf("invalid owner admission reached native acquisition: %v, calls=%d", err, calls)
				}
				if tc.owner != nil && (tc.owner.region != before.region || tc.owner.closed != before.closed || tc.owner.closeErr != before.closeErr) {
					t.Fatal("refused acquisition changed the retained owner or close outcome")
				}
			})
		}
	})
	for _, partial := range []bool{false, true} {
		origin := time.Now()
		action, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
		cleanup, stopCleanup := context.WithDeadline(context.Background(), origin.Add(2*time.Minute))
		acquisitionFailure := errors.New("simulated acquisition failure")
		calls := 0
		acquire := func(context.Context, context.Context, int) (*compilermemory.Region, error) {
			calls++
			if partial {
				return &compilermemory.Region{}, acquisitionFailure
			}
			return nil, acquisitionFailure
		}
		session, err := newNativeHistoryPreparationSession(action, cleanup, origin, len(raw), 4096, acquire)
		if err != nil {
			t.Fatal(err)
		}
		err = session.CompileOwned(raw)
		if !errors.Is(err, acquisitionFailure) || calls != 1 || !session.closed {
			t.Fatalf("acquisition refusal retried or disappeared: %v", err)
		}
		if partial && !errors.Is(err, ErrHistoryPreparationCleanup) {
			t.Fatal("partial invalid owner cleanup was not reported")
		}
		if _, err := session.FinishDiagnostic(); err == nil || calls != 1 {
			t.Fatal("failed native attempt fell back to Go")
		}
		closeErr := session.Close()
		if partial && !errors.Is(closeErr, ErrHistoryPreparationCleanup) {
			t.Fatal("partial cleanup outcome was not cached")
		}
		cancel()
		stopCleanup()
	}
	for _, invalid := range []struct {
		action, cleanup context.Context
	}{
		{nil, context.Background()}, {context.Background(), nil},
		{context.Background(), context.Background()},
	} {
		if session, err := NewNativeHistoryPreparationSession(invalid.action, invalid.cleanup, time.Now(), 1, 1); session != nil || err == nil {
			t.Fatal("native constructor accepted missing original finite contexts")
		}
	}
	if nodes, err := nativeHistorySchemaNodes(reflect.TypeFor[historyPreparationInput](), 0); err != nil || nodes <= 0 {
		t.Fatalf("fixed DTO node bound unavailable: %d, %v", nodes, err)
	}
	for _, schema := range []reflect.Type{nil, reflect.TypeFor[map[string]int](), reflect.TypeFor[*int](), reflect.TypeFor[[]int](), reflect.TypeFor[float64]()} {
		if _, err := nativeHistorySchemaNodes(schema, 0); err == nil {
			t.Fatal("unsupported native schema accepted")
		}
	}
	if _, err := nativeHistorySchemaNodes(reflect.TypeFor[int](), 65); err == nil {
		t.Fatal("schema depth guard changed")
	}
	if _, err := nativeHistorySchemaNodes(reflect.TypeFor[uint64](), math.MaxInt); err == nil {
		t.Fatal("oversized schema depth accepted")
	}
}
