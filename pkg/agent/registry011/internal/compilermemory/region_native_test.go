//go:build darwin && registry_native_limits

package compilermemory

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

func nativeRegion(t *testing.T, capacity int) (*Region, context.CancelFunc) {
	t.Helper()
	action, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	cleanup, cancelCleanup := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
	t.Cleanup(cancelCleanup)
	t.Cleanup(cancel)
	region, err := Acquire(action, cleanup, capacity)
	if region != nil {
		t.Cleanup(func() {
			if err := region.Close(); err != nil {
				t.Error("original region cleanup", err)
			}
		})
	}
	if err != nil || region == nil {
		t.Fatal("actual native acquisition", err)
	}
	if !region.locked || len(region.data)%os.Getpagesize() != 0 {
		t.Fatal("actual page backing absent")
	}
	return region, cancel
}

func TestNativeRegionRetainsAndSealsOriginalBytes(t *testing.T) {
	region, _ := nativeRegion(t, os.Getpagesize()+7)
	original := []byte("owned compiler input")
	span, err := region.Store(original)
	if err != nil {
		t.Fatal(err)
	}
	clear(original)
	result := make([]byte, len("owned compiler input"))
	if err := region.Read(span, result); err != nil || string(result) != "owned compiler input" {
		t.Fatal("original bytes were not owned", err)
	}
	part, err := region.Slice(span, 6, 8)
	if err != nil {
		t.Fatal(err)
	}
	if equal, err := region.Equal(part, []byte("compiler")); err != nil || !equal {
		t.Fatal("bounded native view", equal, err)
	}
	if equal, err := region.Equal(part, []byte("different")); err != nil || equal {
		t.Fatal("native comparison mismatch", equal, err)
	}
	for _, limits := range [][2]int{{-1, 1}, {0, -1}, {span.length + 1, 0}, {span.length, 1}} {
		if _, err := region.Slice(span, limits[0], limits[1]); !errors.Is(err, ErrRange) {
			t.Fatal("invalid native view", limits, err)
		}
	}
	if err := region.Read(span, make([]byte, span.length-1)); !errors.Is(err, ErrRange) {
		t.Fatal("partial destination accepted", err)
	}
	if _, err := region.Store(make([]byte, region.capacity-region.used+1)); !errors.Is(err, ErrRange) {
		t.Fatal("declared capacity changed by page rounding", err)
	}
	if err := region.Seal(); err != nil {
		t.Fatal(err)
	}
	if _, err := region.Store([]byte("changed")); !errors.Is(err, ErrRange) {
		t.Fatal("sealed bytes changed", err)
	}
	if err := region.Close(); err != nil {
		t.Fatal(err)
	}
	if region.data != nil || region.locked {
		t.Fatal("successful close retained native backing")
	}
	if err := region.Read(span, result); !errors.Is(err, ErrClosed) {
		t.Fatal("stale span read", err)
	}
}

func TestNativeRegionCancellationKeepsOriginalCleanup(t *testing.T) {
	region, cancel := nativeRegion(t, 64)
	span, err := region.Store([]byte("retained"))
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := region.Store([]byte("later")); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled work entered", err)
	}
	if err := region.Read(span, make([]byte, 8)); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled read entered", err)
	}
	if err := region.Close(); err != nil {
		t.Fatal("separate original cleanup was canceled", err)
	}
	if err := region.Close(); err != nil {
		t.Fatal("close repeated native work", err)
	}
}

func TestNativeRegionConcurrentReadAndClose(t *testing.T) {
	region, _ := nativeRegion(t, 64)
	span, err := region.Store([]byte("data"))
	if err != nil {
		t.Fatal(err)
	}
	if err := region.Seal(); err != nil {
		t.Fatal(err)
	}
	var readers sync.WaitGroup
	for range 16 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			result := make([]byte, 4)
			err := region.Read(span, result)
			if err == nil && !bytes.Equal(result, []byte("data")) {
				t.Error("native bytes changed during owned access")
			}
			if err != nil && !errors.Is(err, ErrClosed) {
				t.Error("unexpected access outcome", err)
			}
		}()
	}
	if err := region.Close(); err != nil {
		t.Fatal(err)
	}
	readers.Wait()
}
