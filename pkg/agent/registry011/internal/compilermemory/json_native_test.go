//go:build darwin && registry_native_limits

package compilermemory

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

func TestNativeJSONDocumentFeedsOriginalScalarConsumers(t *testing.T) {
	action, cancelAction := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelAction()
	cleanup, cancelCleanup := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelCleanup()
	r, err := Acquire(action, cleanup, 8192)
	if err != nil {
		if r != nil {
			err = errors.Join(err, r.Close())
		}
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	}()
	s, err := r.Store([]byte(`{"clock":{"maximum":18446744073709551615},"roles":["agent","mcp"]}`))
	if err != nil {
		t.Fatal(err)
	}
	d, err := r.ParseJSON(s, JSONLimits{8, 64})
	if err != nil {
		t.Fatal(err)
	}
	clock := jsonTestMember(t, d.Root(), "clock")
	maximum := jsonTestMember(t, clock, "maximum")
	if value, err := maximum.JSONUint64(); value != math.MaxUint64 || err != nil {
		t.Fatalf("native document integer: %d, %v", value, err)
	}
	roles := jsonTestMember(t, d.Root(), "roles")
	role, err := roles.Index(1)
	if err != nil {
		t.Fatal(err)
	}
	if equal, err := role.StringEqual("mcp"); !equal || err != nil {
		t.Fatalf("native document string: %v, %v", equal, err)
	}
	cancelAction()
	if _, err := maximum.JSONUint64(); !errors.Is(err, context.Canceled) {
		t.Fatalf("native document survived original cancellation: %v", err)
	}
}
