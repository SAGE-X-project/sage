package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProviderHistoryPreparationSessionCannotRetryOrRestore(t *testing.T) {
	origin := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
	defer cancel()
	session, err := NewHistoryPreparationSession(ctx, origin, 1024, 2048)
	if err != nil {
		t.Fatal(err)
	}
	copied := new(HistoryPreparationSession)
	reflect.ValueOf(copied).Elem().Set(reflect.ValueOf(session).Elem())
	for _, invalid := range []*HistoryPreparationSession{nil, {}, copied} {
		if output, err := invalid.Prepare([]byte(`{}`)); err == nil || output != nil {
			t.Fatal("non-owner session accepted")
		}
	}
	if err := json.Unmarshal([]byte(`{}`), new(HistoryPreparationSession)); err == nil {
		t.Fatal("session restored from JSON")
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			if output, err := session.Prepare([]byte(`{}`)); err == nil || output != nil {
				t.Error("invalid input produced prepared output")
			}
		})
	}
	workers.Wait()
	if !session.used.Load() {
		t.Fatal("failed attempt was refunded")
	}
	if output, err := session.Prepare([]byte(`null`)); err == nil || output != nil {
		t.Fatal("spent attempt accepted")
	}
}

func TestProviderHistoryPreparationSessionRequiresOriginalFiniteWindow(t *testing.T) {
	origin := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
	defer cancel()
	for _, limits := range [][2]int{{0, 1}, {1, 0}, {64<<20 + 1, 1}, {1, 64<<20 + 1}} {
		if session, err := NewHistoryPreparationSession(ctx, origin, limits[0], limits[1]); err == nil || session != nil {
			t.Fatal("invalid explicit bounds accepted")
		}
	}
	for _, c := range []struct {
		ctx    context.Context
		origin time.Time
	}{{nil, origin}, {context.Background(), origin}, {ctx, time.Time{}}, {ctx, origin.Add(2 * time.Minute)}} {
		if session, err := NewHistoryPreparationSession(c.ctx, c.origin, 1, 1); err == nil || session != nil {
			t.Fatal("missing original finite window accepted")
		}
	}
	for _, input := range [][]byte{nil, []byte("too long")} {
		session, err := NewHistoryPreparationSession(ctx, origin, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		if out, err := session.Prepare(input); err == nil || out != nil || !session.used.Load() {
			t.Fatal("input refusal lost spent state")
		}
	}
	cancel()
	if session, err := NewHistoryPreparationSession(ctx, origin, 1, 1); err == nil || session != nil {
		t.Fatal("canceled original window accepted")
	}
}

func TestProviderHistoryPreparationSummaryRemainsNonAuthorityData(t *testing.T) {
	input := []byte("explicit bounded DATA")
	hash := sha256.Sum256(input)
	digest := "0x" + strings.Repeat("a", 64)
	identity := ProviderCandidateIdentity{Policy: digest, Instance: digest, Provider: digest, Namespace: digest, SourceID: digest, Generation: 1}
	summary := HistoryPreparationSummary{
		Format: historyPreparationResultFormat, InputSHA256: hex.EncodeToString(hash[:]),
		Identity: identity, ManifestSHA256: digest, OriginUnixNano: 1, DeadlineUnixNano: 2,
	}
	encode := func(value HistoryPreparationSummary) []byte {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	raw := encode(summary)
	out, err := DecodeHistoryPreparationSummary(context.Background(), raw, input, len(raw))
	if err != nil || out != summary || out.SourceReady {
		t.Fatalf("diagnostic DATA rejected: %v", err)
	}
	for name, change := range map[string]func(*HistoryPreparationSummary){
		"format":    func(s *HistoryPreparationSummary) { s.Format = "unknown" },
		"input":     func(s *HistoryPreparationSummary) { s.InputSHA256 = strings.Repeat("b", 64) },
		"readiness": func(s *HistoryPreparationSummary) { s.SourceReady = true },
		"origin":    func(s *HistoryPreparationSummary) { s.OriginUnixNano = 0 },
		"deadline":  func(s *HistoryPreparationSummary) { s.DeadlineUnixNano = s.OriginUnixNano },
		"manifest":  func(s *HistoryPreparationSummary) { s.ManifestSHA256 = "missing" },
		"policy":    func(s *HistoryPreparationSummary) { s.Identity.Policy = "missing" },
		"instance":  func(s *HistoryPreparationSummary) { s.Identity.Instance = "missing" },
		"provider":  func(s *HistoryPreparationSummary) { s.Identity.Provider = "missing" },
		"namespace": func(s *HistoryPreparationSummary) { s.Identity.Namespace = "missing" },
		"source":    func(s *HistoryPreparationSummary) { s.Identity.SourceID = "missing" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := summary
			change(&changed)
			if out, err := DecodeHistoryPreparationSummary(context.Background(), encode(changed), input, len(raw)+100); err == nil || out.Format != "" {
				t.Fatal("unbound or authority-claiming DATA accepted")
			}
		})
	}
	for _, maximum := range []int{0, len(raw) - 1, 64<<20 + 1} {
		if _, err := DecodeHistoryPreparationSummary(context.Background(), raw, input, maximum); err == nil {
			t.Fatal("invalid output bound accepted")
		}
	}
	if _, err := DecodeHistoryPreparationSummary(context.Background(), []byte(`{}`), input, len(raw)); err == nil {
		t.Fatal("missing result members accepted")
	}
}

func historyPreparationHandoffInput(t *testing.T) ([]byte, ProviderCandidateIdentity, string) {
	t.Helper()
	in, identity, manifest := historyPreparationInputFromOriginalTestAPI(t)
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	return raw, identity, manifest
}

func TestProviderHistoryPreparationRetainsOriginalPlanAfterInputRelease(t *testing.T) {
	origin := time.Now()
	deadline := origin.Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	raw, identity, manifest := historyPreparationHandoffInput(t)
	original := bytes.Clone(raw)
	session, err := NewHistoryPreparationSession(ctx, origin, 64<<20, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	compiler := session.compiler
	if err := session.CompileOwned(raw); err != nil {
		t.Fatal(err)
	}
	plan := session.plan
	if plan == nil || plan.clock == nil || session.compiler != compiler || !session.used.Load() || session.closed ||
		!compiler.origin.Equal(origin) || !plan.compilerDeadline.Equal(deadline) || plan.config.bounds.history.derivation != nil {
		t.Fatal("compilation lost the original unqualified selection or lifetime")
	}
	clock := plan.clock
	clear(raw) // The producer releases its bytes before the next session phase.
	if session.plan != plan || session.plan.clock != clock || session.plan.config.identity != identity ||
		session.plan.config.bounds.history.collection.digest != manifest {
		t.Fatal("input release replaced the retained plan or clock")
	}
	output, err := session.FinishDiagnostic()
	if err != nil {
		t.Fatal(err)
	}
	summary, err := DecodeHistoryPreparationSummary(ctx, output, original, 64<<10)
	if err != nil || summary.SourceReady || summary.Identity != identity || summary.ManifestSHA256 != manifest ||
		summary.OriginUnixNano != origin.UnixNano() || summary.DeadlineUnixNano != deadline.UnixNano() {
		t.Fatal("handoff changed the input binding, selection or original window", err)
	}
	if _, err := DecodeHistoryPreparationSummary(ctx, output, raw, 64<<10); err == nil {
		t.Fatal("diagnostic result rebound itself to the released input")
	}
	if session.plan != nil || session.compiler != nil || session.inputSHA256 != "" || !session.closed {
		t.Fatal("finished selection remained available for a second consumer")
	}
	if err := session.CompileOwned(original); err == nil {
		t.Fatal("finished attempt compiled again")
	}
	if output, err := session.FinishDiagnostic(); err == nil || output != nil {
		t.Fatal("finished selection was consumed again")
	}
}

type historyPreparationMutableDeadline struct {
	context.Context
	deadline time.Time
}

func (c *historyPreparationMutableDeadline) Deadline() (time.Time, bool) { return c.deadline, true }

func TestProviderHistoryPreparationHandoffRefusesChangedLifetimeAndDisposal(t *testing.T) {
	raw, _, _ := historyPreparationHandoffInput(t)
	for _, name := range []string{"canceled", "canceled-before-publication", "changed-deadline", "result-limit", "closed"} {
		t.Run(name, func(t *testing.T) {
			origin := time.Now()
			ctx, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
			defer cancel()
			selected := ctx
			mutable := &historyPreparationMutableDeadline{Context: ctx, deadline: origin.Add(time.Minute)}
			probe := &providerHistoryCompilerContextCancellation{Context: ctx, cancel: cancel}
			maximumResult := 64 << 10
			switch name {
			case "changed-deadline":
				selected = mutable
			case "canceled-before-publication":
				selected = probe
			case "result-limit":
				maximumResult = 1
			}
			session, err := NewHistoryPreparationSession(selected, origin, 64<<20, maximumResult)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = session.Close() }()
			if err := session.CompileOwned(raw); err != nil {
				t.Fatal(err)
			}
			want := ErrConfiguration
			switch name {
			case "canceled":
				cancel()
				want = context.Canceled
			case "canceled-before-publication":
				probe.cancelAt = probe.checks.Load() + 2
				want = context.Canceled
			case "changed-deadline":
				mutable.deadline = origin.Add(2 * time.Minute)
			case "closed":
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if output, err := session.FinishDiagnostic(); !errors.Is(err, want) || output != nil {
				t.Fatal("changed or discarded selection produced output", err)
			}
			if !session.closed || !session.used.Load() || session.plan != nil || session.compiler != nil || session.inputSHA256 != "" {
				t.Fatal("refused handoff retained a reusable selection")
			}
			if err := session.Close(); err != nil {
				t.Fatal("terminal close was not idempotent", err)
			}
			if err := session.CompileOwned(raw); err == nil {
				t.Fatal("handoff refusal refunded its original attempt")
			}
			if output, err := session.FinishDiagnostic(); err == nil || output != nil {
				t.Fatal("refused selection was consumed again")
			}
		})
	}
}

func TestProviderHistoryPreparationHandoffRejectsNonOwnersAndWrongOrder(t *testing.T) {
	origin := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
	defer cancel()
	session, err := NewHistoryPreparationSession(ctx, origin, 1024, 2048)
	if err != nil {
		t.Fatal(err)
	}
	copied := new(HistoryPreparationSession)
	reflect.ValueOf(copied).Elem().Set(reflect.ValueOf(session).Elem())
	for _, invalid := range []*HistoryPreparationSession{nil, {}, copied} {
		if err := invalid.CompileOwned([]byte(`{}`)); err == nil {
			t.Fatal("non-owner compiled a selection")
		}
		if output, err := invalid.FinishDiagnostic(); err == nil || output != nil {
			t.Fatal("non-owner consumed a selection")
		}
		if err := invalid.Close(); err == nil {
			t.Fatal("non-owner closed a selection")
		}
	}
	if output, err := session.FinishDiagnostic(); err == nil || output != nil || !session.used.Load() {
		t.Fatal("finish-before-compile did not refuse and consume the attempt")
	}
	if err := session.CompileOwned([]byte(`{}`)); err == nil {
		t.Fatal("wrong-order call refunded the attempt")
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestProviderHistoryPreparationHandoffHasOneConcurrentProducerAndConsumer(t *testing.T) {
	origin := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
	defer cancel()
	raw, _, _ := historyPreparationHandoffInput(t)
	session, err := NewHistoryPreparationSession(ctx, origin, 64<<20, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	var workers sync.WaitGroup
	compiled := make(chan error, 8)
	for range cap(compiled) {
		workers.Go(func() { compiled <- session.CompileOwned(raw) })
	}
	workers.Wait()
	close(compiled)
	producers := 0
	for err := range compiled {
		if err == nil {
			producers++
		} else if !errors.Is(err, ErrConfiguration) {
			t.Fatal(err)
		}
	}
	if producers != 1 || session.plan == nil {
		t.Fatal("concurrent calls lost or duplicated the original compiled selection", producers)
	}
	type result struct {
		output []byte
		err    error
	}
	finished := make(chan result, 8)
	for range cap(finished) {
		workers.Go(func() {
			output, err := session.FinishDiagnostic()
			finished <- result{output, err}
		})
	}
	workers.Wait()
	close(finished)
	consumers := 0
	for got := range finished {
		if got.err == nil {
			consumers++
			if summary, err := DecodeHistoryPreparationSummary(ctx, got.output, raw, 64<<10); err != nil || summary.SourceReady {
				t.Fatal("concurrent winner returned unbound or authority-claiming output", err)
			}
		} else if !errors.Is(got.err, ErrConfiguration) || got.output != nil {
			t.Fatal("refused concurrent consumer returned output", got.err)
		}
	}
	if consumers != 1 || !session.closed || session.plan != nil || session.compiler != nil {
		t.Fatal("concurrent diagnostic consumption duplicated or retained the plan", consumers)
	}
}

func TestProviderHistoryPreparationCloseSerializesWithCompilationAndFinish(t *testing.T) {
	origin := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
	defer cancel()
	raw, _, _ := historyPreparationHandoffInput(t)
	session, err := NewHistoryPreparationSession(ctx, origin, 64<<20, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Go(func() {
		<-start
		if err := session.CompileOwned(raw); err != nil && !errors.Is(err, ErrConfiguration) {
			t.Error(err)
		}
	})
	workers.Go(func() {
		<-start
		output, err := session.FinishDiagnostic()
		if err != nil && (!errors.Is(err, ErrConfiguration) || output != nil) {
			t.Error("wrong-order or closed finish returned an unexpected result", err)
		}
	})
	workers.Go(func() {
		<-start
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	})
	close(start)
	workers.Wait()
	if !session.closed || !session.used.Load() || session.plan != nil || session.compiler != nil || session.inputSHA256 != "" {
		t.Fatal("a competing phase revived a closed selection")
	}
	if err := session.CompileOwned(raw); err == nil {
		t.Fatal("close refunded the original attempt")
	}
}
