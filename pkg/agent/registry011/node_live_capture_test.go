//go:build registry_live

package registry011

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
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

func (c *liveCaptureTransport) CloseIdleConnections() { c.base.CloseIdleConnections() }

func (c *liveCaptureTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	c.mu.Lock()
	c.sequence++
	number := c.sequence
	c.mu.Unlock()
	response, err := c.base.RoundTrip(request)
	if number > 256 {
		return response, err
	}
	meta := map[string]any{"url": request.URL.String(), "diagnostic_only": true, "sequence": number, "capture_complete": false}
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
