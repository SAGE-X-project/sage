//go:build linux || darwin

package registry011

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Only the normal owned test process is started. It appends a candidate record
// and exits without Close, establishing OS-lock release on ordinary process
// exit. Host tampering, hostile writers and operating-store qualification are
// outside this runtime test; refusal fixtures remain in finite unit tests.
func TestCandidateRetentionProcessExit(t *testing.T) {
	cfg := retentionTestConfig(t)
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, executable, "-test.run=^TestCandidateRetentionOwnedChild$", "-test.count=1")
	child.Env = append(os.Environ(), "SAGE_RETENTION_OWNED_CHILD="+string(raw))
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("owned child failed: %v %s", err, output)
	}
	if ctx.Err() != nil {
		t.Fatal(ctx.Err())
	}
	if owner, err := OpenCandidateRetentionLog(cfg); err == nil || owner != nil {
		t.Fatal("old pin reopened as unchanged")
	}
	a, pending, err := RecoverCandidateRetentionEnrollment(cfg, enrollmentTestContext(t), "0x"+strings.Repeat("8", 64))
	if err != nil {
		t.Fatal("OS lock not released on owned process exit", err)
	}
	diagnostic, err := pending.Diagnostic()
	if err != nil || diagnostic.State != "pending-external-ack" || diagnostic.Sequence != 1 {
		t.Fatal(diagnostic, err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	cfg.ExpectedTip = diagnostic.Tip
	reopened := retentionTestOpen(t, cfg)
	id, _ := enrollmentTestContext(t).Identity()
	read, err := reopened.ReadInstance(id.Instance)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := read.Diagnostic()
	if err != nil || string(stored.Root) != string(diagnostic.Root) {
		t.Fatal("retained root changed", err)
	}
}

func TestCandidateRetentionOwnedChild(t *testing.T) {
	raw := os.Getenv("SAGE_RETENTION_OWNED_CHILD")
	if raw == "" {
		return
	}
	var cfg CandidateRetentionConfig
	if json.Unmarshal([]byte(raw), &cfg) != nil {
		t.Fatal("child fixture config")
	}
	a, err := OpenCandidateRetentionLog(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// No cleanup or Close: returning normally lets the owned child process exit
	// with the lock FD open. It is never reused as an operating authority.
	_ = retentionTestEnroll(t, a)
}
