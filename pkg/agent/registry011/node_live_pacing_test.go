//go:build registry_live

package registry011

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const liveReadMinimumSpacing = time.Second

type liveIdleTransport interface {
	http.RoundTripper
	CloseIdleConnections()
}

// This test-only scheduler spaces original attempts. Waiting consumes the
// existing request deadline. It never retries, reads a response body, changes
// an HTTP status, or supplies permission for a production history execution.
type liveReadPacer struct {
	base     liveIdleTransport
	spacing  time.Duration
	gate     chan struct{}
	earliest time.Time // accessed only while holding gate
}

func newLiveReadPacer(base liveIdleTransport) *liveReadPacer {
	return &liveReadPacer{base: base, spacing: liveReadMinimumSpacing, gate: make(chan struct{}, 1)}
}

func (p *liveReadPacer) CloseIdleConnections() { p.base.CloseIdleConnections() }

func (p *liveReadPacer) RoundTrip(request *http.Request) (*http.Response, error) {
	ctx := request.Context()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case p.gate <- struct{}{}:
	}
	defer func() { <-p.gate }()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if delay := time.Until(p.earliest); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	response, err := p.base.RoundTrip(request)
	// Start the next gap after this attempt returns headers or an error. The
	// response body is still owned exclusively by the original caller.
	p.earliest = time.Now().Add(p.spacing)
	return response, err
}

func checkLiveReadPacing(t *testing.T) {
	t.Helper()
	t.Run("spacing-preserves-status-and-body-without-retry", func(t *testing.T) {
		arrivals := make(chan time.Time, 2)
		var calls atomic.Int32
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case arrivals <- time.Now():
			default:
			}
			if calls.Add(1) == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = io.WriteString(w, "owned rate-limit response")
				return
			}
			_, _ = io.WriteString(w, "owned next response")
		}))
		defer server.Close()
		pacer := newLiveReadPacer(server.Client().Transport.(*http.Transport))
		pacer.spacing = 30 * time.Millisecond
		client := &http.Client{Transport: pacer, Timeout: 3 * time.Second}
		defer client.CloseIdleConnections()
		for i, want := range []string{"owned rate-limit response", "owned next response"} {
			response, err := client.Get(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			body, readErr := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			wantStatus := http.StatusOK
			if i == 0 {
				wantStatus = http.StatusTooManyRequests
			}
			if readErr != nil || closeErr != nil || response.StatusCode != wantStatus || string(body) != want {
				t.Fatalf("original response changed: status=%d body=%q read=%v close=%v", response.StatusCode, body, readErr, closeErr)
			}
		}
		first, second := <-arrivals, <-arrivals
		if calls.Load() != 2 || second.Sub(first) < pacer.spacing {
			t.Fatalf("attempt count or spacing changed: calls=%d gap=%s", calls.Load(), second.Sub(first))
		}
	})
	t.Run("deadline-during-spacing-sends-no-request", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()
		pacer := newLiveReadPacer(server.Client().Transport.(*http.Transport))
		pacer.spacing = time.Minute
		client := &http.Client{Transport: pacer, Timeout: 3 * time.Second}
		defer client.CloseIdleConnections()
		first, err := client.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		if err = first.Body.Close(); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if response != nil {
			_ = response.Body.Close()
		}
		if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 1 {
			t.Fatalf("expired wait issued another request: calls=%d error=%v", calls.Load(), err)
		}
	})
	t.Run("deadline-while-another-attempt-owns-gate", func(t *testing.T) {
		var calls atomic.Int32
		entered, release := make(chan struct{}), make(chan struct{})
		var releaseOnce sync.Once
		releaseOwner := func() { releaseOnce.Do(func() { close(release) }) }
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) == 1 {
				close(entered)
				<-release
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()
		defer releaseOwner()
		pacer := newLiveReadPacer(server.Client().Transport.(*http.Transport))
		client := &http.Client{Transport: pacer, Timeout: 3 * time.Second}
		defer client.CloseIdleConnections()
		ownerDone := make(chan error, 1)
		go func() {
			response, err := client.Get(server.URL)
			if response != nil {
				if closeErr := response.Body.Close(); err == nil {
					err = closeErr
				}
			}
			ownerDone <- err
		}()
		select {
		case <-entered:
		case <-time.After(3 * time.Second):
			t.Fatal("owned first attempt did not reach server")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if response != nil {
			_ = response.Body.Close()
		}
		if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 1 {
			t.Fatalf("gate wait issued another request: calls=%d error=%v", calls.Load(), err)
		}
		releaseOwner()
		if err = <-ownerDone; err != nil {
			t.Fatal("original attempt failed", err)
		}
	})
}
