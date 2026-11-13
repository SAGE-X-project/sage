//go:build registry_live

package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"
)

func checkLiveCaptureReadLimits(t *testing.T) {
	t.Helper()
	for _, limit := range []int{0, 3, 64} {
		meta := map[string]any{}
		saved := 0
		body := &liveCaptureBody{ReadCloser: io.NopCloser(bytes.NewReader([]byte("fixed-public-data"))), owner: &liveCaptureTransport{dir: t.TempDir(), t: t, remaining: limit}, meta: meta, name: "owned-unit.bin", save: func() { saved++ }}
		got, e := io.ReadAll(body)
		if e != nil || string(got) != "fixed-public-data" {
			t.Fatal("passive capture changed read", e)
		}
		if e = body.Close(); e != nil {
			t.Fatal(e)
		}
		if e = body.Close(); e != nil {
			t.Fatal(e)
		}
		if saved != 1 || meta["response_bytes"] != min(limit, len(got)) || meta["capture_complete"] != (limit >= len(got)) {
			t.Fatal("mislabelled diagnostic limit", meta)
		}
	}
}

// Retain only this public scheduling hint, never authentication headers or
// cookies. These are bounded parsed header values, not a raw HTTP transcript
// or permission to retry, extend a deadline, or accept the failed response.
// Each decoded value is valid UTF-8 within the byte cap; JSON escaping has a
// separate encoded length. Invalid bytes in the bounded prefix are omitted
// and explicitly marked sanitized and incomplete.
type liveRetryAfterDiagnostic struct {
	Present   bool     `json:"present"`
	Complete  bool     `json:"complete"`
	Sanitized bool     `json:"sanitized"`
	Values    []string `json:"values"`
}

func liveCaptureRetryAfter(header http.Header) liveRetryAfterDiagnostic {
	const maximumValues, maximumValueBytes = 4, 1024
	values := header.Values("Retry-After")
	result := liveRetryAfterDiagnostic{
		Present: len(values) != 0, Complete: len(values) <= maximumValues,
		Values: make([]string, 0, min(len(values), maximumValues)),
	}
	for _, value := range values[:min(len(values), maximumValues)] {
		if len(value) > maximumValueBytes {
			value = value[:maximumValueBytes]
			result.Complete = false
		}
		// Bound BEFORE normalization, including a rune cut at the byte limit.
		// Removing invalid bytes cannot expand the retained decoded value.
		if !utf8.ValidString(value) {
			value = strings.ToValidUTF8(value, "")
			result.Sanitized = true
			result.Complete = false
		}
		// A short diagnostic must not retain a much larger header backing.
		result.Values = append(result.Values, strings.Clone(value))
	}
	return result
}

func checkLiveCaptureRetryAfter(t *testing.T) {
	t.Helper()
	for _, tc := range []struct {
		name      string
		values    []string
		want      []string
		complete  bool
		sanitized bool
	}{
		{"absent", nil, nil, true, false},
		{"empty-value", []string{""}, []string{""}, true, false},
		{"seconds", []string{"60"}, []string{"60"}, true, false},
		{"date", []string{"Fri, 06 Nov 2026 00:00:00 GMT"}, []string{"Fri, 06 Nov 2026 00:00:00 GMT"}, true, false},
		{"value-limit", []string{"1", "2", "3", "4", "5"}, []string{"1", "2", "3", "4"}, false, false},
		{"byte-limit", []string{strings.Repeat("7", 1025)}, []string{strings.Repeat("7", 1024)}, false, false},
		{"valid-multibyte-limit", []string{strings.Repeat("é", 512)}, []string{strings.Repeat("é", 512)}, true, false},
		{"invalid-prefix-limit", []string{strings.Repeat(string([]byte{0xff}), 1024)}, []string{""}, false, true},
		{"invalid-prefix-byte", []string{string([]byte{'6', 0xff, '0'})}, []string{"60"}, false, true},
		{"multibyte-cut", []string{strings.Repeat("a", 1023) + "é"}, []string{strings.Repeat("a", 1023)}, false, true},
		{"json-escaping", []string{strings.Repeat("<", 1024)}, []string{strings.Repeat("<", 1024)}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			header := http.Header{"Retry-After": tc.values, "Set-Cookie": []string{"unrelated-owned-cookie"}}
			got := liveCaptureRetryAfter(header)
			if got.Present != (len(tc.values) != 0) || got.Complete != tc.complete || got.Sanitized != tc.sanitized || !slices.Equal(got.Values, tc.want) {
				t.Fatal("incorrect bounded Retry-After diagnostic", got)
			}
			raw, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			var stored liveRetryAfterDiagnostic
			if err = json.Unmarshal(raw, &stored); err != nil {
				t.Fatal(err)
			}
			if stored.Present != got.Present || stored.Complete != got.Complete || stored.Sanitized != got.Sanitized || !slices.Equal(stored.Values, tc.want) {
				t.Fatal("persisted diagnostic changed its values or scope", stored)
			}
			for _, value := range stored.Values {
				if len(value) > 1024 || !utf8.ValidString(value) {
					t.Fatal("persisted decoded diagnostic escaped its byte or UTF-8 bound")
				}
			}
			if len(tc.values) != 0 {
				header.Values("Retry-After")[0] = "changed owned header"
				if !slices.Equal(got.Values, tc.want) {
					t.Fatal("diagnostic retained mutable header slice")
				}
			}
		})
	}
	t.Run("failed-response-keeps-status-without-reading-body", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.Header().Set("Retry-After", "60")
			w.Header().Set("Set-Cookie", "unrelated-owned-cookie")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, "owned rate-limit response")
		}))
		defer server.Close()
		dir := t.TempDir()
		capture := &liveCaptureTransport{base: server.Client().Transport.(*http.Transport), dir: dir, t: t, remaining: 1024}
		client := &http.Client{Transport: capture, Timeout: 3 * time.Second}
		defer client.CloseIdleConnections()
		response, err := client.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		closeErr := response.Body.Close() // Match the real caller's unread refusal.
		if closeErr != nil || response.StatusCode != http.StatusTooManyRequests || response.Header.Get("Retry-After") != "60" || calls.Load() != 1 {
			t.Fatal("capture changed the original failed response or retried", response.StatusCode, closeErr, calls.Load())
		}
		raw, err := os.ReadFile(filepath.Join(dir, "diagnostic-001.json"))
		if err != nil {
			t.Fatal(err)
		}
		var record struct {
			Status   int                          `json:"accepted_http_status_not_asserted"`
			Hint     liveRetryAfterDiagnostic     `json:"retry_after_header_diagnostic"`
			Bytes    int                          `json:"response_bytes"`
			Complete bool                         `json:"capture_complete"`
			Context  liveRequestContextDiagnostic `json:"request_context_diagnostic"`
		}
		if err = json.Unmarshal(raw, &record); err != nil {
			t.Fatal(err)
		}
		if record.Status != http.StatusTooManyRequests || !record.Hint.Present || !record.Hint.Complete || record.Hint.Sanitized || !slices.Equal(record.Hint.Values, []string{"60"}) || record.Bytes != 0 || record.Complete || bytes.Contains(raw, []byte("unrelated-owned-cookie")) {
			t.Fatal("failed response diagnostic lost its scope or retained unrelated headers", record)
		}
		if record.Context.DeadlineUTC == "" || record.Context.StateBefore != "none" || record.Context.StateAfter != "none" ||
			record.Context.TransportDeadlineExceeded || record.Context.TransportCanceled || record.Context.TransportTimeout || record.Context.ElapsedNS < 0 {
			t.Fatal("received HTTP refusal was mislabelled as transport cancellation", record.Context)
		}
	})
}

// A failed local transport and a canceled request are separate observations.
// The private dial stub returns an owned error without opening any connection;
// these scenarios cannot submit an RPC request or establish live evidence.
func checkLiveCaptureContextDiagnostics(t *testing.T) {
	t.Helper()
	refusal := errors.New("owned transport refusal")
	for _, tc := range []struct {
		name, mode                           string
		transportErr                         error
		state                                string
		deadline, exceeded, canceled, timeout bool
	}{
		{"live-transport-refusal", "live", refusal, "none", false, false, false, false},
		{"canceled-request", "canceled", refusal, "canceled", false, false, true, false},
		{"expired-deadline", "expired", refusal, "deadline_exceeded", true, true, false, true},
		{"live-request-transport-deadline", "live", context.DeadlineExceeded, "none", false, true, false, true},
		{"live-deadline-transport-refusal", "deadline", refusal, "none", true, false, false, false},
		{"canceled-with-deadline", "canceled-deadline", refusal, "canceled", true, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			cancel := func() {}
			switch tc.mode {
			case "canceled":
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "expired":
				ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-time.Hour))
			case "deadline", "canceled-deadline":
				ctx, cancel = context.WithDeadline(ctx, time.Now().Add(time.Hour))
				if tc.mode == "canceled-deadline" {
					cancel()
				}
			}
			defer cancel()
			base := &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) {
				return nil, tc.transportErr
			}}
			defer base.CloseIdleConnections()
			dir := t.TempDir()
			capture := &liveCaptureTransport{base: base, dir: dir, t: t, remaining: 1024}
			request, err := http.NewRequestWithContext(ctx, http.MethodPost, SepoliaRPCURL, nil)
			if err != nil {
				t.Fatal(err)
			}
			wantErr := tc.transportErr
			if ctx.Err() != nil {
				wantErr = ctx.Err()
			}
			response, err := capture.RoundTrip(request)
			if response != nil || !errors.Is(err, wantErr) || capture.sequence != 1 || capture.remaining != 1024 {
				t.Fatal("passive context capture changed the original refusal or body budget", response, err)
			}
			raw, err := os.ReadFile(filepath.Join(dir, "diagnostic-001.json"))
			if err != nil {
				t.Fatal(err)
			}
			var record struct {
				Context    liveRequestContextDiagnostic `json:"request_context_diagnostic"`
				Diagnostic bool                         `json:"diagnostic_only"`
				Complete   bool                         `json:"capture_complete"`
			}
			if err := json.Unmarshal(raw, &record); err != nil {
				t.Fatal(err)
			}
			got := record.Context
			if !record.Diagnostic || record.Complete || got.StateBefore != tc.state || got.StateAfter != tc.state || got.ElapsedNS < 0 ||
				got.TransportDeadlineExceeded != tc.exceeded || got.TransportCanceled != tc.canceled || got.TransportTimeout != tc.timeout {
				t.Fatal("context and transport observations were conflated or accepted", record)
			}
			deadline, hasDeadline := ctx.Deadline()
			if hasDeadline != tc.deadline || (got.DeadlineUTC != "") != tc.deadline ||
				tc.deadline && got.DeadlineUTC != deadline.UTC().Format(time.RFC3339Nano) {
				t.Fatal("diagnostic changed or invented the original request deadline", got.DeadlineUTC)
			}
			if bytes.Contains(raw, []byte("accepted_http_status_not_asserted")) || bytes.Contains(raw, []byte("response_file")) {
				t.Fatal("unreceived response was presented as an HTTP observation")
			}
		})
	}
}

// Test-only passive capture of the EXISTING transport's consumed bytes, even
// when the decoder refuses them. It issues no additional request, redirects,
// proxy, tracer, retry, fallback or acceptance override. Failed/cut captures are
// diagnostics only, never evidence of an accepted observation or HTTP headers.
type liveCaptureTransport struct {
	base                *http.Transport
	dir                 string
	t                   *testing.T
	mu                  sync.Mutex
	sequence, remaining int
}

// These are local observations of the same request and transport call. They
// cannot identify a remote tracer failure when no response was received.
type liveRequestContextDiagnostic struct {
	DeadlineUTC               string `json:"deadline_utc,omitempty"`
	StateBefore               string `json:"state_before"`
	StateAfter                string `json:"state_after"`
	ElapsedNS                 int64  `json:"elapsed_ns"`
	TransportDeadlineExceeded bool   `json:"transport_deadline_exceeded"`
	TransportCanceled         bool   `json:"transport_canceled"`
	TransportTimeout          bool   `json:"transport_timeout"`
}

func liveCaptureContextErrorState(err error) string {
	switch {
	case err == nil:
		return "none"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "other"
	}
}

func (c *liveCaptureTransport) CloseIdleConnections() { c.base.CloseIdleConnections() }

func (c *liveCaptureTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	c.mu.Lock()
	c.sequence++
	number := c.sequence
	c.mu.Unlock()
	contextDiagnostic := liveRequestContextDiagnostic{
		StateBefore: liveCaptureContextErrorState(request.Context().Err()),
	}
	if deadline, ok := request.Context().Deadline(); ok {
		contextDiagnostic.DeadlineUTC = deadline.UTC().Format(time.RFC3339Nano)
	}
	started := time.Now()
	response, err := c.base.RoundTrip(request)
	contextDiagnostic.ElapsedNS = time.Since(started).Nanoseconds()
	contextDiagnostic.StateAfter = liveCaptureContextErrorState(request.Context().Err())
	contextDiagnostic.TransportDeadlineExceeded = errors.Is(err, context.DeadlineExceeded)
	contextDiagnostic.TransportCanceled = errors.Is(err, context.Canceled)
	var timeout net.Error
	contextDiagnostic.TransportTimeout = errors.As(err, &timeout) && timeout.Timeout()
	if number > 256 {
		return response, err
	}
	meta := map[string]any{"url": request.URL.String(), "diagnostic_only": true, "sequence": number, "capture_complete": false}
	meta["request_context_diagnostic"] = contextDiagnostic
	if request.GetBody != nil {
		body, e := request.GetBody()
		if e == nil {
			raw, readErr := io.ReadAll(io.LimitReader(body, 65537))
			_ = body.Close()
			if readErr == nil && len(raw) <= 65536 {
				meta["request_bytes"] = raw
			}
		}
	}
	stem := fmt.Sprintf("diagnostic-%03d", number)
	save := func() {
		b, e := json.MarshalIndent(meta, "", "  ")
		if e != nil {
			c.t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(c.dir, stem+".json"), append(b, '\n'), 0644); e != nil {
			c.t.Error(e)
		}
	}
	if err != nil {
		meta["transport_error"] = err.Error()
		save()
		return response, err
	}
	meta["accepted_http_status_not_asserted"] = response.StatusCode
	meta["retry_after_header_diagnostic"] = liveCaptureRetryAfter(response.Header)
	response.Body = &liveCaptureBody{ReadCloser: response.Body, owner: c, meta: meta, save: save, name: stem + ".bin"}
	return response, nil
}

type liveCaptureBody struct {
	io.ReadCloser
	owner          *liveCaptureTransport
	meta           map[string]any
	save           func()
	name           string
	mu             sync.Mutex
	once           sync.Once
	data           []byte
	truncated, eof bool
}

func (b *liveCaptureBody) Read(p []byte) (int, error) {
	n, e := b.ReadCloser.Read(p)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.owner.mu.Lock()
	take := min(n, 8<<20-len(b.data), b.owner.remaining)
	b.owner.remaining -= take
	b.owner.mu.Unlock()
	b.data = append(b.data, p[:take]...)
	b.truncated = b.truncated || take != n
	b.eof = b.eof || e == io.EOF
	return n, e
}

func (b *liveCaptureBody) Close() error {
	err := b.ReadCloser.Close()
	b.once.Do(func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.meta["response_file"] = b.name
		b.meta["response_bytes"] = len(b.data)
		b.meta["capture_complete"] = b.eof && !b.truncated
		b.meta["truncated"] = b.truncated
		if e := os.WriteFile(filepath.Join(b.owner.dir, b.name), b.data, 0644); e != nil {
			b.owner.t.Error(e)
		}
		b.save()
	})
	return err
}
