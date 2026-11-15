package registry011

import (
	"context"
	"testing"
	"time"
)

// Only the existing normal owned TLS fixture is used. Neither scenario sends
// an HTTP request or constructs an invalid peer. This arithmetic BYTE subject
// owns no original T/C/attempt/order and cannot authorize a retained result.
func TestProviderHistoryCaptureRejectsUnfundedTLSAndClosedOwnedTransportBeforeWire(t *testing.T) {
	for _, mode := range []string{"certificate-bound", "closed-before-deadline"} {
		t.Run(mode, func(t *testing.T) {
			request := historyGroupSimple(t)
			_, limits := historyGroupBounds()
			certificates := providerTLSBounds{2, 4, 8 << 10}
			conn := providerIntegrationOwnedTLS(t, request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
			state := conn.ConnectionState()
			if !state.HandshakeComplete || len(state.VerifiedChains) == 0 || len(state.VerifiedChains[0]) == 0 || len(state.VerifiedChains[0][0].Raw) <= 1 {
				t.Fatal("ordinary verified owned TLS identity required before the isolated refusal")
			}
			switch mode {
			case "certificate-bound":
				// This standalone UNIT selection precedes its first observer work.
				// The real verified certificate cannot fit its admitted one-byte bound.
				certificates.certificateBytes = 1
			case "closed-before-deadline":
				if err := conn.Close(); err != nil {
					t.Fatal("close only the owned fixture connection", err)
				}
				if err := conn.SetDeadline(time.Time{}); err == nil {
					t.Fatal("owned connection did not close before capture")
				}
			}
			m := historyCaptureUnitMeter(limits)
			upper, valid := providerHistoryCaptureUpper(uint64(len(request.wire)), limits, certificates)
			if !valid || !m.preflightWork(upper) || m.actual != (candidateWorkVector{}) {
				t.Fatal("positive complete BYTE allowance required before the expected refusal")
			}
			ctx, cancel := context.WithTimeout(context.Background(), limits.Timeout)
			defer cancel()
			data, err := providerHistoryCaptureBytes(ctx, conn, limits, request, m, certificates)
			if err == nil || data != nil || m.actual[0] != 0 || m.actual[1] != 0 || m.actual[2] != 0 || m.actual[10] == 0 {
				t.Fatal("unfunded identity or closed transport supplied exchange DATA or observed wire bytes", mode, err)
			}
			if mode == "closed-before-deadline" && !m.failed {
				t.Fatal("failed deadline installation preserved a reusable BYTE observer")
			}
		})
	}
}
