//go:build linux || darwin

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

// These owned temporary stores and explicit pins are candidate fixtures only.
// No fixture is an operating namespace, absence witness or external ACK.
func retentionTestConfig(t *testing.T) CandidateRetentionConfig {
	t.Helper()
	id, _ := enrollmentTestContext(t).Identity()
	anchor, err := retentionEncode(struct{ Format, Namespace, Selection string }{
		"sage-retaining-authority-v1", id.Namespace, "0x" + strings.Repeat("7", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ANCHOR"), anchor, 0600); err != nil {
		t.Fatal(err)
	}
	// The permanent lock is provisioned only as an owned test fixture.
	// Production open never creates or replaces its ownership inode.
	if err := os.WriteFile(filepath.Join(dir, "AUTHORITY.LOCK"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	pin := enrollmentTestHash("sage-retaining-anchor-v1", anchor)
	return CandidateRetentionConfig{dir, id.Namespace, pin, pin, 8, 16 << 10, 128 << 10}
}

func retentionTestOpen(t *testing.T, config CandidateRetentionConfig) *CandidateRetentionLog {
	t.Helper()
	a, err := OpenCandidateRetentionLog(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	return a
}

func retentionTestEnroll(t *testing.T, a *CandidateRetentionLog) CandidateRetentionDiagnostic {
	t.Helper()
	c := enrollmentTestContext(t)
	id, _ := c.Identity()
	read, err := a.ReadInstance(id.Instance)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := a.EnrollInstance(read, c, "0x"+strings.Repeat("8", 64))
	if err != nil {
		t.Fatal(err)
	}
	d, err := pending.Diagnostic()
	if err != nil || d.State != "pending-external-ack" {
		t.Fatal("local append must remain pending", err)
	}
	return d
}

func TestCandidateRetentionPendingReopenAndCopies(t *testing.T) {
	config := retentionTestConfig(t)
	a := retentionTestOpen(t, config)
	id, _ := enrollmentTestContext(t).Identity()
	before, err := a.ReadInstance(id.Instance)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := before.Diagnostic()
	if err != nil || len(initial.Root) != 0 || initial.Sequence != 0 || initial.Tip != config.AnchorDigest || initial.State != "local-observation" {
		t.Fatal(initial, err)
	}
	d := retentionTestEnroll(t, a)
	if d.Sequence != 1 || d.PreviousTip != config.AnchorDigest || d.Tip == d.PreviousTip || d.RootDigest != enrollmentTestHash("sage-provider-enrollment-v1", d.Root) {
		t.Fatal(d)
	}
	if _, err := before.Diagnostic(); err == nil {
		t.Fatal("stale read accepted")
	}
	read, err := a.ReadInstance(id.Instance)
	if err != nil {
		t.Fatal(err)
	}
	observed, err := read.Diagnostic()
	if err != nil || !bytes.Equal(observed.Root, d.Root) {
		t.Fatal(err)
	}
	observed.Root[0] = '!'
	again, err := read.Diagnostic()
	if err != nil || !bytes.Equal(again.Root, d.Root) {
		t.Fatal("caller mutated retained root")
	}
	if _, err := a.EnrollInstance(read, enrollmentTestContext(t), d.Operation); err == nil {
		t.Fatal("pending handle appended twice")
	}
	if _, err := OpenCandidateRetentionLog(config); err == nil {
		t.Fatal("concurrent local owner")
	}
	lockBefore, err := os.Stat(filepath.Join(config.Directory, "AUTHORITY.LOCK"))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil || a.Close() != nil {
		t.Fatal("idempotent close", err)
	}
	if _, err := OpenCandidateRetentionLog(config); err == nil {
		t.Fatal("old expected tip accepted by normal reopen")
	}
	config.ExpectedTip = d.Tip
	reopened := retentionTestOpen(t, config)
	lockAfter, _ := os.Stat(filepath.Join(config.Directory, "AUTHORITY.LOCK"))
	if !os.SameFile(lockBefore, lockAfter) {
		t.Fatal("lock inode was replaced")
	}
	if _, err := reopened.EnrollInstance(read, enrollmentTestContext(t), d.Operation); err == nil {
		t.Fatal("foreign read")
	}
	newRead, err := reopened.ReadInstance(id.Instance)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.EnrollInstance(newRead, enrollmentTestContext(t), "0x"+strings.Repeat("9", 64)); err == nil {
		t.Fatal("existing instance replaced")
	}
	if _, err := a.ReadInstance(id.Instance); err == nil {
		t.Fatal("closed owner read")
	}
}

func TestCandidateRetentionExactPendingRecovery(t *testing.T) {
	config := retentionTestConfig(t)
	a := retentionTestOpen(t, config)
	d := retentionTestEnroll(t, a)
	rawBefore, _ := os.ReadFile(filepath.Join(config.Directory, retentionName(1)))
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"operation", "old-tip", "source", "generation", "namespace", "nil", "zero"} {
		t.Run(mode, func(t *testing.T) {
			cfg, c, op := config, enrollmentTestContext(t), d.Operation
			switch mode {
			case "operation":
				op = "0x" + strings.Repeat("9", 64)
			case "old-tip":
				cfg.ExpectedTip = d.Tip
			case "source":
				c.identity.SourceID = "0x" + strings.Repeat("9", 64)
			case "generation":
				c.identity.Generation = 1
			case "namespace":
				cfg.Namespace = "0x" + strings.Repeat("9", 64)
			case "nil":
				c = nil
			case "zero":
				op = ""
			}
			if owner, pending, err := RecoverCandidateRetentionEnrollment(cfg, c, op); err == nil || owner != nil || pending != nil {
				if owner != nil {
					_ = owner.Close()
				}
				t.Fatal("inexact recovery")
			}
		})
	}
	recovered, pending, err := RecoverCandidateRetentionEnrollment(config, enrollmentTestContext(t), d.Operation)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = recovered.Close() })
	diagnostic, err := pending.Diagnostic()
	if err != nil || diagnostic.State != "pending-external-ack" || diagnostic.Tip != d.Tip || !bytes.Equal(diagnostic.Root, d.Root) {
		t.Fatal(diagnostic, err)
	}
	rawAfter, _ := os.ReadFile(filepath.Join(config.Directory, retentionName(1)))
	if !bytes.Equal(rawBefore, rawAfter) {
		t.Fatal("recovery rewrote entry")
	}
	id, _ := enrollmentTestContext(t).Identity()
	read, err := recovered.ReadInstance(id.Instance)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recovered.EnrollInstance(read, enrollmentTestContext(t), d.Operation); err == nil {
		t.Fatal("recovery granted another append")
	}
	if err := recovered.Close(); err != nil {
		t.Fatal(err)
	}
	if historical, err := pending.Diagnostic(); err != nil || historical.State != "pending-external-ack" {
		t.Fatal("historical pending is not an ACK", err)
	}
	config.ExpectedTip = d.Tip
	_ = retentionTestOpen(t, config)
}

func TestCandidateRetentionConfigurationAndNoProvision(t *testing.T) {
	for _, mode := range []string{"directory", "namespace", "anchor", "expected", "records-zero", "records-large", "record-zero", "record-large", "stored-zero", "stored-large", "record-over-stored"} {
		t.Run(mode, func(t *testing.T) {
			c := retentionTestConfig(t)
			switch mode {
			case "directory":
				c.Directory = ""
			case "namespace":
				c.Namespace = ""
			case "anchor":
				c.AnchorDigest = ""
			case "expected":
				c.ExpectedTip = ""
			case "records-zero":
				c.MaxRecords = 0
			case "records-large":
				c.MaxRecords = 65537
			case "record-zero":
				c.MaxRecordBytes = 0
			case "record-large":
				c.MaxRecordBytes = 128<<10 + 1
			case "stored-zero":
				c.MaxStoredBytes = 0
			case "stored-large":
				c.MaxStoredBytes = 256<<20 + 1
			case "record-over-stored":
				c.MaxStoredBytes = c.MaxRecordBytes - 1
			}
			if a, err := OpenCandidateRetentionLog(c); err != ErrConfiguration || a != nil {
				t.Fatal(err)
			}
		})
	}
	c := retentionTestConfig(t)
	missing := filepath.Join(c.Directory, "not-created")
	c.Directory = missing
	if a, err := OpenCandidateRetentionLog(c); err == nil || a != nil {
		t.Fatal("created missing store")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("created directory")
	}
	c.Directory = t.TempDir()
	if a, err := OpenCandidateRetentionLog(c); err == nil || a != nil {
		t.Fatal("inferred absent authority")
	}
	if _, err := os.Stat(filepath.Join(c.Directory, "ANCHOR")); !os.IsNotExist(err) {
		t.Fatal("provisioned anchor")
	}
	if owner, pending, err := RecoverCandidateRetentionEnrollment(retentionTestConfig(t), enrollmentTestContext(t), "0x"+strings.Repeat("8", 64)); err == nil || owner != nil || pending != nil {
		t.Fatal("recovered absent entry")
	}
}

func TestCandidateRetentionOpaqueAndOwnerRefusals(t *testing.T) {
	var log *CandidateRetentionLog
	var read *CandidateRetentionRead
	var pending *CandidateRetentionPending
	if log.Close() != nil {
		t.Fatal("nil close")
	}
	if _, err := log.ReadInstance(""); err == nil {
		t.Fatal("nil log")
	}
	if _, err := log.EnrollInstance(nil, nil, ""); err == nil {
		t.Fatal("nil enroll")
	}
	if _, err := read.Diagnostic(); err == nil {
		t.Fatal("nil read")
	}
	if _, err := pending.Diagnostic(); err == nil {
		t.Fatal("nil pending")
	}
	for _, value := range []any{&CandidateRetentionLog{}, &CandidateRetentionRead{}, &CandidateRetentionPending{}} {
		if err := json.Unmarshal([]byte(`{}`), value); err != ErrCandidateRetention {
			t.Fatal("JSON restored opaque state")
		}
	}
	a := retentionTestOpen(t, retentionTestConfig(t))
	bad := &CandidateRetentionLog{owner: a, disk: a.disk}
	if bad.Close() != ErrCandidateRetention {
		t.Fatal("foreign close")
	}
	if _, err := bad.ReadInstance("0x" + strings.Repeat("8", 64)); err == nil {
		t.Fatal("copied owner")
	}
	if _, err := a.ReadInstance(""); err == nil {
		t.Fatal("invalid instance")
	}
	id, _ := enrollmentTestContext(t).Identity()
	r, err := a.ReadInstance(id.Instance)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"nil-read", "operation", "nil-candidate", "namespace", "instance", "generation"} {
		t.Run(mode, func(t *testing.T) {
			c, read, op := enrollmentTestContext(t), r, "0x"+strings.Repeat("8", 64)
			switch mode {
			case "nil-read":
				read = nil
			case "operation":
				op = ""
			case "nil-candidate":
				c = nil
			case "namespace":
				c.identity.Namespace = "0x" + strings.Repeat("9", 64)
			case "instance":
				c.identity.Instance = "0x" + strings.Repeat("9", 64)
			case "generation":
				c.identity.Generation = 1
			}
			if _, err := a.EnrollInstance(read, c, op); err == nil {
				t.Fatal("invalid enrollment")
			}
		})
	}
	if _, err := (&CandidateRetentionRead{owner: bad}).Diagnostic(); err == nil {
		t.Fatal("invalid snapshot owner")
	}
	if _, err := (&CandidateRetentionPending{owner: bad, root: "x"}).Diagnostic(); err == nil {
		t.Fatal("invalid pending owner")
	}
	if _, err := (&CandidateRetentionPending{owner: a}).Diagnostic(); err == nil {
		t.Fatal("missing pending root")
	}
}

func TestCandidateRetentionConcurrentLocalEnrollment(t *testing.T) {
	a := retentionTestOpen(t, retentionTestConfig(t))
	c := enrollmentTestContext(t)
	id, _ := c.Identity()
	read, err := a.ReadInstance(id.Instance)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := a.EnrollInstance(read, c, "0x"+strings.Repeat("8", 64))
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal("local comparison winners", success)
	}
}

func TestCandidateRetentionSyncFailureFencesWithoutACK(t *testing.T) {
	for _, which := range []int{1, 2} {
		t.Run(string(rune('0'+which)), func(t *testing.T) {
			config := retentionTestConfig(t)
			a := retentionTestOpen(t, config)
			calls := 0
			a.disk.syncFile = func(f *os.File) error {
				calls++
				if calls == which {
					return errors.New("inert injected sync error")
				}
				return f.Sync()
			}
			c := enrollmentTestContext(t)
			id, _ := c.Identity()
			read, err := a.ReadInstance(id.Instance)
			if err != nil {
				t.Fatal(err)
			}
			if pending, err := a.EnrollInstance(read, c, "0x"+strings.Repeat("8", 64)); err == nil || pending != nil {
				t.Fatal("uncertain write returned acknowledgement")
			}
			if _, err := a.ReadInstance(id.Instance); err == nil {
				t.Fatal("uncertain owner not fenced")
			}
			if err := a.Close(); err != nil {
				t.Fatal(err)
			}
			if owner, err := OpenCandidateRetentionLog(config); err == nil || owner != nil {
				t.Fatal("uncertain append inferred as old prefix")
			}
		})
	}
}
