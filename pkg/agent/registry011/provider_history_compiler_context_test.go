package registry011

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// Cancel the actual backing context at a deterministic observation. Done and
// Err keep their ordinary sticky contract; no production clock is replaced.
type providerHistoryCompilerContextCancellation struct {
	context.Context
	cancel   context.CancelFunc
	cancelAt int64
	checks   atomic.Int64
}

func (c *providerHistoryCompilerContextCancellation) Err() error {
	if c.checks.Add(1) == c.cancelAt {
		c.cancel()
	}
	return c.Context.Err()
}

func TestProviderHistoryUniformCompilerKeepsUncancelledCosts(t *testing.T) {
	manifest, methods, certificates, _, limits := historyRoleSourceInputs(t)
	limits.legacy.objectBytes = 512 << 10
	want, ok := providerHistoryCollectionUniformSourceUpper(manifest, methods, certificates, 512<<10, limits)
	if !ok {
		t.Fatal("unchanged numeric component precursor")
	}
	finite, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, tc := range []struct {
		name string
		ctx  context.Context
	}{{"numeric", nil}, {"ordinary", context.Background()}, {"finite", finite}} {
		t.Run(tc.name, func(t *testing.T) {
			got, valid := providerHistoryCollectionUniformSourceUpperContext(tc.ctx, manifest, methods, certificates, 512<<10, limits)
			if !valid || got != want {
				t.Fatal("context observations changed a full source row", valid)
			}
		})
	}
}

func TestProviderHistoryUniformCompilerRefusesCanceledPartialCosts(t *testing.T) {
	for _, name := range []string{"canceled-before", "expired-before", "canceled-between-cuts"} {
		t.Run(name, func(t *testing.T) {
			manifest, methods, certificates, _, limits := historyRoleSourceInputs(t)
			limits.legacy.objectBytes = 512 << 10
			if manifest.counts.groups < 3 {
				t.Fatal("scenario requires several distinct original cuts")
			}
			original := *manifest
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			selected, backing := ctx, ctx
			switch name {
			case "canceled-before":
				cancel()
			case "expired-before":
				expired, stop := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				defer stop()
				selected, backing = expired, expired
			case "canceled-between-cuts":
				selected = &providerHistoryCompilerContextCancellation{Context: ctx, cancel: cancel, cancelAt: 3}
			}
			got, ok := providerHistoryCollectionUniformSourceUpperContext(selected, manifest, methods, certificates, 512<<10, limits)
			if ok || got != (providerHistoryUniformSourceCost{}) || backing.Err() == nil || *manifest != original {
				t.Fatal("cancellation leaked partial cost data or changed the selection", ok)
			}
			if _, valid := providerHistoryCollectionUniformSourceUpper(manifest, methods, certificates, 512<<10, limits); !valid {
				t.Fatal("a refused invocation damaged the original numeric input")
			}
		})
	}
}

func TestProviderHistoryOwnedCompilerStopsCanceledEnumeration(t *testing.T) {
	config := historySelectedSourceConfig(t)
	if config.bounds.history.collection.counts.groups < 16 {
		t.Fatal("scenario requires a nontrivial original enumeration")
	}
	original := *config.bounds.history.collection
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	selected := &providerHistoryCompilerContextCancellation{Context: ctx, cancel: cancel, cancelAt: 16}
	plan, err := providerHistoryCompileOwnedSourcePlan(selected, config, 1<<20, 1, 1)
	if plan != nil || !errors.Is(err, context.Canceled) || ctx.Err() != context.Canceled {
		t.Fatal("owned compilation failed to retain cancellation or leaked a plan", plan, err)
	}
	if *config.bounds.history.collection != original || config.bounds.history.derivation != nil {
		t.Fatal("canceled compilation changed the selection or minted authority")
	}
	if _, ok := providerHistoryCompileSourcePlan(config, historySelectedSourceClock(config)); !ok {
		t.Fatal("canceled compilation damaged the input of the separate numeric diagnostic")
	}
}
