package registry011

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func historyOriginalBudget(t *testing.T, p providerHistoryOriginalPlan, role uint8, count uint64) *providerHistoryInputBudget {
	t.Helper()
	u, err := p.upper()
	if err != nil {
		t.Fatal(err)
	}
	var envelope providerHistoryInputEnvelope
	envelope[providerReadEvent] = providerHistoryInputRow{8192, count * u.readBytes, count * u.readCalls}
	envelope[providerHashEvent] = providerHistoryInputRow{8192, count * u.hashBytes, count * u.hashWrites}
	b, err := newProviderHistoryInputBudget(admissionTestDigest("a"), role, envelope)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestProviderHistoryOriginalBudgetRechargesIndependentAndRepeatedReads(t *testing.T) {
	d := historyOriginalDisk(t)
	payload := bytes.Repeat([]byte("a"), 8193)
	p := historyOriginalPlan(t, "response", payload)
	if err := os.WriteFile(filepath.Join(d.directory, providerObjectName(p.reference.kind, p.reference.digest)), payload, 0600); err != nil {
		t.Fatal(err)
	}
	u, _ := p.upper()
	for role := uint8(0); role < 2; role++ {
		b, r := historyOriginalBudget(t, p, role, 2), historyOriginalReader(t, d)
		for i := 0; i < 2; i++ {
			if digest, err := r.scanWithInputBudget(context.Background(), p, b); err != nil || digest != p.reference.digest {
				t.Fatal("normal funded integrity scan failed", err)
			}
		}
		if b.used[providerReadEvent] != (providerHistoryInputUse{2 * u.readBytes, 2 * u.readCalls}) ||
			b.used[providerHashEvent] != (providerHistoryInputUse{2 * u.hashBytes, 2 * u.hashWrites}) || b.failed {
			t.Fatal("repeated input was shared, deduplicated or refunded")
		}
	}
}

func TestProviderHistoryOriginalBudgetRefusesBeforeOriginalIO(t *testing.T) {
	p := historyOriginalPlan(t, "response", []byte("abc"))
	// No original is created. Refusal must precede even opening that file.
	for _, state := range []string{"read-width", "hash-width", "read-bytes", "hash-calls"} {
		t.Run(state, func(t *testing.T) {
			d := historyOriginalDisk(t)
			b := historyOriginalBudget(t, p, 0, 1)
			envelope := b.envelope
			switch state {
			case "read-width":
				envelope[providerReadEvent].width = 8191
			case "hash-width":
				envelope[providerHashEvent].width = 8191
			case "read-bytes":
				envelope[providerReadEvent].bytes--
			case "hash-calls":
				envelope[providerHashEvent] = providerHistoryInputRow{}
			}
			b = historyInputBudget(t, envelope)
			r := historyOriginalReader(t, d)
			if digest, err := r.scanWithInputBudget(context.Background(), p, b); err == nil || digest != "" || r.failed || !b.failed || b.used != (providerHistoryInputUses{}) {
				t.Fatal("insufficient input allowance reached the reader")
			}
		})
	}
}

func TestProviderHistoryOriginalBudgetKeepsFailedAttemptCharged(t *testing.T) {
	d := historyOriginalDisk(t)
	p := historyOriginalPlan(t, "response", []byte("abc"))
	b, r := historyOriginalBudget(t, p, 0, 2), historyOriginalReader(t, d)
	if digest, err := r.scanWithInputBudget(context.Background(), p, b); err == nil || digest != "" || !r.failed || !b.failed {
		t.Fatal("missing original became successful integrity evidence")
	}
	claim, _ := providerHistoryOriginalInputClaim(p)
	if b.used != claim {
		t.Fatal("failed prefix refunded its full original-input upper")
	}
	next := historyOriginalReader(t, d)
	if digest, err := next.scanWithInputBudget(context.Background(), p, b); err == nil || digest != "" || next.failed || b.used != claim {
		t.Fatal("another reader recreated failed input capacity")
	}
}

func TestProviderHistoryOriginalBudgetRejectsIncompletePreflight(t *testing.T) {
	d := historyOriginalDisk(t)
	p := historyOriginalPlan(t, "response", []byte("abc"))
	b, r := historyOriginalBudget(t, p, 0, 1), historyOriginalReader(t, d)
	for _, row := range []struct {
		r *providerHistoryOriginalReader
		c context.Context
		p providerHistoryOriginalPlan
		b *providerHistoryInputBudget
	}{
		{nil, context.Background(), p, b},
		{r, nil, p, b},
		{r, context.Background(), providerHistoryOriginalPlan{}, b},
		{r, context.Background(), p, nil},
	} {
		if digest, err := row.r.scanWithInputBudget(row.c, row.p, row.b); err == nil || digest != "" || b.failed || b.used != (providerHistoryInputUses{}) || r.failed {
			t.Fatal("incomplete preflight spent input allowance or opened the reader")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if digest, err := r.scanWithInputBudget(ctx, p, b); err == nil || digest != "" || !r.failed || !b.failed || b.used == (providerHistoryInputUses{}) {
		t.Fatal("cancelled admitted attempt was not charged and closed")
	}
}

func TestProviderHistoryOriginalBudgetSerializesNormalConcurrentReads(t *testing.T) {
	d := historyOriginalDisk(t)
	payload := []byte("abc")
	p := historyOriginalPlan(t, "response", payload)
	if err := os.WriteFile(filepath.Join(d.directory, providerObjectName(p.reference.kind, p.reference.digest)), payload, 0600); err != nil {
		t.Fatal(err)
	}
	b, r := historyOriginalBudget(t, p, 0, 2), historyOriginalReader(t, d)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Go(func() {
			if digest, err := r.scanWithInputBudget(context.Background(), p, b); err != nil || digest != p.reference.digest {
				t.Errorf("normal serialized scan failed: %v", err)
			}
		})
	}
	wg.Wait()
	claim, _ := providerHistoryOriginalInputClaim(p)
	if b.failed || b.used[providerReadEvent].bytes != 2*claim[providerReadEvent].bytes || b.used[providerHashEvent].bytes != 2*claim[providerHashEvent].bytes {
		t.Fatal("serialized normal reads shared one input charge")
	}
}
