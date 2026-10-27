package registry011

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// SourceReadLimits bounds an entire batch acquisition, including initialization,
// ancestry, each complete block and all final rechecks. Limits are explicit test
// or application inputs, not approved operating defaults. The ceilings are
// 65536 attempted requests, 256 MiB of response JSON and five minutes. Existing
// transport and per-block Node limits also apply; no budget resets or retries
// are used to complete a refused batch.
type SourceReadLimits struct {
	MaxRequests, MaxWireBytes int
	AcquisitionTimeout        time.Duration
	Node                      NodeLimits
}

// A private clone shares only the already-owned HTTPS client. This scope is
// never returned to the Agent; nested scopes and caller-supplied transports are
// not a production acquisition option. Serial admission bounds concurrent
// internal calls too, before the response is streamed or retained.
type rpcAcquisition struct {
	mu                  sync.Mutex
	remaining, requests int
	maximumRequests     int
	deadline            time.Time
	context             context.Context
	exchanges           []NodeExchange
	// Non-nil only for the private connected Agent adapter. The legacy scope
	// constructor never installs, strips or replaces this mandatory gate.
	integration *providerIntegration
}

func newSourceAcquisition(ctx context.Context, client *ReadRPC, limits SourceReadLimits) (*ReadRPC, context.Context, context.CancelFunc, error) {
	if ctx == nil || client == nil || client.client == nil || client.acquisition != nil || limits.MaxRequests < 1 || limits.MaxRequests > 65536 || limits.MaxWireBytes < 1 || limits.MaxWireBytes > 256<<20 || limits.AcquisitionTimeout <= 0 || limits.AcquisitionTimeout > 5*time.Minute {
		return nil, nil, nil, ErrConfiguration
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	child, cancel := context.WithTimeout(ctx, limits.AcquisitionTimeout)
	deadline, _ := child.Deadline()
	clone := *client
	clone.acquisition = &rpcAcquisition{remaining: limits.MaxWireBytes, maximumRequests: limits.MaxRequests, deadline: deadline, context: child}
	return &clone, child, cancel, nil
}

func (a *rpcAcquisition) admit(ctx context.Context, maximum int) (context.Context, int, func(), error) {
	a.mu.Lock()
	if err := a.context.Err(); err != nil {
		a.mu.Unlock()
		return nil, 0, nil, err
	}
	if err := ctx.Err(); err != nil {
		a.mu.Unlock()
		return nil, 0, nil, err
	}
	if a.requests >= a.maximumRequests || a.remaining <= 0 || !time.Now().Before(a.deadline) {
		a.mu.Unlock()
		return nil, 0, nil, ErrReplayUnavailable
	}
	a.requests++ // Failed requests consume the same aggregate admission budget.
	child, cancel := context.WithDeadline(ctx, a.deadline)
	stop := context.AfterFunc(a.context, cancel)
	return child, min(maximum, a.remaining), func() { stop(); cancel(); a.mu.Unlock() }, nil
}

// Called only while holding the admission lock. Failed response bytes consume
// the budget too; over-limit reads exhaust it and never create an exchange.
func (a *rpcAcquisition) consume(size int) {
	a.remaining = max(0, a.remaining-size)
}

func (a *rpcAcquisition) retain(method string, params any, raw json.RawMessage) error {
	encoded, err := json.Marshal(params)
	if err != nil {
		return ErrConfiguration
	}
	// Both buffers belong exclusively to this private acquisition. Public output
	// never exposes its client or internal admission scope for subsequent writes.
	a.exchanges = append(a.exchanges, NodeExchange{method, encoded, raw})
	return nil
}
