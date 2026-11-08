package registry011

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
		ctx context.Context
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
		"format": func(s *HistoryPreparationSummary) { s.Format = "unknown" },
		"input": func(s *HistoryPreparationSummary) { s.InputSHA256 = strings.Repeat("b", 64) },
		"readiness": func(s *HistoryPreparationSummary) { s.SourceReady = true },
		"origin": func(s *HistoryPreparationSummary) { s.OriginUnixNano = 0 },
		"deadline": func(s *HistoryPreparationSummary) { s.DeadlineUnixNano = s.OriginUnixNano },
		"manifest": func(s *HistoryPreparationSummary) { s.ManifestSHA256 = "missing" },
		"policy": func(s *HistoryPreparationSummary) { s.Identity.Policy = "missing" },
		"instance": func(s *HistoryPreparationSummary) { s.Identity.Instance = "missing" },
		"provider": func(s *HistoryPreparationSummary) { s.Identity.Provider = "missing" },
		"namespace": func(s *HistoryPreparationSummary) { s.Identity.Namespace = "missing" },
		"source": func(s *HistoryPreparationSummary) { s.Identity.SourceID = "missing" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := summary
			change(&changed)
			if out, err := DecodeHistoryPreparationSummary(context.Background(), encode(changed), input, len(raw)+100); err == nil || out.Format != "" {
				t.Fatal("unbound or authority-claiming DATA accepted")
			}
		})
	}
	for _, maximum := range []int{0, len(raw)-1, 64<<20 + 1} {
		if _, err := DecodeHistoryPreparationSummary(context.Background(), raw, input, maximum); err == nil {
			t.Fatal("invalid output bound accepted")
		}
	}
	if _, err := DecodeHistoryPreparationSummary(context.Background(), []byte(`{}`), input, len(raw)); err == nil {
		t.Fatal("missing result members accepted")
	}
}
