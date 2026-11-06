//go:build darwin

package registryworker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func testJournalBinding() journalBinding {
	return journalBinding{strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)}
}

func testJournalEvidence(raw []byte) EvidenceResult {
	return EvidenceResult{ReadbackComplete: true, ObservedBytes: int64(len(raw)), Prefix: bytes.Clone(raw)}
}

func completedJournal(b journalBinding, output []byte) []byte {
	digest := sha256.Sum256(output)
	return []byte(b.header() + "completed " + hex.EncodeToString(digest[:]) + "\n")
}

func TestWorkerJournalAcceptsOnlyItsExactAttemptAndOutput(t *testing.T) {
	b := testJournalBinding()
	output := []byte("verified fixture result")
	raw := completedJournal(b, output)
	got, err := readWorkerJournal(testJournalEvidence(raw), b, output)
	if err != nil || !got.Bound || !got.Started || !got.Terminal || got.Outcome != "completed" || got.OutputSHA256 == "" || got.Failure != "" {
		t.Fatal("complete same-attempt records refused", got, err)
	}
	for _, change := range []struct {
		name string
		edit func(*journalBinding, *EvidenceResult, *[]byte)
	}{
		{"another-attempt", func(b *journalBinding, _ *EvidenceResult, _ *[]byte) { b.nonce = strings.Repeat("d", 64) }},
		{"another-input", func(b *journalBinding, _ *EvidenceResult, _ *[]byte) { b.input = strings.Repeat("d", 64) }},
		{"another-worker", func(b *journalBinding, _ *EvidenceResult, _ *[]byte) { b.worker = strings.Repeat("d", 64) }},
		{"unreadable", func(_ *journalBinding, e *EvidenceResult, _ *[]byte) { e.ReadbackComplete = false }},
		{"overflow", func(_ *journalBinding, e *EvidenceResult, _ *[]byte) { e.LimitExceeded = true }},
		{"size-mismatch", func(_ *journalBinding, e *EvidenceResult, _ *[]byte) { e.ObservedBytes++ }},
		{"changed-output", func(_ *journalBinding, _ *EvidenceResult, out *[]byte) { *out = []byte("another result") }},
		{"invalid-binding", func(b *journalBinding, _ *EvidenceResult, _ *[]byte) { b.nonce = "invalid" }},
	} {
		t.Run(change.name, func(t *testing.T) {
			binding, evidence, actual := b, testJournalEvidence(raw), bytes.Clone(output)
			change.edit(&binding, &evidence, &actual)
			if _, err := readWorkerJournal(evidence, binding, actual); !errors.Is(err, ErrJournal) {
				t.Fatal("mismatched result accepted", err)
			}
		})
	}
	// An empty completed result is distinct from an absent terminal record.
	if _, err := readWorkerJournal(testJournalEvidence(completedJournal(b, nil)), b, nil); err != nil {
		t.Fatal("empty completed output refused", err)
	}
}

func TestWorkerJournalRetainsIncompleteAndFailedOutcomes(t *testing.T) {
	b := testJournalBinding()
	for _, code := range []string{"configuration", "compilation", "qualification", "execution", "verification", "cleanup", "canceled"} {
		got, err := readWorkerJournal(testJournalEvidence([]byte(b.header()+"failed "+code+"\n")), b, nil)
		if !errors.Is(err, ErrJournal) || !got.Bound || !got.Terminal || got.Outcome != "failed" || got.Failure != code {
			t.Fatal("explicit failure lost or accepted as success", code, got, err)
		}
	}
	got, err := readWorkerJournal(testJournalEvidence([]byte(b.header())), b, nil)
	if !errors.Is(err, ErrJournal) || !got.Started || got.Terminal || got.Outcome != "" {
		t.Fatal("unterminated start was accepted", got, err)
	}
	complete := completedJournal(b, []byte("result"))
	for cut := 0; cut < len(complete); cut++ {
		if _, err := readWorkerJournal(testJournalEvidence(complete[:cut]), b, []byte("result")); !errors.Is(err, ErrJournal) {
			t.Fatal("partial record accepted", cut)
		}
	}
	for _, raw := range [][]byte{
		append(bytes.Clone(complete), '\n'), append(bytes.Clone(complete), complete...),
		[]byte(b.header() + "completed " + strings.Repeat("z", 64) + "\n"),
		[]byte(b.header() + "failed arbitrary-private-details\n"),
		[]byte(b.header() + "failed execution\nfailed cleanup\n"),
		[]byte(b.header() + "started\n"),
		bytes.Repeat([]byte("x"), journalMaximumBytes+1),
	} {
		if _, err := readWorkerJournal(testJournalEvidence(raw), b, []byte("result")); !errors.Is(err, ErrJournal) {
			t.Fatal("malformed or duplicate terminal record accepted", err)
		}
	}
}

func testJournalFile(t *testing.T) (*os.File, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "journal")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f, path
}

func TestWorkerJournalWriterPersistsOneTerminalAndNeverRetries(t *testing.T) {
	for _, name := range []string{"complete", "failure", "cancel", "closed-file"} {
		t.Run(name, func(t *testing.T) {
			f, path := testJournalFile(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			b := testJournalBinding()
			w, err := newJournalWriter(ctx, f, b)
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "complete":
				err = w.Complete([]byte("result"))
			case "failure":
				if !errors.Is(w.Fail("private unstructured message"), ErrJournal) {
					t.Fatal("arbitrary failure details accepted")
				}
				err = w.Fail("execution")
			case "cancel":
				cancel()
				err = w.Complete(nil)
				if !errors.Is(err, context.Canceled) {
					t.Fatal("expired action wrote completion", err)
				}
			case "closed-file":
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
				err = w.Complete(nil)
				if err == nil {
					t.Fatal("closed descriptor accepted")
				}
			}
			if (name == "complete" || name == "failure") && err != nil {
				t.Fatal(err)
			}
			if !errors.Is(w.Complete(nil), ErrSpent) || !errors.Is(w.Fail("cleanup"), ErrSpent) {
				t.Fatal("terminal outcome was retried")
			}
			closeErr := w.Close()
			if name != "closed-file" && closeErr != nil {
				t.Fatal(closeErr)
			}
			if !errors.Is(w.Close(), ErrSpent) {
				t.Fatal("descriptor close was retried")
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got, verifyErr := readWorkerJournal(testJournalEvidence(raw), b, []byte("result"))
			if name == "complete" {
				if verifyErr != nil || got.Outcome != "completed" {
					t.Fatal(got, verifyErr)
				}
			} else if !errors.Is(verifyErr, ErrJournal) || got.Terminal != (name == "failure") {
				t.Fatal("failure or partial record became a completion", got, verifyErr)
			}
		})
	}
}

func TestWorkerJournalWriterRefusesInvalidOriginals(t *testing.T) {
	b := testJournalBinding()
	for _, name := range []string{"nil-context", "canceled", "nil-file", "closed", "directory", "nonempty", "public", "bad-binding", "read-only"} {
		t.Run(name, func(t *testing.T) {
			f, path := testJournalFile(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			selected := ctx
			binding := b
			switch name {
			case "nil-context":
				selected = nil
			case "canceled":
				cancel()
			case "nil-file":
				f = nil
			case "closed":
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
			case "directory":
				var err error
				f, err = os.Open(filepath.Dir(path))
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = f.Close() })
			case "nonempty":
				if _, err := f.WriteString("previous attempt"); err != nil {
					t.Fatal(err)
				}
			case "public":
				if err := f.Chmod(0o644); err != nil {
					t.Fatal(err)
				}
			case "bad-binding":
				binding.input = strings.Repeat("x", 64)
			case "read-only":
				var err error
				f, err = os.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = f.Close() })
			}
			if w, err := newJournalWriter(selected, f, binding); w != nil || err == nil {
				t.Fatal("invalid original descriptor accepted", w, err)
			}
		})
	}
	var nilWriter *JournalWriter
	if !errors.Is(nilWriter.Complete(nil), ErrJournal) || !errors.Is(nilWriter.Close(), ErrJournal) {
		t.Fatal("absent writer accepted")
	}
	foreign := &JournalWriter{self: &JournalWriter{}}
	if !errors.Is(foreign.Complete(nil), ErrJournal) || !errors.Is(foreign.Close(), ErrJournal) {
		t.Fatal("foreign owner accepted")
	}
	if err := foreign.UnmarshalJSON([]byte("{}")); !errors.Is(err, ErrJournal) {
		t.Fatal("serialized writer accepted", err)
	}
	f, _ := testJournalFile(t)
	w, err := newJournalWriter(context.Background(), f, b)
	if err != nil || w.Close() != nil || !errors.Is(w.Complete(nil), ErrSpent) {
		t.Fatal("closed incomplete writer was revived", err)
	}
}

func TestWorkerJournalConcurrentTerminalHasOneWinner(t *testing.T) {
	f, path := testJournalFile(t)
	b := testJournalBinding()
	w, err := newJournalWriter(context.Background(), f, b)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- w.Complete([]byte("same result"))
		}()
	}
	wg.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		} else if !errors.Is(err, ErrSpent) {
			t.Fatal(err)
		}
	}
	if winners != 1 || w.Close() != nil {
		t.Fatal("more than one terminal record was accepted", winners)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readWorkerJournal(testJournalEvidence(raw), b, []byte("same result")); err != nil {
		t.Fatal("concurrent completion corrupted the retained terminal", err)
	}
}

func TestWorkerJournalBindingIsFreshAndMatchesOnlyOriginalInput(t *testing.T) {
	input := []byte("finite fixture input")
	first := newJournalBinding(input, strings.Repeat("a", 64))
	second := newJournalBinding(input, strings.Repeat("a", 64))
	if !first.valid() || !second.valid() || first.nonce == second.nonce || first.input != second.input || first.worker != second.worker {
		t.Fatal("attempt identity was reused or changed the fixed input")
	}
	if got, err := journalBindingForInput(first.environment(), input); err != nil || got != first {
		t.Fatal("parent's original input binding refused", got, err)
	}
	for _, raw := range []string{"", strings.Repeat("a", 194), strings.Repeat("a", 64) + ":" + strings.Repeat("b", 64) + ":" + strings.Repeat("c", 64),
		strings.Repeat("a", 64) + ":" + strings.Repeat("z", 64) + ":" + strings.Repeat("c", 64)} {
		if b, err := journalBindingForInput(raw, input); b != (journalBinding{}) || !errors.Is(err, ErrJournal) {
			t.Fatal("invalid environment or another input produced a journal binding", b, err)
		}
	}
}

func TestWorkerJournalOwnsOnlyItsInheritedDescriptor(t *testing.T) {
	b := testJournalBinding()
	if w, err := openJournalFile(context.Background(), nil, b); w != nil || !errors.Is(err, ErrJournal) {
		t.Fatal("absent inherited descriptor accepted", w, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f, _ := testJournalFile(t)
	if w, err := openJournalFile(ctx, f, b); w != nil || err == nil {
		t.Fatal("expired bootstrap opened a journal", w, err)
	}
	if _, err := f.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatal("failed bootstrap left its inherited descriptor open", err)
	}
	f, path := testJournalFile(t)
	w, err := openJournalFile(context.Background(), f, b)
	if err != nil || w.Complete([]byte("result")) != nil || w.Close() != nil {
		t.Fatal("owned descriptor did not complete", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readWorkerJournal(testJournalEvidence(raw), b, []byte("result")); err != nil {
		t.Fatal("owned descriptor result refused", err)
	}
}
