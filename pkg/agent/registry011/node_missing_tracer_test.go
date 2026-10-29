package registry011

import (
	"encoding/json"
	"testing"
)

// Finite response DATA from the selected RPC's unavailable native tracer.
// This never evaluates a tracer, sends a request or tries another adapter.
func TestGasProfileRefusesUnavailableNativeTracerResponse(t *testing.T) {
	for _, raw := range []string{
		`[{"txHash":"0xf71902487e666d730444c528709909d4d4e0d320f9b74ebcdc2d0d3f7691d81b","error":"ReferenceError: stateGasTracer is not defined at <eval>:1:2(0)"}]`,
		`[{"txHash":"0xf71902487e666d730444c528709909d4d4e0d320f9b74ebcdc2d0d3f7691d81b","result":null,"error":"unsupported native tracer"}]`,
	} {
		if profile, err := replayGasProfile(json.RawMessage(raw), 1); err == nil || profile != "" {
			t.Fatal("unavailable native tracer became a strict gas profile")
		}
	}
}
