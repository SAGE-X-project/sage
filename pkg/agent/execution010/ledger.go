// Package execution010 provides durable execution denial and outcome storage.
// Callers must authenticate and authorize exact envelopes before every operation.
// This package neither verifies signatures nor dispatches tools.
package execution010

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
)

const header = "sage-execution-ledger|0.10.0\n"
const maxSize = 64 * 1024 * 1024
const maxRows = 4096

// ErrDenied covers invalid transitions, identities, and exhausted storage.
var ErrDenied = errors.New("execution ledger denied")

// ErrUnavailable means storage is closed, failed or cannot be recovered.
var ErrUnavailable = errors.New("execution ledger unavailable")

// Entry retains exact caller-validated canonical intent and terminal envelope bytes.
// Hex encoding is lowercase. Identity fields are trusted projections of IntentHex.
// State is RESERVED, EXECUTING, COMPLETED, REJECTED or UNKNOWN. No pruning occurs.
type Entry struct {
	Issuer    string `json:"issuer"`
	Recipient string `json:"recipient"`
	CallID    string `json:"call_id"`
	Nonce     string `json:"nonce"`
	Expires   int64  `json:"expires"`
	IntentHex string `json:"intent_hex"`
	State     string `json:"state"`
	ResultHex string `json:"result_hex"`
}
type callKey struct{ issuer, call string }
type nonceKey struct{ issuer, recipient, nonce string }

// Ledger is a serialized single-writer journal on trusted Linux/macOS storage.
// A crash leaves its exclusive lock. Only trusted administration may remove that
// lock after proving the old writer is gone. Disk rollback protection is external.
type Ledger struct {
	mu      sync.Mutex
	file    *os.File
	lock    string
	entries map[callKey]Entry
	nonces  map[nonceKey]callKey
	size    int64
	rows    int
	failed  bool
}

func atom(s string) bool {
	if len(s) == 0 || len(s) > 256 {
		return false
	}
	for _, c := range []byte(s) {
		if c < 33 || c > 126 || c == '"' || c == '\\' || c == '<' || c == '>' || c == '&' {
			return false
		}
	}
	return true
}
func envelope(s string, empty bool) bool {
	if s == "" {
		return empty
	}
	if len(s) > 2*1024*1024 || len(s)%2 != 0 {
		return false
	}
	for _, c := range []byte(s) {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	_, e := hex.DecodeString(s)
	return e == nil
}
func valid(e Entry) bool {
	if !atom(e.Issuer) || !atom(e.Recipient) || !atom(e.CallID) || !atom(e.Nonce) || e.Expires < 0 || e.Expires > 9007199254740991 || !envelope(e.IntentHex, false) {
		return false
	}
	switch e.State {
	case "RESERVED", "EXECUTING":
		return e.ResultHex == ""
	case "UNKNOWN":
		return envelope(e.ResultHex, true)
	case "COMPLETED", "REJECTED":
		return envelope(e.ResultHex, false)
	default:
		return false
	}
}
func identity(a, b Entry) bool {
	a.State = ""
	a.ResultHex = ""
	b.State = ""
	b.ResultHex = ""
	return a == b
}
func (l *Ledger) check(e Entry) (bool, error) {
	if !valid(e) {
		return false, ErrDenied
	}
	k := callKey{e.Issuer, e.CallID}
	n := nonceKey{e.Issuer, e.Recipient, e.Nonce}
	old, exists := l.entries[k]
	if owner, ok := l.nonces[n]; ok && owner != k {
		return false, ErrDenied
	}
	if !exists {
		if e.State != "RESERVED" && e.State != "REJECTED" {
			return false, ErrDenied
		}
		return true, nil
	}
	if !identity(old, e) {
		return false, ErrDenied
	}
	if old == e {
		return false, nil
	}
	switch old.State {
	case "RESERVED":
		if e.State == "EXECUTING" || e.State == "REJECTED" || e.State == "UNKNOWN" {
			return true, nil
		}
	case "EXECUTING":
		if e.State == "COMPLETED" || e.State == "UNKNOWN" {
			return true, nil
		}
	case "UNKNOWN":
		if old.ResultHex == "" && e.State == "UNKNOWN" && e.ResultHex != "" {
			return true, nil
		}
	}
	return false, ErrDenied
}
func (l *Ledger) remember(e Entry) {
	k := callKey{e.Issuer, e.CallID}
	l.entries[k] = e
	l.nonces[nonceKey{e.Issuer, e.Recipient, e.Nonce}] = k
}
func (l *Ledger) append(e Entry) error {
	if l.rows >= maxRows {
		return ErrDenied
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if l.size+int64(len(b)) > maxSize {
		return ErrDenied
	}
	n, err := l.file.Write(b)
	if err != nil || n != len(b) {
		l.failed = true
		return ErrUnavailable
	}
	if l.file.Sync() != nil {
		l.failed = true
		return ErrUnavailable
	}
	l.size += int64(n)
	l.rows++
	l.remember(e)
	return nil
}

// Open creates only when explicitly requested. Ordinary reopen requires complete
// existing state and durably converts RESERVED/EXECUTING entries to UNKNOWN before
// returning. It never dispatches, reconstructs lost state or installs policy epochs.
// Initialization is only for a new isolated execution scope with no old grants.
func Open(path string, create bool) (*Ledger, error) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return nil, ErrUnavailable
	}
	lock := path + ".lock"
	// #nosec G304 -- trusted caller-owned storage path, exclusive writer lock.
	guard, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	if err = guard.Close(); err != nil {
		_ = os.Remove(lock)
		return nil, err
	}
	flags := os.O_RDWR | os.O_APPEND
	if create {
		flags |= os.O_CREATE | os.O_EXCL
	}
	// #nosec G304 -- trusted deployment path; normal reopen never creates missing state.
	f, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		_ = os.Remove(lock)
		return nil, err
	}
	l := &Ledger{file: f, lock: lock, entries: map[callKey]Entry{}, nonces: map[nonceKey]callKey{}}
	fail := func(err error) (*Ledger, error) { _ = l.Close(); return nil, err }
	if create {
		if _, err = f.WriteString(header); err != nil {
			return fail(err)
		}
		if err = f.Sync(); err != nil {
			return fail(err)
		}
		// #nosec G304 -- sync the parent of the trusted configured journal.
		dir, e := os.Open(filepath.Dir(path))
		if e != nil {
			return fail(e)
		}
		e = dir.Sync()
		closeErr := dir.Close()
		if e != nil {
			return fail(e)
		}
		if closeErr != nil {
			return fail(closeErr)
		}
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return fail(err)
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxSize+1))
	if err != nil {
		return fail(err)
	}
	if len(raw) > maxSize || !bytes.HasPrefix(raw, []byte(header)) || !bytes.HasSuffix(raw, []byte("\n")) {
		return fail(ErrUnavailable)
	}
	l.size = int64(len(raw))
	body := raw[len(header):]
	if len(body) > 0 {
		for _, line := range bytes.Split(body[:len(body)-1], []byte("\n")) {
			var entry Entry
			if json.Unmarshal(line, &entry) != nil {
				return fail(ErrUnavailable)
			}
			canonical, e := json.Marshal(entry)
			if e != nil || !bytes.Equal(canonical, line) {
				return fail(ErrUnavailable)
			}
			changed, e := l.check(entry)
			if e != nil || !changed || l.rows >= maxRows {
				return fail(ErrUnavailable)
			}
			l.rows++
			l.remember(entry)
		}
	}
	keys := make([]callKey, 0, len(l.entries))
	for k := range l.entries {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].issuer != keys[j].issuer {
			return keys[i].issuer < keys[j].issuer
		}
		return keys[i].call < keys[j].call
	})
	for _, k := range keys {
		e := l.entries[k]
		if e.State == "RESERVED" || e.State == "EXECUTING" {
			e.State = "UNKNOWN"
			if err = l.append(e); err != nil {
				// Recovery is an integrity operation. Even failures detected before
				// the write begins keep exclusive ownership for administration.
				l.failed = true
				return fail(err)
			}
		}
	}
	return l, nil
}

// Commit atomically persists the call and nonce identity with its state. Changed
// is false for an identical retry; that is never permission to execute again.
// EXECUTING must be durable before effects. Callers separately serialize current
// authorization/retirement and dispatch; this storage API is not a dispatch gate.
// A terminal result must already be verified, signed and bound by the caller.
func (l *Ledger) Commit(e Entry) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.failed || l.file == nil {
		return false, ErrUnavailable
	}
	changed, err := l.check(e)
	if err != nil || !changed {
		return changed, err
	}
	if err = l.append(e); err != nil {
		return false, err
	}
	return true, nil
}

// Reserve atomically inserts a RESERVED identity or reads the exact existing
// identity in any state. It never transitions an existing entry. The caller must
// authenticate every attempt; returned result bytes remain unverified storage.
func (l *Ledger) Reserve(e Entry) (Entry, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.failed || l.file == nil {
		return Entry{}, false, ErrUnavailable
	}
	if !valid(e) || e.State != "RESERVED" {
		return Entry{}, false, ErrDenied
	}
	if old, ok := l.entries[callKey{e.Issuer, e.CallID}]; ok {
		if !identity(old, e) {
			return Entry{}, false, ErrDenied
		}
		return old, false, nil
	}
	if _, err := l.check(e); err != nil {
		return Entry{}, false, err
	}
	if err := l.append(e); err != nil {
		return Entry{}, false, err
	}
	return e, true, nil
}

// Lookup returns storage state only, never authority or a verified result.
// Retrieval freshness, live identity, policy and result expiry are caller duties.
func (l *Ledger) Lookup(issuer, call string) (Entry, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.failed || l.file == nil {
		return Entry{}, false, ErrUnavailable
	}
	e, ok := l.entries[callKey{issuer, call}]
	return e, ok, nil
}

// Close releases the lock only for a healthy handle. Failed storage requires
// explicit administrative recovery; its lock is retained even after close.
func (l *Ledger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	if err != nil {
		l.failed = true
		return err
	}
	if l.failed {
		return ErrUnavailable
	}
	return os.Remove(l.lock)
}
