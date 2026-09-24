package guard010

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// ClientClock provides trusted UTC and monotonic milliseconds, never peer time.
// Both values must be nonnegative safe integers and must not move backwards.
type ClientClock interface {
	Sample(context.Context) (utcMillis, monotonicMillis int64, err error)
}

// ClientSender performs one bounded protected transport handoff under the client lock.
// It must bind this ID and exact intent to one request, provide fresh outer security
// fields, honor expiry, and never queue later duplicate transmissions or reenter.
// An error is unverified and may mean the request was transmitted.
type ClientSender interface {
	Commit(context.Context, string, []byte) error
}

// ClientServices are trusted, bounded, non-reentrant host providers. Intent
// authority and policy are checked afresh for each attempted transmission.
type ClientServices struct {
	IntentAuthority Authority
	Policy          IntentPolicy
	ResultAuthority Authority
	Clock           ClientClock
	Sender          ClientSender
	// Expected identities come from trusted Client configuration, not the intent.
	ExpectedIssuer    string
	ExpectedRecipient string
}

// HopParent reports the durable authorized admission state of an incoming call.
// Missing, rejected, and unknown states must fail. The exact authenticated
// envelope is supplied so a parent identifier alone cannot grant authority.
type HopParent interface {
	Authorized(context.Context, []byte) error
}

// HopServices are trusted providers for a captured A-to-B input. They are
// separate from B's authority and policy for the outgoing B-to-C call.
type HopServices struct {
	Authority Authority
	Policy    IntentPolicy
	Parent    HopParent
}

type hopBinding struct {
	incoming []byte
	services HopServices
}

// ClientInvocation binds one outstanding transport request to a single Client.
// Keep it protected from plugins. ID must also be bound to the actual outer request.
type ClientInvocation struct {
	owner     *Client
	id        string
	canonical []byte
}

func (i *ClientInvocation) ID() string {
	if i == nil {
		return ""
	}
	return i.id
}
func (i *ClientInvocation) Intent() []byte {
	if i == nil {
		return nil
	}
	return append([]byte(nil), i.canonical...)
}

// ClientDelivery exposes output only on the first durably accepted completed
// terminal. Pending and ignored replies never carry consumable output.
type ClientDelivery struct {
	status         string
	first, ignored bool
	output         []byte
}

func (d *ClientDelivery) Status() string      { return d.status }
func (d *ClientDelivery) FirstTerminal() bool { return d.first }
func (d *ClientDelivery) Ignored() bool       { return d.ignored }
func (d *ClientDelivery) Output() []byte      { return append([]byte(nil), d.output...) }

type clientEvent struct {
	Kind      string `json:"kind"`
	ID        string `json:"id"`
	At        int64  `json:"at"`
	IntentHex string `json:"intent_hex"`
	ResultHex string `json:"result_hex"`
}

const clientHeader = "sage-guard-client|0.10.0\n"
const clientMaxSize = 8 << 20

// Client owns one operation's protected durable journal. The host must maintain
// one stable journal per issuer/call and must never initialize a second journal
// for the same operation. Normal reopen requires existing state. A crash or failed
// write retains the exclusive lock for protected administration. Rollback protection
// and exactly-once downstream effects are external; persistence precedes delivery.
type Client struct {
	mu                                                              sync.Mutex
	file                                                            *os.File
	lock                                                            string
	services                                                        ClientServices
	hop                                                             *hopBinding
	intent, terminal                                                []byte
	seen                                                            map[string]bool
	last, lastMono, lastWall, observedUTC, observedMono, openedMono int64
	rows                                                            int
	size                                                            int64
	failed                                                          bool
}

func checkHop(ctx context.Context, incoming, outgoing []byte, s ClientServices, h HopServices) error {
	if ctx == nil || h.Authority == nil || h.Policy == nil || h.Parent == nil {
		return ErrInvalid
	}
	_, parent, parentCanonical, err := intentEnvelope(incoming)
	if err != nil || !bytes.Equal(incoming, parentCanonical) || str(parent, "recipient") != s.ExpectedIssuer {
		return ErrInvalid
	}
	if _, err = VerifyIntent(ctx, incoming, s.ExpectedIssuer, h.Authority, h.Policy); err != nil {
		return ErrInvalid
	}
	if h.Parent.Authorized(ctx, append([]byte(nil), incoming...)) != nil || ctx.Err() != nil {
		return ErrInvalid
	}
	_, child, _, err := intentEnvelope(outgoing)
	if err != nil || str(child, "issuer") != s.ExpectedIssuer || str(child, "recipient") != s.ExpectedRecipient ||
		str(child, "request_id") == str(parent, "request_id") || str(child, "call_id") == str(parent, "call_id") ||
		(child["parent_call_id"] != nil && str(child, "parent_call_id") != str(parent, "call_id")) {
		return ErrInvalid
	}
	digest, err := OriginalCommitment([][]byte{incoming})
	if err != nil || digest != str(child, "original_digest") {
		return ErrInvalid
	}
	return nil
}

// OpenHopClient binds an authenticated, authorized A-to-B call to B's fresh
// capture and independently authorized B-to-C operation. The host must route
// every multi-hop protected effect through this entry point, retain the parent
// admission in protected storage, and use it again when reopening the journal.
func OpenHopClient(ctx context.Context, path string, create bool, incoming, outgoing []byte, s ClientServices, h HopServices) (*Client, error) {
	if checkHop(ctx, incoming, outgoing, s, h) != nil {
		return nil, ErrInvalid
	}
	c, err := openClient(ctx, path, create, outgoing, s, true)
	if err != nil {
		return nil, err
	}
	c.hop = &hopBinding{incoming: append([]byte(nil), incoming...), services: h}
	return c, nil
}

func clientTerminal(raw, intent []byte) bool {
	env, b, e := object(raw)
	if e != nil || !bytes.Equal(raw, b) || !closed(env, "result proof") {
		return false
	}
	r, ok := env["result"].(map[string]any)
	if !ok || !closed(r, "version request_id call_id issuer recipient created expires keyid alg intent_digest status output") || !common(r) {
		return false
	}
	_, i, _, e := intentEnvelope(intent)
	if e != nil {
		return false
	}
	if _, ok = b64(str(env, "proof"), 64); !ok {
		return false
	}
	out, ok := r["output"].(map[string]any)
	if !ok {
		return false
	}
	status := str(r, "status")
	if status != "completed" && status != "rejected" && status != "unknown" {
		return false
	}
	if status != "completed" && len(out) != 0 {
		return false
	}
	return str(r, "intent_digest") == hash(intent) && str(r, "request_id") == str(i, "request_id") && str(r, "call_id") == str(i, "call_id") && str(r, "issuer") == str(i, "recipient") && str(r, "recipient") == str(i, "issuer")
}
func (c *Client) apply(e clientEvent) error {
	switch e.Kind {
	case "open":
		if c.rows != 0 || e.ID != "" || e.At != 0 || e.ResultHex != "" || e.IntentHex != hex.EncodeToString(c.intent) {
			return ErrInvalid
		}
	case "send":
		if c.rows == 0 || len(c.terminal) != 0 || !uuid.MatchString(e.ID) || e.At < 0 || e.At > 9007199254740991 || e.IntentHex != "" || e.ResultHex != "" {
			return ErrInvalid
		}
		if _, exists := c.seen[e.ID]; exists || (c.last >= 0 && e.At-c.last < 1000) {
			return ErrInvalid
		}
		c.seen[e.ID] = true
		c.last = e.At
	case "close":
		if !c.seen[e.ID] || e.At != 0 || e.IntentHex != "" || e.ResultHex != "" {
			return ErrInvalid
		}
		c.seen[e.ID] = false
	case "terminal":
		closed, exists := c.seen[e.ID]
		raw, err := hex.DecodeString(e.ResultHex)
		if !exists || closed || len(c.terminal) != 0 || e.At != 0 || e.IntentHex != "" || err != nil || hex.EncodeToString(raw) != e.ResultHex || !clientTerminal(raw, c.intent) {
			return ErrInvalid
		}
		c.terminal = raw
	default:
		return ErrInvalid
	}
	c.rows++
	return nil
}
func (c *Client) append(e clientEvent) error {
	b, err := json.Marshal(e)
	if err != nil {
		return ErrInvalid
	}
	b = append(b, '\n')
	if c.rows >= 1024 || c.size+int64(len(b)) > clientMaxSize {
		c.failed = true
		return ErrInvalid
	}
	n, err := c.file.Write(b)
	if err != nil || n != len(b) || c.file.Sync() != nil {
		c.failed = true
		return ErrInvalid
	}
	if c.apply(e) != nil {
		c.failed = true
		return ErrInvalid
	}
	c.size += int64(len(b))
	return nil
}
func (c *Client) sample(ctx context.Context) (int64, int64, error) {
	u, m, e := c.services.Clock.Sample(ctx)
	if e != nil || ctx.Err() != nil || u < 0 || m < 0 || u > 9007199254740991 || m > 9007199254740991 || u < c.observedUTC || m < c.observedMono || u < c.last {
		c.failed = true
		return 0, 0, ErrInvalid
	}
	c.observedUTC = u
	c.observedMono = m
	return u, m, nil
}

// OpenClient initializes only a root operation with no parent ID, or reopens its
// exact protected original. A declared parent requires OpenHopClient. Existing
// terminal outcomes are never delivered again on open.
// Outstanding pre-restart transports are abandoned; retries need new invocations.
func OpenClient(ctx context.Context, path string, create bool, raw []byte, s ClientServices) (*Client, error) {
	return openClient(ctx, path, create, raw, s, false)
}

func openClient(ctx context.Context, path string, create bool, raw []byte, s ClientServices, hop bool) (*Client, error) {
	if ctx == nil || s.IntentAuthority == nil || s.Policy == nil || s.ResultAuthority == nil || s.Clock == nil || s.Sender == nil || (runtime.GOOS != "linux" && runtime.GOOS != "darwin") {
		return nil, ErrInvalid
	}
	_, i, canonical, e := intentEnvelope(raw)
	if e != nil {
		return nil, ErrInvalid
	}
	if (!hop && i["parent_call_id"] != nil) || !did(s.ExpectedIssuer) || !did(s.ExpectedRecipient) ||
		str(i, "issuer") != s.ExpectedIssuer || str(i, "recipient") != s.ExpectedRecipient {
		return nil, ErrInvalid
	}
	if create {
		if _, e = VerifyIntent(ctx, canonical, str(i, "recipient"), s.IntentAuthority, s.Policy); e != nil {
			return nil, ErrInvalid
		}
	}
	lock := path + ".lock"
	// #nosec G304 -- protected per-operation host storage, not a wire path.
	guard, e := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return nil, ErrInvalid
	}
	if guard.Close() != nil {
		_ = os.Remove(lock)
		return nil, ErrInvalid
	}
	flags := os.O_RDWR | os.O_APPEND
	if create {
		flags |= os.O_CREATE | os.O_EXCL
	}
	// #nosec G304 -- normal reopen must not recreate a missing journal.
	file, e := os.OpenFile(path, flags, 0600)
	if e != nil {
		_ = os.Remove(lock)
		return nil, ErrInvalid
	}
	c := &Client{file: file, lock: lock, services: s, intent: canonical, seen: map[string]bool{}, last: -1, lastMono: -1, lastWall: -1, observedUTC: -1, observedMono: -1}
	fail := func() (*Client, error) { c.failed = true; _ = c.Close(); return nil, ErrInvalid }
	if create {
		if _, e = file.WriteString(clientHeader); e != nil || file.Sync() != nil {
			return fail()
		}
		// #nosec G304 -- sync the parent of trusted configured storage.
		dir, er := os.Open(filepath.Dir(path))
		if er != nil {
			return fail()
		}
		er = dir.Sync()
		ce := dir.Close()
		if er != nil || ce != nil {
			return fail()
		}
		c.size = int64(len(clientHeader))
		if c.append(clientEvent{Kind: "open", IntentHex: hex.EncodeToString(canonical)}) != nil {
			return fail()
		}
	} else {
		if _, e = file.Seek(0, io.SeekStart); e != nil {
			return fail()
		}
		b, er := io.ReadAll(io.LimitReader(file, clientMaxSize+1))
		if er != nil || len(b) > clientMaxSize || !bytes.HasPrefix(b, []byte(clientHeader)) || !bytes.HasSuffix(b, []byte("\n")) {
			return fail()
		}
		body := b[len(clientHeader):]
		if len(body) == 0 {
			return fail()
		}
		for _, line := range bytes.Split(body[:len(body)-1], []byte("\n")) {
			var event clientEvent
			if json.Unmarshal(line, &event) != nil {
				return fail()
			}
			normal, _ := json.Marshal(event)
			if !bytes.Equal(normal, line) || c.rows >= 1024 || c.apply(event) != nil {
				return fail()
			}
		}
		c.size = int64(len(b))
	}
	_, mono, er := c.sample(ctx)
	if er != nil {
		return fail()
	}
	c.openedMono = mono
	return c, nil
}

// Begin persists a fresh outer UUID, then performs one bounded protected handoff.
// Use a fresh outer nonce/session sequence as required by the transport. The core
// returns the unchanged original intent, never a new call identity. Polling needs
// both UTC and monotonic elapsed time >=1s since the previous handoff finished;
// reopening adds a conservative 1s wait. It never hands off after terminal acceptance.
func (c *Client) Begin(ctx context.Context, id string) (ticket *ClientInvocation, err error) {
	if c == nil || ctx == nil {
		return nil, ErrInvalid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	defer func() {
		if recover() != nil {
			c.failed = true
			ticket = nil
			err = ErrInvalid
		}
	}()
	if c.failed || c.file == nil || len(c.terminal) != 0 || !uuid.MatchString(id) {
		return nil, ErrInvalid
	}
	if _, ok := c.seen[id]; ok {
		return nil, ErrInvalid
	}
	u, m, e := c.sample(ctx)
	if e != nil {
		return nil, ErrInvalid
	}
	if c.last >= 0 && (u-c.last < 1000 || (c.lastWall >= 0 && u-c.lastWall < 1000) || (c.lastMono >= 0 && m-c.lastMono < 1000) || (c.lastMono < 0 && m-c.openedMono < 1000)) {
		return nil, ErrInvalid
	}
	_, i, _, _ := intentEnvelope(c.intent)
	if _, e = VerifyIntent(ctx, c.intent, str(i, "recipient"), c.services.IntentAuthority, c.services.Policy); e != nil {
		return nil, ErrInvalid
	}
	if c.hop != nil && checkHop(ctx, c.hop.incoming, c.intent, c.services, c.hop.services) != nil {
		return nil, ErrInvalid
	}
	expires, _ := number(i, "expires")
	if u >= expires*1000 {
		return nil, ErrInvalid
	}
	if c.append(clientEvent{Kind: "send", ID: id, At: u}) != nil {
		return nil, ErrInvalid
	}
	c.lastMono = m
	if _, e = VerifyIntent(ctx, c.intent, str(i, "recipient"), c.services.IntentAuthority, c.services.Policy); e != nil {
		return nil, ErrInvalid
	}
	if c.hop != nil && checkHop(ctx, c.hop.incoming, c.intent, c.services, c.hop.services) != nil {
		return nil, ErrInvalid
	}
	u, _, e = c.sample(ctx)
	if e != nil || u >= expires*1000 {
		return nil, ErrInvalid
	}
	sendErr := c.services.Sender.Commit(ctx, id, append([]byte(nil), c.intent...))
	u, m, e = c.sample(ctx)
	if e != nil {
		return nil, ErrInvalid
	}
	c.lastMono = m
	c.lastWall = u
	if sendErr != nil {
		_ = c.append(clientEvent{Kind: "close", ID: id})
		return nil, ErrInvalid
	}
	return &ClientInvocation{owner: c, id: id, canonical: append([]byte(nil), c.intent...)}, nil
}
func (c *Client) closeInvocation(t *ClientInvocation) error {
	if t == nil || t.owner != c || !c.seen[t.id] {
		return ErrInvalid
	}
	return c.append(clientEvent{Kind: "close", ID: t.id})
}

// Failed records an unverified transport failure. It cannot create a remote verdict
// or authorize a new call. Reconciliation uses Begin with the same stored intent.
func (c *Client) Failed(t *ClientInvocation) error {
	if c == nil {
		return ErrInvalid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failed || c.file == nil {
		return ErrInvalid
	}
	return c.closeInvocation(t)
}

// Accept consumes this transport invocation even when authentication fails. It
// persists the first terminal before exposing output. A crash after persistence
// may lose delivery; it never causes redelivery or exactly-once downstream effects.
func (c *Client) Accept(ctx context.Context, t *ClientInvocation, raw []byte) (delivery *ClientDelivery, err error) {
	if c == nil || ctx == nil {
		return nil, ErrInvalid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	defer func() {
		if recover() != nil {
			c.failed = true
			delivery = nil
			err = ErrInvalid
		}
	}()
	if c.failed || c.file == nil || c.closeInvocation(t) != nil {
		return nil, ErrInvalid
	}
	if _, _, e := c.sample(ctx); e != nil {
		return nil, ErrInvalid
	}
	v, e := VerifyResult(ctx, raw, c.services.ResultAuthority, acceptedInvocation(c.intent))
	if e != nil {
		return nil, ErrInvalid
	}
	if len(c.terminal) != 0 {
		if v.Status() == "pending" || bytes.Equal(v.Canonical(), c.terminal) {
			return &ClientDelivery{ignored: true}, nil
		}
		return nil, ErrInvalid
	}
	if v.Status() == "pending" {
		return &ClientDelivery{status: "pending"}, nil
	}
	if c.append(clientEvent{Kind: "terminal", ID: t.id, ResultHex: hex.EncodeToString(v.Canonical())}) != nil {
		return nil, ErrInvalid
	}
	if _, _, e = c.sample(ctx); e != nil {
		return nil, ErrInvalid
	}
	v, e = VerifyResult(ctx, v.Canonical(), c.services.ResultAuthority, acceptedInvocation(c.intent))
	if e != nil {
		return nil, ErrInvalid
	}
	d := &ClientDelivery{status: v.Status(), first: true}
	if v.Status() == "completed" {
		d.output = v.Output()
	}
	return d, nil
}

// Close releases a healthy exclusive lock. Failed handles retain it for recovery.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.file == nil {
		return nil
	}
	e := c.file.Close()
	c.file = nil
	if e != nil || c.failed {
		return ErrInvalid
	}
	return os.Remove(c.lock)
}
