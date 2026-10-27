package registry011

import (
	"context"
	"testing"
)

// Positive normal owned TLS/file runtime only, with closed protocol responses.
// No malformed peer, alternate operating endpoint or transport callback exists.
func providerOwnedPinChildrenUnit(t *testing.T) (*providerIntegration, *providerIntegrationPermit, string, [4]string) {
	t.Helper()
	c, _ := providerRetainedFirstExchangeFixture(t)
	if err := c.installBootstrap(); err != nil {
		t.Fatal(err)
	}
	raw := providerPinInput(c.bootstrap, "finite-owned-pin")
	input, err := newProviderIntegrationObject("pin-input", raw, c.store.config.bounds.maximumObjectBytes)
	if err != nil {
		t.Fatal(err)
	}
	parent, err := c.beginLocalObjects(input.digest, []providerIntegrationObjectInput{{"pin-input", raw}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.restoreMeter(parent.meter) })
	requests, err := providerPinRequests(c.selected.Hash)
	if err != nil {
		t.Fatal(err)
	}
	results := providerPinResultFixture(t)
	var digests [4]string
	for i, request := range requests {
		ticket, err := c.admitRPC(request, false)
		if err != nil {
			t.Fatal("child admission", i, err)
		}
		permit, err := c.consume(ticket, request)
		if err != nil || !c.use(permit) {
			t.Fatal("child one-use consumption", i, err)
		}
		response := `{"jsonrpc":"2.0","id":1,"result":` + string(results[i]) + `}`
		conn := providerIntegrationOwnedTLS(t, request.body, response)
		ctx, cancel := context.WithTimeout(context.Background(), c.primitive.capture.Timeout)
		sample, captureErr := providerIntegrationCapture(ctx, conn, c.primitive.capture, request, permit.meter)
		cancel()
		_ = conn.Close()
		_, _, err = c.finishRPC(permit, sample, captureErr)
		c.restoreMeter(permit.meter)
		if err != nil {
			t.Fatal("normal owned child result", i, err)
		}
		digests[i] = c.lastExchange
	}
	return c, parent, input.digest, digests
}

func TestProviderIntegrationOwnedPinClosureRetainsEveryChildAndParent(t *testing.T) {
	for _, mode := range []string{"normal", "missing-child", "unfunded-header", "unused-parent"} {
		t.Run(mode, func(t *testing.T) {
			c, parent, input, digests := providerOwnedPinChildrenUnit(t)
			before := c.ledger.current.counts
			switch mode {
			case "missing-child":
				digests[0] = admissionTestDigest("f")
			case "unfunded-header":
				parent.meter.upper[3] = 0
			case "unused-parent":
				parent.used = false // Inert local finalization refusal; no dispatch.
			}
			pin, err := c.finishPinResult(parent, input, digests)
			if mode == "normal" {
				if err != nil || !lowerHash(pin) || c.ledger.current.counts != [3]uint64{7, 7, 7} || c.store.sequence != 22 || c.root.status != "enrolling" || c.root.checkpoint != "" {
					t.Fatal("normal owned closure skipped or invented work", err)
				}
			} else if err == nil || pin != "" || c.ledger.current.counts != before {
				t.Fatal("incomplete local result closed or refunded the parent", mode)
			}
			if c.pin != "" {
				t.Fatal("local result selected or activated an operating pin")
			}
		})
	}
}

func TestProviderIntegrationPinPreparationRefusesBeforeChildWork(t *testing.T) {
	for _, mode := range []string{"object", "preparation"} {
		t.Run(mode, func(t *testing.T) {
			c, s := providerRetainedFirstExchangeFixture(t)
			if err := c.installBootstrap(); err != nil {
				t.Fatal(err)
			}
			before := c.ledger.current.counts
			if mode == "object" {
				s.config.bounds.maximumObjectBytes = 1
			} else {
				c.preparations = s.config.bounds.maximumCommits
			}
			if reads, err := c.pinAgent(context.Background(), "finite-owned-pin"); err == nil || reads != nil || c.ledger.current.counts != before || c.pin != "" {
				t.Fatal("unfunded pin preparation dispatched or installed a result", mode)
			}
		})
	}
	if providerCheckPinPart(-1, nil) {
		t.Fatal("unknown pin part accepted")
	}
}
