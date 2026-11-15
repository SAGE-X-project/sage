package registryworker

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	journalMagic        = "sage-registry-worker-journal-v1"
	journalEnvironment  = "SAGE_REGISTRY_WORKER_JOURNAL"
	journalMaximumBytes = 512
)

// ErrJournal refuses missing, incomplete or mismatched child records. Their
// original retained bytes remain available even when ordinary output is nil.
var ErrJournal = errors.New("registry worker: incomplete or mismatched journal")

// There is exactly one inherited descriptor4 per worker process. A fresh
// *os.File wrapper does not duplicate an FD or share its closed-state tracking.
// Consume this claim before any wrapper; neither failure nor Close releases it.
var journalDescriptorClaimed atomic.Bool

type journalBinding struct {
	nonce, input, worker string
}

func newJournalBinding(input []byte, worker string) journalBinding {
	var nonce [32]byte
	// In the pinned stock Go runtime Read always fills the slice; an entropy
	// failure terminates the process rather than returning an error or zeros.
	_, _ = rand.Read(nonce[:])
	digest := sha256.Sum256(input)
	return journalBinding{hex.EncodeToString(nonce[:]), hex.EncodeToString(digest[:]), worker}
}

func (b journalBinding) valid() bool {
	return journalDigest(b.nonce) && journalDigest(b.input) && journalDigest(b.worker)
}

func journalDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func (b journalBinding) environment() string {
	return b.nonce + ":" + b.input + ":" + b.worker
}

func (b journalBinding) header() string {
	return journalMagic + "\n" + b.nonce + "\n" + b.input + "\n" + b.worker + "\nstarted\n"
}

// JournalResult records the child's claim, not independent truth about its
// computation. Terminal can remain true after another parent failure; only
// Run's nil error can publish Output, and neither grants Registry authority.
type JournalResult struct {
	Bound, Started, Terminal bool
	Outcome, OutputSHA256    string
	Failure                  string
}

func readWorkerJournal(e EvidenceResult, b journalBinding, output []byte) (out JournalResult, err error) {
	if !b.valid() || !e.ReadbackComplete || e.LimitExceeded || e.ObservedBytes != int64(len(e.Prefix)) ||
		len(e.Prefix) > journalMaximumBytes {
		return out, ErrJournal
	}
	header := b.header()
	if !strings.HasPrefix(string(e.Prefix), header) {
		return out, ErrJournal
	}
	out.Bound, out.Started = true, true
	tail := string(e.Prefix[len(header):])
	if strings.HasPrefix(tail, "completed ") && len(tail) == len("completed ")+64+1 && strings.HasSuffix(tail, "\n") {
		digest := tail[len("completed ") : len(tail)-1]
		if !journalDigest(digest) {
			return out, ErrJournal
		}
		out.Terminal, out.Outcome, out.OutputSHA256 = true, "completed", digest
		actual := sha256.Sum256(output)
		if digest != hex.EncodeToString(actual[:]) {
			return out, ErrJournal
		}
		return out, nil
	}
	if strings.HasPrefix(tail, "failed ") && strings.HasSuffix(tail, "\n") {
		code := strings.TrimSuffix(strings.TrimPrefix(tail, "failed "), "\n")
		if validJournalFailure(code) {
			out.Terminal, out.Outcome, out.Failure = true, "failed", code
		}
	}
	return out, ErrJournal
}

func validJournalFailure(code string) bool {
	switch code {
	case "configuration", "compilation", "qualification", "execution", "verification", "cleanup", "canceled":
		return true
	default:
		return false
	}
}

// JournalWriter owns descriptor4 of this single child. It writes one start
// record and at most one terminal record, syncing each. It never selects an
// evidence path, supplies a derivation or certifies the selected executable.
// The parent retains this descriptor's file after forced termination.
type JournalWriter struct {
	self   *JournalWriter
	mu     sync.Mutex
	ctx    context.Context
	file   *os.File
	spent  bool
	closed bool
}

func (*JournalWriter) UnmarshalJSON([]byte) error { return ErrJournal }

// OpenJournal is the worker-side entry for the parent's optional channel.
// The input bytes are checked against the parent's binding before any start
// record. The same process must perform its normal trusted-build checks; an
// inherited digest does not authenticate a worker to itself or qualify a host.
func OpenJournal(ctx context.Context, input []byte) (*JournalWriter, error) {
	if !journalDescriptorClaimed.CompareAndSwap(false, true) {
		return nil, ErrSpent
	}
	b, err := journalBindingForInput(os.Getenv(journalEnvironment), input)
	if err != nil {
		return nil, err
	}
	return openJournalFile(ctx, os.NewFile(4, "parent-terminal-evidence"), b)
}

func journalBindingForInput(raw string, input []byte) (journalBinding, error) {
	if len(raw) != 3*64+2 {
		return journalBinding{}, ErrJournal
	}
	parts := strings.Split(raw, ":")
	if len(parts) != 3 {
		return journalBinding{}, ErrJournal
	}
	b := journalBinding{parts[0], parts[1], parts[2]}
	digest := sha256.Sum256(input)
	if !b.valid() || b.input != hex.EncodeToString(digest[:]) {
		return journalBinding{}, ErrJournal
	}
	return b, nil
}

func openJournalFile(ctx context.Context, f *os.File, b journalBinding) (*JournalWriter, error) {
	if f == nil {
		return nil, ErrJournal
	}
	w, err := newJournalWriter(ctx, f, b)
	if err != nil {
		return nil, errors.Join(err, f.Close())
	}
	return w, nil
}

func newJournalWriter(ctx context.Context, file *os.File, b journalBinding) (*JournalWriter, error) {
	if ctx == nil || ctx.Err() != nil || file == nil || !b.valid() {
		return nil, ErrJournal
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Size() != 0 {
		return nil, ErrJournal
	}
	w := &JournalWriter{ctx: ctx, file: file}
	w.self = w
	if err := w.write(b.header()); err != nil {
		return nil, err
	}
	return w, nil
}

// Complete binds the child's terminal claim to the exact ordinary output.
// Call only after that output's underlying work and child cleanup succeeded.
// The parent separately checks exit, readiness, raw readback and its cleanup.
func (w *JournalWriter) Complete(output []byte) error {
	digest := sha256.Sum256(output)
	return w.terminal("completed " + hex.EncodeToString(digest[:]) + "\n")
}

// Fail records a fixed nonsensitive reason. Arbitrary messages, input bytes,
// keys and Registry objects are never written into this journal channel.
func (w *JournalWriter) Fail(code string) error {
	if !validJournalFailure(code) {
		return ErrJournal
	}
	return w.terminal("failed " + code + "\n")
}

func (w *JournalWriter) terminal(record string) error {
	if w == nil || w.self != w {
		return ErrJournal
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.spent || w.closed {
		return ErrSpent
	}
	w.spent = true
	return w.write(record)
}

func (w *JournalWriter) write(record string) error {
	if w.ctx.Err() != nil {
		return w.ctx.Err()
	}
	n, err := w.file.WriteString(record)
	if err != nil || n != len(record) {
		return errors.Join(ErrJournal, err)
	}
	return errors.Join(w.file.Sync(), w.ctx.Err())
}

// Close closes this writer's descriptor once. It does not invent a terminal
// record after an incomplete action or remove its parent-owned retained file.
func (w *JournalWriter) Close() error {
	if w == nil || w.self != w {
		return ErrJournal
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return ErrSpent
	}
	w.closed = true
	return w.file.Close()
}
