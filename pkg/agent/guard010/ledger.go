package guard010

import (
	"context"
	"encoding/hex"
	"sync"

	"github.com/sage-x-project/sage/pkg/agent/execution010"
)

// GuardLedger owns its storage handle. All public reservations freshly verify
// an envelope; callers cannot supply Entry projections or stale verification flags.
// This is reservation storage, not a dispatch or policy-retirement gate.
type GuardLedger struct {
	mu        sync.Mutex
	store     *execution010.Ledger
	recipient string
}

// Reservation is an authenticated storage observation, never execution authority
// or authenticated tool output. An existing record never counts as a new grant.
type Reservation struct {
	created bool
	state   string
	digest  string
}

func (r *Reservation) Created() bool { return r != nil && r.created }
func (r *Reservation) State() string {
	if r == nil {
		return ""
	}
	return r.state
}
func (r *Reservation) IntentDigest() string {
	if r == nil {
		return ""
	}
	return r.digest
}

// OpenLedger initializes only a new isolated scope when create is explicit.
// Normal recovery must use create=false; lost state never becomes an empty ledger.
// Paths and administrative initialization belong to the trusted host.
func OpenLedger(path string, create bool, recipient string) (*GuardLedger, error) {
	if !did(recipient) {
		return nil, ErrInvalid
	}
	store, err := execution010.Open(path, create)
	if err != nil {
		return nil, ErrInvalid
	}
	return &GuardLedger{store: store, recipient: recipient}, nil
}

// Reserve authenticates every attempt, including reads of existing calls. It
// derives all storage identity fields from a privately verified intent and
// atomically reserves call and nonce or returns the unchanged state. It exposes
// no terminal bytes: signed response production and client consumption are separate.
// Callbacks must honor bounded deadlines. If time expires during durable storage,
// denial preserves any committed reservation and never authorizes an effect.
func (l *GuardLedger) Reserve(ctx context.Context, raw []byte, a Authority, p IntentPolicy) (*Reservation, error) {
	if l == nil || ctx == nil {
		return nil, ErrInvalid
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.store == nil {
		return nil, ErrInvalid
	}
	verified, err := VerifyIntent(ctx, raw, l.recipient, a, p)
	if err != nil {
		return nil, ErrInvalid
	}
	entry, err := reservationEntry(verified)
	if err != nil {
		return nil, ErrInvalid
	}
	if ctx.Err() != nil {
		return nil, ErrInvalid
	}
	stored, created, err := l.store.Reserve(entry)
	if err != nil {
		return nil, ErrInvalid
	}
	_, intent, _, err := intentEnvelope(verified.canonical)
	if err != nil {
		return nil, ErrInvalid
	}
	now, err := a.Now(ctx)
	if err != nil || !times(intent, now) || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	return &Reservation{created: created, state: stored.State, digest: verified.Digest()}, nil
}
func reservationEntry(v *VerifiedIntent) (execution010.Entry, error) {
	if v == nil || len(v.canonical) == 0 {
		return execution010.Entry{}, ErrInvalid
	}
	_, m, _, err := intentEnvelope(v.canonical)
	if err != nil {
		return execution010.Entry{}, ErrInvalid
	}
	expires, ok := number(m, "expires")
	if !ok {
		return execution010.Entry{}, ErrInvalid
	}
	return execution010.Entry{Issuer: str(m, "issuer"), Recipient: str(m, "recipient"), CallID: str(m, "call_id"), Nonce: str(m, "nonce"), Expires: expires, IntentHex: hex.EncodeToString(v.canonical), State: "RESERVED"}, nil
}

// Close releases a healthy single-writer handle. A poisoned store retains its
// lock for protected administrative recovery, following execution010 semantics.
func (l *GuardLedger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.store == nil {
		return nil
	}
	err := l.store.Close()
	l.store = nil
	return err
}
