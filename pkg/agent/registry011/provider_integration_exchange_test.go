package registry011

import (
	"crypto/tls"
	"testing"
)

// This is finite inert parser data for refused-completion units only. It is
// never obtained through a transport, retained as a completed Exchange or used
// to qualify a provider, clock, policy or protected-message permission.
func providerInertCompletionInput(c *providerIntegration) *ProviderTransportSample {
	return &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: c.first.method, HTTPVersion: "HTTP/1.1", ServerName: "rpc.sepolia.ethpandaops.io", Complete: true, RequestHead: []byte(c.first.head), RequestBody: []byte(c.first.body), ResponseBody: []byte(`{"jsonrpc":"2.0","id":1,"result":"0x0"}`), ResponseWire: []byte("finite inert response"), WireBytes: len(c.first.wire) + 21, VerifiedChains: [][]string{{admissionTestDigest("a")}}, TLSVersion: tls.VersionTLS13, CipherSuite: tls.TLS_AES_128_GCM_SHA256}
}

func TestProviderRPCIncompleteExecutionNeverPublishesOrRefunds(t *testing.T) {
	for _, mode := range []string{"missing-sample", "failed-execution", "invalid-exchange", "clock-unavailable", "copy-budget", "wrong-meter-ticket", "missing-attribution", "retention-cap"} {
		t.Run(mode, func(t *testing.T) {
			c, s := providerPendingIntegrationUnit(t)
			permit, err := c.consume(c.firstTicket, c.first)
			if err != nil || !c.use(permit) {
				t.Fatal("normal retained consumption", err)
			}
			var sample *ProviderTransportSample
			var executionErr error
			switch mode {
			case "failed-execution":
				executionErr = errProviderIntegration
			case "invalid-exchange":
				sample = &ProviderTransportSample{Complete: true}
			case "clock-unavailable":
				c.clock.maximum = c.clock.read
			case "copy-budget":
				sample = providerInertCompletionInput(c)
				permit.meter.upper[10] = permit.meter.actual[10] + 1
			case "wrong-meter-ticket":
				sample = providerInertCompletionInput(c)
				permit.meter.ticket = admissionTestDigest("f")
			case "missing-attribution":
				s.activeMeter.Store(nil)
				executionErr = errProviderIntegration
			case "retention-cap":
				s.config.bounds.maximumCommits = int(s.sequence)
				executionErr = errProviderIntegration
			}
			sequence := s.sequence
			response, result, err := c.finishRPC(permit, sample, executionErr)
			if err == nil || response != nil || result != nil || !c.failed || c.firstExchange != "" || c.bootstrap != "" || c.root.checkpoint != "" {
				t.Fatal("incomplete execution produced an Exchange, start or output")
			}
			// The independent retained view, rather than a provisional local
			// ledger after a failed commit, is the durable debit being asserted.
			totals := s.replay.ledger.current.totals.bootstrap
			terminal := mode == "missing-sample" || mode == "failed-execution" || mode == "invalid-exchange"
			if terminal {
				if s.sequence != sequence+1 || totals.charged != c.firstTicket.work || totals.outstanding != (candidateWorkVector{}) {
					t.Fatal("terminal failure did not retain the complete original charge")
				}
			} else if s.sequence != sequence || totals.outstanding != c.firstTicket.work || totals.charged != (candidateWorkVector{}) {
				t.Fatal("unknown completion fabricated a terminal record or refunded work")
			}
		})
	}
}
