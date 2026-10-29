package registry011

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestProviderEnrollmentOwnedReopenAndRecovery(t *testing.T) {
	cfg := enrollmentTestNamespace(t, nil)
	candidate := enrollmentTestContext(t)
	n := enrollmentTestOpen(t, cfg)
	snapshot, err := n.ReadCandidate()
	if err != nil {
		t.Fatal(err)
	}
	empty, err := snapshot.Diagnostic()
	if err != nil || len(empty.Enrollment) != 0 || empty.Tip != cfg.AnchorDigest {
		t.Fatal("initial read", err)
	}
	if second, err := OpenCandidateProviderNamespace(cfg); second != nil || err == nil {
		t.Fatal("second lock owner")
	}
	receipt, err := n.EnrollCandidate(snapshot, candidate)
	if err != nil {
		t.Fatal(err)
	}
	diagnostic, err := receipt.Diagnostic()
	if err != nil {
		t.Fatal(err)
	}
	if diagnostic.Identity != candidate.identity || diagnostic.ExpectedOldTip != cfg.AnchorDigest || diagnostic.Tip == cfg.AnchorDigest || diagnostic.Fence != 1 || len(diagnostic.Enrollment) == 0 {
		t.Fatal("wrong exact acknowledgement")
	}
	if diagnostic.RootDigest != enrollmentTestHash("sage-provider-enrollment-v1", diagnostic.Enrollment) {
		t.Fatal("root binding")
	}
	stored, err := os.ReadFile(filepath.Join(cfg.Directory, "ENROLLMENT"))
	if err != nil {
		t.Fatal(err)
	}
	if diagnostic.Tip != enrollmentTestHash("sage-provider-namespace-entry-v1", stored) {
		t.Fatal("tip binding")
	}
	if _, err := snapshot.Diagnostic(); err != ErrProviderRetentionConflict {
		t.Fatal("stale snapshot")
	}
	diagnostic.Enrollment[0] = '!'
	diagnostic.Identity.Policy = "changed"
	diagnostic, err = receipt.Diagnostic()
	if err != nil || diagnostic.Enrollment[0] != '{' || diagnostic.Identity.Policy != candidate.identity.Policy {
		t.Fatal("mutable receipt")
	}
	current, err := n.ReadCandidate()
	if err != nil {
		t.Fatal(err)
	}
	view, err := current.Diagnostic()
	if err != nil {
		t.Fatal(err)
	}
	view.Enrollment[0] = '!'
	if again, err := current.Diagnostic(); err != nil || again.Enrollment[0] != '{' {
		t.Fatal("mutable read")
	}
	if r, err := n.EnrollCandidate(current, candidate); r != nil || err != ErrProviderRetentionConflict {
		t.Fatal("existing enrollment overwritten")
	}
	if err := n.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := current.Diagnostic(); err != ErrProviderRetentionConflict {
		t.Fatal("closed snapshot")
	}
	if _, err := receipt.Diagnostic(); err != nil {
		t.Fatal("historical diagnostic lost")
	}
	if stale, err := OpenCandidateProviderNamespace(cfg); stale != nil || err != ErrProviderRetentionConflict {
		t.Fatal("stale external high-water pin")
	}
	recovered, replayed, err := RecoverCandidateProviderEnrollment(cfg, candidate)
	if err != nil {
		t.Fatal(err)
	}
	rd, err := replayed.Diagnostic()
	if err != nil || rd.Tip != diagnostic.Tip || !bytes.Equal(rd.Enrollment, diagnostic.Enrollment) {
		t.Fatal("different recovered operation")
	}
	if err := recovered.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(filepath.Join(cfg.Directory, "ENROLLMENT"))
	if err != nil || !bytes.Equal(again, stored) {
		t.Fatal("recovery rewrote evidence")
	}
	cfg.ExpectedTip = diagnostic.Tip
	reopened := enrollmentTestOpen(t, cfg)
	read, err := reopened.ReadCandidate()
	if err != nil {
		t.Fatal(err)
	}
	if v, err := read.Diagnostic(); err != nil || v.Tip != cfg.ExpectedTip || !bytes.Equal(v.Enrollment, diagnostic.Enrollment) {
		t.Fatal("reopen changed root")
	}
	if r, err := reopened.EnrollCandidate(read, candidate); r != nil || err != ErrProviderRetentionConflict {
		t.Fatal("reopen reset first use")
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
	if a, b, err := RecoverCandidateProviderEnrollment(cfg, candidate); a != nil || b != nil || err != ErrProviderRetentionConflict {
		t.Fatal("recovery ignored expected old tip")
	}
}

func TestProviderEnrollmentInitialStateMatrix(t *testing.T) {
	for _, existing := range []bool{false, true} {
		for _, file := range []string{"absent", "exact", "partial", "foreign"} {
			name := "empty/"
			if existing {
				name = "retained/"
			}
			name += file
			t.Run(name, func(t *testing.T) {
				var retained any
				if existing {
					retained = enrollmentTestRoot(t, false, "retired")
				}
				cfg := enrollmentTestNamespace(t, retained)
				candidate := enrollmentTestContext(t)
				if file != "absent" {
					id := candidate.identity
					if file == "foreign" {
						id.SourceID = "0x" + strings.Repeat("8", 64)
					}
					_, raw, err := makeCandidateEnrollment(id, cfg.AnchorDigest)
					if err != nil {
						t.Fatal(err)
					}
					if file == "partial" {
						raw = []byte("{")
					}
					if err := os.WriteFile(filepath.Join(cfg.Directory, "ENROLLMENT"), raw, 0600); err != nil {
						t.Fatal(err)
					}
				}
				n, err := OpenCandidateProviderNamespace(cfg)
				if file == "absent" {
					if err != nil {
						t.Fatal(err)
					}
					s, err := n.ReadCandidate()
					if err != nil {
						t.Fatal(err)
					}
					if existing {
						if r, err := n.EnrollCandidate(s, candidate); r != nil || err != ErrProviderRetentionConflict {
							t.Fatal("retained tombstone reset")
						}
					}
					if err := n.Close(); err != nil {
						t.Fatal(err)
					}
				} else if n != nil || err == nil {
					t.Fatal("stale or malformed entry accepted")
				}
				n, r, err := RecoverCandidateProviderEnrollment(cfg, candidate)
				if !existing && file == "exact" {
					if err != nil || r == nil {
						t.Fatal("exact recovery", err)
					}
					if err := n.Close(); err != nil {
						t.Fatal(err)
					}
				} else if n != nil || r != nil || err == nil {
					t.Fatal("ambiguous recovery accepted")
				}
			})
		}
	}
	for _, status := range []string{"enrolling", "active", "quarantined", "fenced", "retired"} {
		t.Run(status, func(t *testing.T) {
			cfg := enrollmentTestNamespace(t, enrollmentTestRoot(t, false, status))
			n := enrollmentTestOpen(t, cfg)
			s, err := n.ReadCandidate()
			if err != nil {
				t.Fatal(err)
			}
			before, err := s.Diagnostic()
			if err != nil {
				t.Fatal(err)
			}
			if r, err := n.EnrollCandidate(s, enrollmentTestContext(t)); r != nil || err != ErrProviderRetentionConflict {
				t.Fatal("existing root replaced")
			}
			after, err := s.Diagnostic()
			if err != nil || !bytes.Equal(before.Enrollment, after.Enrollment) {
				t.Fatal("root or tombstone lost")
			}
		})
	}
}

func TestProviderEnrollmentConcurrencyAndOwnerBinding(t *testing.T) {
	cfg := enrollmentTestNamespace(t, nil)
	n := enrollmentTestOpen(t, cfg)
	s, err := n.ReadCandidate()
	if err != nil {
		t.Fatal(err)
	}
	candidate := enrollmentTestContext(t)
	other := enrollmentTestOpen(t, enrollmentTestNamespace(t, nil))
	if r, err := other.EnrollCandidate(s, candidate); r != nil || err != ErrProviderRetentionConflict {
		t.Fatal("foreign read owner")
	}
	for _, read := range []*CandidateNamespaceSnapshot{nil, {}} {
		if r, err := n.EnrollCandidate(read, candidate); r != nil || err != ErrProviderRetentionConflict {
			t.Fatal("zero read")
		}
	}
	for _, change := range []string{"nil", "zero", "namespace", "instance", "generation"} {
		t.Run(change, func(t *testing.T) {
			var c *ProviderCandidateContext
			if change != "nil" {
				c = &ProviderCandidateContext{}
				if change != "zero" {
					*c = *candidate
				}
			}
			switch change {
			case "namespace":
				c.identity.Namespace = "0x" + strings.Repeat("8", 64)
			case "instance":
				c.identity.Instance = "0x" + strings.Repeat("8", 64)
			case "generation":
				c.identity.Generation = 1
			}
			if r, err := n.EnrollCandidate(s, c); r != nil || err != ErrConfiguration {
				t.Fatal("invalid proposed identity")
			}
			if a, b, err := RecoverCandidateProviderEnrollment(cfg, c); a != nil || b != nil || err != ErrConfiguration {
				t.Fatal("invalid recovery context")
			}
		})
	}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() { _, err := n.EnrollCandidate(s, candidate); results <- err })
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		switch err {
		case nil:
			success++
		case ErrProviderRetentionConflict:
			conflict++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal("multiple initial CAS winners", success, conflict)
	}
}

func TestProviderEnrollmentPinsFilesAndBoundedMetadata(t *testing.T) {
	base := enrollmentTestNamespace(t, nil)
	for _, mode := range []string{"directory", "namespace", "instance", "anchor", "tip", "zero-bytes", "negative-bytes", "large-bytes"} {
		t.Run(mode, func(t *testing.T) {
			cfg := base
			switch mode {
			case "directory":
				cfg.Directory = ""
			case "namespace":
				cfg.Namespace = ""
			case "instance":
				cfg.Instance = ""
			case "anchor":
				cfg.AnchorDigest = ""
			case "tip":
				cfg.ExpectedTip = ""
			case "zero-bytes":
				cfg.MaxBytes = 0
			case "negative-bytes":
				cfg.MaxBytes = -1
			case "large-bytes":
				cfg.MaxBytes = 64<<10 + 1
			}
			if n, err := OpenCandidateProviderNamespace(cfg); n != nil || err != ErrConfiguration {
				t.Fatal("configuration defaulted")
			}
		})
	}
	for _, mode := range []string{"missing-directory", "missing-anchor", "unknown-file", "many-files", "bad-anchor", "wrong-pin", "anchor-directory", "oversize-anchor", "stale-lock"} {
		t.Run(mode, func(t *testing.T) {
			cfg := enrollmentTestNamespace(t, nil)
			anchor := filepath.Join(cfg.Directory, "ANCHOR")
			switch mode {
			case "missing-directory":
				cfg.Directory = filepath.Join(cfg.Directory, "absent")
			case "missing-anchor":
				if err := os.Remove(anchor); err != nil {
					t.Fatal(err)
				}
			case "unknown-file":
				if err := os.WriteFile(filepath.Join(cfg.Directory, "UNKNOWN"), []byte("data"), 0600); err != nil {
					t.Fatal(err)
				}
			case "many-files":
				for _, name := range []string{"A", "B", "C", "D"} {
					if err := os.WriteFile(filepath.Join(cfg.Directory, name), []byte("data"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			case "bad-anchor":
				if err := os.WriteFile(anchor, []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
			case "wrong-pin":
				cfg.AnchorDigest = "0x" + strings.Repeat("8", 64)
			case "anchor-directory":
				if err := os.Remove(anchor); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(anchor, 0700); err != nil {
					t.Fatal(err)
				}
			case "oversize-anchor":
				cfg.MaxBytes = 1
			case "stale-lock":
				if err := os.WriteFile(filepath.Join(cfg.Directory, "LOCK"), []byte("locked\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if n, err := OpenCandidateProviderNamespace(cfg); n != nil || err == nil {
				t.Fatal("unavailable authority became initial state")
			}
			if mode == "stale-lock" {
				if raw, err := os.ReadFile(filepath.Join(cfg.Directory, "LOCK")); err != nil || string(raw) != "locked\n" {
					t.Fatal("stale lock stolen")
				}
			}
		})
	}
	cfg := enrollmentTestNamespace(t, nil)
	anchor, err := os.ReadFile(filepath.Join(cfg.Directory, "ANCHOR"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxBytes = len(anchor)
	n := enrollmentTestOpen(t, cfg)
	s, err := n.ReadCandidate()
	if err != nil {
		t.Fatal(err)
	}
	if r, err := n.EnrollCandidate(s, enrollmentTestContext(t)); r != nil || err != ErrProviderRetention {
		t.Fatal("metadata byte ceiling ignored")
	}
	if _, err := os.Stat(filepath.Join(cfg.Directory, "ENROLLMENT")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("oversize entry written")
	}
}

func TestProviderEnrollmentFailuresFenceWithoutReset(t *testing.T) {
	for _, mode := range []string{"sync", "readback", "read-drift", "enroll-drift", "closed-root", "crash-lock"} {
		t.Run(mode, func(t *testing.T) {
			cfg := enrollmentTestNamespace(t, nil)
			n, err := OpenCandidateProviderNamespace(cfg)
			if err != nil {
				t.Fatal(err)
			}
			s, err := n.ReadCandidate()
			if err != nil {
				t.Fatal(err)
			}
			originalSync := n.disk.syncFile
			candidate := enrollmentTestContext(t)
			switch mode {
			case "sync":
				n.disk.syncFile = func(*os.File) error { return errors.New("owned fixture Sync failure") }
			case "readback":
				n.disk.syncFile = func(f *os.File) error {
					if err := originalSync(f); err != nil {
						return err
					}
					if filepath.Base(f.Name()) == "ENROLLMENT" {
						return os.WriteFile(filepath.Join(cfg.Directory, "ENROLLMENT"), []byte("{"), 0600)
					}
					return nil
				}
			case "read-drift", "enroll-drift":
				if err := os.WriteFile(filepath.Join(cfg.Directory, "ANCHOR"), []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
			case "closed-root", "crash-lock":
				if err := n.disk.root.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "read-drift" || mode == "closed-root" || mode == "crash-lock" {
				if r, err := n.ReadCandidate(); r != nil || err != ErrProviderRetention {
					t.Fatal("failed read returned a token")
				}
			} else {
				if r, err := n.EnrollCandidate(s, candidate); r != nil || err != ErrProviderRetention {
					t.Fatal("failed commit returned a receipt")
				}
			}
			if r, err := n.ReadCandidate(); r != nil || err != ErrProviderRetention {
				t.Fatal("fenced read")
			}
			if r, err := n.EnrollCandidate(s, candidate); r != nil || err != ErrProviderRetention {
				t.Fatal("fenced commit")
			}
			if _, err := s.Diagnostic(); err != ErrProviderRetentionConflict {
				t.Fatal("fenced snapshot")
			}
			n.disk.syncFile = originalSync
			closeErr := n.Close()
			if mode == "closed-root" || mode == "crash-lock" {
				if closeErr == nil {
					t.Fatal("close of unavailable root succeeded")
				}
				if err := n.Close(); err != closeErr {
					t.Fatal("repeated close erased failure")
				}
				if reopened, err := OpenCandidateProviderNamespace(cfg); reopened != nil || err == nil {
					t.Fatal("crash lock automatically recovered")
				}
				return
			}
			if closeErr != nil {
				t.Fatal(closeErr)
			}
			recovered, receipt, err := RecoverCandidateProviderEnrollment(cfg, candidate)
			if mode == "sync" {
				if err != nil || receipt == nil {
					t.Fatal("exact postfailure recovery", err)
				}
				if err := recovered.Close(); err != nil {
					t.Fatal(err)
				}
			} else if recovered != nil || receipt != nil || err == nil {
				t.Fatal("partial/different evidence reset")
			}
		})
	}
}

func TestProviderEnrollmentEmptyPathCannotReplacePinnedTip(t *testing.T) {
	cfg := enrollmentTestNamespace(t, nil)
	n := enrollmentTestOpen(t, cfg)
	s, err := n.ReadCandidate()
	if err != nil {
		t.Fatal(err)
	}
	r, err := n.EnrollCandidate(s, enrollmentTestContext(t))
	if err != nil {
		t.Fatal(err)
	}
	d, err := r.Diagnostic()
	if err != nil {
		t.Fatal(err)
	}
	if err := n.Close(); err != nil {
		t.Fatal(err)
	}
	other := enrollmentTestNamespace(t, nil)
	other.ExpectedTip = d.Tip
	if n, err := OpenCandidateProviderNamespace(other); n != nil || err != ErrProviderRetentionConflict {
		t.Fatal("empty directory reset independently pinned history")
	}
	other.ExpectedTip = cfg.AnchorDigest
	other.Namespace = "0x" + strings.Repeat("8", 64)
	if n, err := OpenCandidateProviderNamespace(other); n != nil || err == nil {
		t.Fatal("changed namespace accepted")
	}
}

func TestProviderEnrollmentZeroAndJSONStateRefuse(t *testing.T) {
	for _, n := range []*CandidateProviderNamespace{nil, {}} {
		if s, err := n.ReadCandidate(); s != nil || err != ErrProviderRetention {
			t.Fatal("zero read")
		}
		if r, err := n.EnrollCandidate(nil, nil); r != nil || err != ErrProviderRetention {
			t.Fatal("zero enrollment")
		}
		if err := n.Close(); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range []*CandidateNamespaceSnapshot{nil, {}} {
		if _, err := s.Diagnostic(); err != ErrProviderRetention {
			t.Fatal("zero snapshot")
		}
	}
	for _, r := range []*CandidateRetentionReceipt{nil, {}} {
		if _, err := r.Diagnostic(); err != ErrProviderRetention {
			t.Fatal("zero receipt")
		}
	}
	for _, value := range []any{&CandidateProviderNamespace{}, &CandidateNamespaceSnapshot{}, &CandidateRetentionReceipt{}} {
		if err := json.Unmarshal([]byte(`{"ready":true,"tip":"0x123"}`), value); err != ErrProviderRetention {
			t.Fatal("JSON reconstructed private state")
		}
	}
}
