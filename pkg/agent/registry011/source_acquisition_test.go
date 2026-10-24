package registry011

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func acquisitionTestLimits() SourceReadLimits {
	return SourceReadLimits{MaxRequests: 16, MaxWireBytes: 1 << 20, AcquisitionTimeout: time.Second, Node: nodeTestLimits()}
}

// All responses are finite owned data. No endpoint or malformed service is
// started for refusal cases; the original HTTPS transport is never redirected.
func TestSourceAggregateIncludesPinningAndAllLaterReads(t *testing.T) {
	base, fixture := fixtureRPC(t, "")
	limits := acquisitionTestLimits()
	limits.MaxRequests = 4
	client, ctx, cancel, err := newSourceAcquisition(context.Background(), base, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	reads, err := client.PinAgentReads(ctx, SepoliaRegistryAddress, "owned-agent")
	if err != nil || len(fixture.calls) != 4 || len(client.acquisition.exchanges) != 4 {
		t.Fatal("pinning was not included", err)
	}
	if _, err := reads.CodeBytes(ctx, testRegistry); !errors.Is(err, ErrReplayUnavailable) || len(fixture.calls) != 4 {
		t.Fatal("a later API reset the shared request budget", err)
	}
	if base.acquisition != nil || base.client != client.client {
		t.Fatal("original client mutated or transport replaced")
	}
	// The caller's ordinary reader is unchanged by a refused private scope.
	if _, err := base.PinAgentReads(context.Background(), SepoliaRegistryAddress, "owned-agent"); err != nil || len(fixture.calls) != 8 {
		t.Fatal("scope escaped into the original reader", err)
	}
	for _, exchange := range client.acquisition.exchanges {
		if !json.Valid(exchange.Params) || !json.Valid(exchange.Response) {
			t.Fatal("original evidence missing")
		}
	}
}

func TestSourceAggregateBytesAndFailedAttemptsRemainCharged(t *testing.T) {
	const body = `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`
	for _, mode := range []string{"exact", "stream-exceeds", "decode-fails", "request-fails"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			base := testReadRPC(t, func(*http.Request) (*http.Response, error) {
				calls++
				if mode == "request-fails" {
					return nil, errors.New("owned unavailable result")
				}
				if mode == "decode-fails" {
					return jsonResponse(strings.Repeat(" ", len(body))), nil
				}
				return jsonResponse(body), nil
			})
			limits := acquisitionTestLimits()
			limits.MaxRequests, limits.MaxWireBytes = 1, len(body)
			if mode == "stream-exceeds" {
				limits.MaxWireBytes--
			}
			client, ctx, cancel, err := newSourceAcquisition(context.Background(), base, limits)
			if err != nil {
				t.Fatal(err)
			}
			defer cancel()
			got, err := client.call(ctx, "eth_chainId", []any{})
			if mode == "exact" {
				if err != nil || string(got) != `"0xaa36a7"` || len(client.acquisition.exchanges) != 1 {
					t.Fatal("exact boundary", err)
				}
			} else if err == nil || got != nil || len(client.acquisition.exchanges) != 0 {
				t.Fatal("failed exchange was accepted", err)
			}
			if client.acquisition.requests != 1 || mode != "request-fails" && client.acquisition.remaining != 0 {
				t.Fatal("failed work was not charged")
			}
			if _, err := client.call(ctx, "eth_chainId", []any{}); !errors.Is(err, ErrReplayUnavailable) || calls != 1 {
				t.Fatal("failure permitted another attempt", err)
			}
		})
	}
}

func TestSourceScopeCancellationReachesEveryInternalContext(t *testing.T) {
	entered := make(chan struct{})
	base := testReadRPC(t, func(req *http.Request) (*http.Response, error) {
		close(entered)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	client, _, cancel, err := newSourceAcquisition(context.Background(), base, acquisitionTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := client.call(context.Background(), "eth_chainId", []any{})
		done <- err
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("normal owned request never started")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("scope cancellation ignored")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled internal request did not finish")
	}
	if _, err := client.call(context.Background(), "eth_chainId", []any{}); !errors.Is(err, context.Canceled) || client.acquisition.requests != 1 {
		t.Fatal("cancelled scope admitted another request", err)
	}
}

func TestSourceScopeRefusesNestedAndInvalidConfiguration(t *testing.T) {
	base := testReadRPC(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid scope performed a request")
		return nil, nil
	})
	for name, edit := range map[string]func(*SourceReadLimits){
		"no-requests":       func(l *SourceReadLimits) { l.MaxRequests = 0 },
		"too-many-requests": func(l *SourceReadLimits) { l.MaxRequests = 65537 },
		"no-bytes":          func(l *SourceReadLimits) { l.MaxWireBytes = 0 },
		"too-many-bytes":    func(l *SourceReadLimits) { l.MaxWireBytes = (256 << 20) + 1 },
		"no-time":           func(l *SourceReadLimits) { l.AcquisitionTimeout = 0 },
		"too-much-time":     func(l *SourceReadLimits) { l.AcquisitionTimeout = 5*time.Minute + 1 },
	} {
		t.Run(name, func(t *testing.T) {
			limits := acquisitionTestLimits()
			edit(&limits)
			if c, _, cancel, err := newSourceAcquisition(context.Background(), base, limits); c != nil || cancel != nil || !errors.Is(err, ErrConfiguration) {
				t.Fatal("invalid limits admitted", err)
			}
		})
	}
	for _, c := range []*ReadRPC{nil, {}} {
		if got, _, _, err := newSourceAcquisition(context.Background(), c, acquisitionTestLimits()); got != nil || !errors.Is(err, ErrConfiguration) {
			t.Fatal(err)
		}
	}
	//nolint:staticcheck // Verify refusal of an explicitly missing context.
	if got, _, _, err := newSourceAcquisition(nil, base, acquisitionTestLimits()); got != nil || !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, _, _, err := newSourceAcquisition(ctx, base, acquisitionTestLimits()); got != nil || !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	client, ctx, finish, err := newSourceAcquisition(context.Background(), base, acquisitionTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer finish()
	if got, _, _, err := newSourceAcquisition(ctx, client, acquisitionTestLimits()); got != nil || !errors.Is(err, ErrConfiguration) {
		t.Fatal("nested scope reset", err)
	}
	// A past absolute deadline is an inert state scenario, without a sleep.
	client.acquisition.deadline = time.Now().Add(-time.Second)
	if _, err := client.call(ctx, "eth_chainId", []any{}); !errors.Is(err, ErrReplayUnavailable) {
		t.Fatal("expired scope admitted", err)
	}
}

func TestSourceAdmittedEncodingFailureAndCallerCancellation(t *testing.T) {
	base := testReadRPC(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("refused local input contacted the node")
		return nil, nil
	})
	limits := acquisitionTestLimits()
	limits.MaxRequests = 1
	client, ctx, finish, err := newSourceAcquisition(context.Background(), base, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer finish()
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := client.call(cancelled, "eth_chainId", []any{}); !errors.Is(err, context.Canceled) || client.acquisition.requests != 0 {
		t.Fatal("cancelled caller consumed network work", err)
	}
	if _, err := client.call(ctx, "eth_chainId", make(chan struct{})); err == nil || client.acquisition.requests != 1 {
		t.Fatal("admitted input failure reset the budget", err)
	}
	if _, err := client.call(ctx, "eth_chainId", []any{}); !errors.Is(err, ErrReplayUnavailable) {
		t.Fatal("admitted failure permitted retry", err)
	}
}
