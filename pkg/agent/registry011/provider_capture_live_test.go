//go:build provider_capture_live

package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// One ordinary read to the selected official RPC. No policy or Source is
// selected, no key is accessed, and no transaction or retry can be submitted.
func TestProviderCaptureOfficialRPC(t *testing.T) {
	dir := os.Getenv("SAGE_PROVIDER_CAPTURE_DIR")
	if dir == "" {
		t.Fatal("explicit evidence directory required")
	}
	if e := os.MkdirAll(dir, 0755); e != nil {
		t.Fatal(e)
	}
	sample, captureErr := CaptureSepoliaProviderFinality(context.Background(), ProviderCaptureLimits{20 * time.Second, 64 << 10, 4 << 20, 8 << 20, 128})
	if sample != nil {
		for name, raw := range map[string][]byte{"request-head.bin": sample.RequestHead, "request-body.json": sample.RequestBody, "response-head.bin": sample.ResponseHead, "response-body.json": sample.ResponseBody, "response-wire.bin": sample.ResponseWire} {
			if e := os.WriteFile(filepath.Join(dir, name), raw, 0644); e != nil {
				t.Fatal(e)
			}
		}
	}
	report := map[string]any{"scope": "one data-only raw HTTP/1.1 and TLS-origin feasibility sample; not ExchangeV1 or provider execution/history/clock/store qualification", "source_ready": false, "policy_activated": false, "paid_transactions": 0, "sample": sample}
	if captureErr != nil {
		report["capture_error"] = captureErr.Error()
	}
	data, e := json.MarshalIndent(report, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(dir, "capture.json"), append(data, '\n'), 0644); e != nil {
		t.Fatal(e)
	}
	if captureErr != nil {
		t.Fatal(captureErr)
	}
	if sample == nil || !sample.Complete {
		t.Fatal("complete data-only sample required")
	}
}
