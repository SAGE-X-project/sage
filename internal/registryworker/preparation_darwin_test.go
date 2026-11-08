//go:build darwin && registry_native_limits

package registryworker

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry011"
)

// The existing CI builds this fixed production entry before the runtime tests.
// An absent build is explicit, rather than replacing the compiler with echo.
func fixedPreparationAsset(t *testing.T) Asset {
	t.Helper()
	path, ok := os.LookupEnv("SAGE_REGISTRY_PREPARATION_WORKER")
	if !ok {
		t.Skip("fixed production preparation worker was not supplied by the native CI build")
	}
	if !filepath.IsAbs(path) {
		t.Fatal("fixed production worker path must be absolute")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 128<<20 {
		t.Fatalf("fixed production worker unavailable: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || int64(len(raw)) != info.Size() {
		t.Fatalf("fixed production worker read failed: %v", err)
	}
	digest := sha256.Sum256(raw)
	return Asset{Path: path, Bytes: info.Size(), SHA256: hex.EncodeToString(digest[:])}
}

func TestPreparationFixedChildRetainsOriginalRefusalJournal(t *testing.T) {
	c := nativeConfiguration(t)
	c.Worker = fixedPreparationAsset(t)
	c.RequireJournal, c.RequirePreparation = true, true
	c.TerminalEvidence = EvidenceConfiguration{Directory: privateTerminalDirectory(t), MaximumBytes: journalMaximumBytes}
	owner, err := New(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	result, err := owner.Run([]byte(`{}`))
	if err == nil || !result.Started || !result.LimitsApplied || result.ExitCode == 0 ||
		result.Output != nil || !result.CleanupComplete || !result.TerminalEvidence.ReadbackComplete ||
		!result.Journal.Bound || !result.Journal.Started || !result.Journal.Terminal ||
		result.Journal.Outcome != "failed" || result.Journal.Failure != "compilation" {
		t.Fatalf("fixed child refusal or parent readback missing: result=%+v err=%v", result, err)
	}
	stored, err := os.ReadFile(result.TerminalEvidence.Path)
	if err != nil || !bytes.Equal(stored, result.TerminalEvidence.Prefix) {
		t.Fatalf("retained refusal bytes differ: %v", err)
	}
	assertNoPrivateCopies(t, c.Directory)
}

func TestPreparationFixedChildCompilesBoundedOriginalInput(t *testing.T) {
	path, ok := os.LookupEnv("SAGE_HISTORY_PREPARATION_INPUT")
	if !ok {
		t.Skip("bounded original preparation input was not supplied by CI")
	}
	input, err := os.ReadFile(path)
	if err != nil || len(input) == 0 || len(input) > 64<<20 {
		t.Fatalf("bounded preparation input missing: %v", err)
	}
	reference, err := os.ReadFile(path+".expected.json")
	if err != nil {
		t.Fatal(err)
	}
	var expected struct {
		Identity registry011.ProviderCandidateIdentity `json:"identity"`
		ManifestSHA256 string `json:"manifest_sha256"`
	}
	if err := json.Unmarshal(reference, &expected); err != nil {
		t.Fatal(err)
	}
	c := nativeConfiguration(t)
	c.Worker = fixedPreparationAsset(t)
	c.RequireJournal, c.RequirePreparation = true, true
	// Explicit bounds for this small, test-owned preparation case. These
	// neither change the operating profile nor qualify its full resource set.
	c.Limits.InputBytes, c.Limits.ResultBytes = len(input), 64<<10
	c.Limits.Duration, c.Limits.CleanupReserve, c.Limits.ChildCleanupReserve = 15*time.Second, 2*time.Second, time.Second
	c.TerminalEvidence = EvidenceConfiguration{Directory: privateTerminalDirectory(t), MaximumBytes: journalMaximumBytes}
	owner, err := New(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	result, err := owner.Run(input)
	if err != nil || !result.Started || !result.LimitsApplied || result.ExitCode != 0 || !result.CleanupComplete ||
		!result.TerminalEvidence.ReadbackComplete || !result.Journal.Bound || !result.Journal.Terminal || result.Journal.Outcome != "completed" {
		t.Fatalf("actual fixed preparation did not complete: result=%+v err=%v", result, err)
	}
	summary, err := registry011.DecodeHistoryPreparationSummary(context.Background(), result.Output, input, c.Limits.ResultBytes)
	if err != nil || summary.SourceReady || summary.Identity != expected.Identity || summary.ManifestSHA256 != expected.ManifestSHA256 ||
		summary.OriginUnixNano != result.PreparationWindow.Origin.UnixNano() ||
		summary.DeadlineUnixNano != result.PreparationWindow.ActionDeadline.UnixNano() ||
		!time.Now().Before(result.PreparationWindow.ActionDeadline) {
		t.Fatalf("actual preparation did not preserve original DATA/window: %v", err)
	}
	digest := sha256.Sum256(result.Output)
	if result.Journal.OutputSHA256 != hex.EncodeToString(digest[:]) {
		t.Fatal("actual terminal output digest differs")
	}
	stored, err := os.ReadFile(result.TerminalEvidence.Path)
	if err != nil || !bytes.Equal(stored, result.TerminalEvidence.Prefix) {
		t.Fatalf("actual terminal readback differs: %v", err)
	}
	assertNoPrivateCopies(t, c.Directory)
}

func TestPreparationFixedChildStopsAtOriginalBoundaryOnParentLoss(t *testing.T) {
	asset := fixedPreparationAsset(t)
	// These are explicit, small test lifetimes for our own child. They neither
	// select the whole-history operating profile nor qualify native resources.
	origin := time.Now()
	action, cancelAction := context.WithDeadline(context.Background(), origin.Add(5*time.Second))
	hardStop, cancelHardStop := context.WithDeadline(context.Background(), origin.Add(6*time.Second))
	whole, cancelWhole := context.WithDeadline(context.Background(), origin.Add(7*time.Second))
	defer cancelAction()
	defer cancelHardStop()
	defer cancelWhole()
	owner := &Owner{origin: origin, action: action, hardStop: hardStop, whole: whole,
		configuration: Configuration{Limits: Limits{InputBytes: 1024, ResultBytes: 2048}}}
	owner.self = owner
	binding := newJournalBinding([]byte(`{}`), asset.SHA256)
	control, err := newPreparationControl(owner, binding)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = control.close() }()
	readyRead, readyWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = readyRead.Close() }()
	defer func() { _ = readyWrite.Close() }()
	journal, err := os.OpenFile(filepath.Join(t.TempDir(), "journal"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = journal.Close() }()
	outer, cancelOuter := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelOuter()
	cmd := exec.CommandContext(outer, asset.Path, "--sage-registry-history-worker")
	cmd.Env = []string{"LANG=C", "LC_ALL=C", journalEnvironment + "=" + binding.environment()}
	cmd.ExtraFiles = []*os.File{readyWrite, journal, control.read}
	// Keep only this test-owned stdin writer open. Even blocked input cannot
	// extend the original cleanup boundary after the live parent pipe closes.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stdin.Close() }()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	control.releaseRead()
	marker, err := bufio.NewReader(stderr).ReadString('\n')
	if err != nil || marker != "sage-registry-preparation-input-ready\n" {
		_ = stdin.Close()
		_ = cmd.Wait()
		t.Fatalf("compiler-before-input boundary was not reached: %q err=%v", marker, err)
	}
	cancelAction()
	err = cmd.Wait()
	if err == nil || outer.Err() != nil || cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != 125 {
		t.Fatalf("child did not enforce its original hard stop: err=%v state=%v", err, cmd.ProcessState)
	}
}
