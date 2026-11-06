//go:build darwin && registry_native_limits

package registryworker

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func nativeConfiguration(t *testing.T) Configuration {
	t.Helper()
	directory := t.TempDir()
	c := candidateConfiguration(directory)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, build := range []struct {
		name, executable string
		arguments        []string
	}{
		{"launcher", "/usr/bin/cc", []string{"-std=c11", "-Wall", "-Wextra", "-Werror", "-O2", "launcher/launcher.c"}},
		{"worker", "go", []string{"build", "-mod=readonly", "./testdata/goworker"}},
	} {
		out := filepath.Join(directory, build.name+"-built")
		args := append([]string{}, build.arguments...)
		if build.name == "worker" {
			args = []string{"build", "-mod=readonly", "-o", out, "./testdata/goworker"}
		} else {
			args = append(args, "-o", out)
		}
		cmd := exec.CommandContext(ctx, build.executable, args...)
		// Trusted CI preparation only: default cancellation owns this compiler
		// driver, not its descendants. Runner teardown bounds failed preparation;
		// it is not a qualified compiler-family cleanup mechanism.
		cmd.WaitDelay = time.Second
		output := limitedOutput{limit: 64 << 10, cancel: cancel}
		cmd.Stdout, cmd.Stderr = &output, &output
		if err := cmd.Run(); err != nil {
			t.Fatalf("owned compiler failed: %v %s", err, output.data.Bytes())
		}
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		asset := fileAsset(t, filepath.Join(directory, build.name+"-pinned"), data)
		if build.name == "launcher" {
			c.Launcher = asset
		} else {
			c.Worker = asset
		}
	}
	return c
}

func TestNativeCommandCancellationAfterWaitKeepsProcessLifetime(t *testing.T) {
	for _, reserve := range []time.Duration{0, time.Second} {
		cmd, err := platformCommand(context.Background(), "/usr/bin/true", "/unused", Limits{ChildCleanupReserve: reserve})
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Cancel(); !errors.Is(err, os.ErrProcessDone) {
			t.Fatal("completed child cancellation lost process lifetime protection", reserve, err)
		}
	}
}

func TestNativeOwnerDoesNotCertifyCleanupAfterItsOriginalBoundary(t *testing.T) {
	c := nativeConfiguration(t)
	o, err := New(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	// Synthetic unit boundary around a safe real child; this does not qualify
	// the host clock or reproduce a blocked filesystem operation.
	o.whole = &expiryAfterRemoval{Context: o.whole,
		pattern: filepath.Join(c.Directory, ".sage-registry-worker-*"), done: make(chan struct{})}
	result, err := o.Run([]byte("echo"))
	if !errors.Is(err, context.DeadlineExceeded) || result.CleanupComplete || len(result.Output) != 0 || !result.Started || !result.LimitsApplied {
		t.Fatal("late cleanup certified completed output", result, err)
	}
	assertNoPrivateCopies(t, c.Directory)
	if _, err := o.Run([]byte("echo")); !errors.Is(err, ErrSpent) {
		t.Fatal("late cleanup restored an attempt", err)
	}
}

func assertNoPrivateCopies(t *testing.T, directory string) {
	t.Helper()
	entries, err := filepath.Glob(filepath.Join(directory, ".sage-registry-worker-*"))
	if err != nil || len(entries) != 0 {
		t.Fatal("private copies survived the terminal path", entries, err)
	}
}

func TestNativeOwnerRunsOnceWithAuthenticLimitReadback(t *testing.T) {
	c := nativeConfiguration(t)
	o, err := New(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	result, err := o.Run([]byte("echo"))
	if err != nil || !result.Started || !result.LimitsApplied || !result.CleanupComplete || result.ExitCode != 0 || result.Signal != 0 || string(result.Output) != "completed" || result.OutputExceeded {
		t.Fatalf("one owning invocation did not complete: %+v %v", result, err)
	}
	assertNoPrivateCopies(t, c.Directory)
	if _, err := o.Run([]byte("echo")); !errors.Is(err, ErrSpent) {
		t.Fatal("completed owner reused", err)
	}
	// Later mutation of the caller's configuration cannot change the spent
	// owner's fixed native scope or restore an execution attempt.
	c.Limits.CPUSeconds++
	if o.configuration.Limits.CPUSeconds == c.Limits.CPUSeconds {
		t.Fatal("owner retained mutable configuration")
	}
}

func TestNativeOwnerWithholdsOutputAfterEveryTerminalRefusal(t *testing.T) {
	base := nativeConfiguration(t)
	for _, test := range []struct {
		name  string
		input string
		edit  func(*Configuration)
	}{
		{"empty-input", "", func(*Configuration) {}},
		{"too-large-input", "echo", func(c *Configuration) { c.Limits.InputBytes = 3 }},
		{"wrong-launcher-hash", "echo", func(c *Configuration) { c.Launcher.SHA256 = string(bytes.Repeat([]byte("0"), 64)) }},
		{"wrong-worker-hash", "echo", func(c *Configuration) { c.Worker.SHA256 = string(bytes.Repeat([]byte("0"), 64)) }},
		{"missing-directory", "echo", func(c *Configuration) { c.Directory = filepath.Join(c.Directory, "absent") }},
		{"missing-terminal-store", "echo", func(c *Configuration) {
			c.TerminalEvidence = EvidenceConfiguration{Directory: filepath.Join(c.Directory, "absent"), MaximumBytes: 4096}
		}},
		{"output-overflow", "overflow", func(c *Configuration) { c.Limits.ResultBytes = 4 }},
		{"worker-refusal", "unknown", func(*Configuration) {}},
		{"missing-authentic-readback", "echo", func(c *Configuration) { c.Launcher = c.Worker }},
		{"invalid-executable", "echo", func(c *Configuration) {
			c.Launcher = fileAsset(t, filepath.Join(c.Directory, "invalid-executable"), []byte("not an executable"))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := base
			test.edit(&c)
			o, err := New(context.Background(), c)
			if err != nil {
				t.Fatal(err)
			}
			result, err := o.Run([]byte(test.input))
			if err == nil || len(result.Output) != 0 {
				t.Fatal("terminal refusal published output", result, err)
			}
			assertNoPrivateCopies(t, base.Directory)
			if _, err := o.Run([]byte("echo")); !errors.Is(err, ErrSpent) {
				t.Fatal("refused attempt restored", err)
			}
		})
	}
}

func TestNativeOwnerWithholdsDataWhenCleanupCannotComplete(t *testing.T) {
	c := nativeConfiguration(t)
	o, err := New(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	result, err := o.Run([]byte("unexpected-file"))
	if err == nil || !result.Started || !result.LimitsApplied || result.CleanupComplete || len(result.Output) != 0 {
		t.Fatal("failed cleanup published completed data", result, err)
	}
	entries, err := filepath.Glob(filepath.Join(c.Directory, ".sage-registry-worker-*"))
	if err != nil || len(entries) != 1 {
		t.Fatal("failed directory was silently removed", entries, err)
	}
	retained := filepath.Join(entries[0], "unexpected")
	if data, readErr := os.ReadFile(retained); readErr != nil || string(data) != "preserved" {
		t.Fatal("unexpected original was recursively removed", readErr)
	}
	for _, name := range []string{"launcher", "worker"} {
		if _, statErr := os.Stat(filepath.Join(entries[0], name)); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatal("creator's private copy was not removed", statErr)
		}
	}
	if _, err := o.Run([]byte("echo")); !errors.Is(err, ErrSpent) {
		t.Fatal("cleanup failure restored an attempt", err)
	}
}

func TestNativeOwnerCancelsAfterWorkerReadiness(t *testing.T) {
	c := nativeConfiguration(t)
	ready := filepath.Join(c.Directory, "readiness-"+strconv.Itoa(os.Getpid()))
	result := cancelReadyWorker(t, c, "wait "+ready, ready)
	if result.Signal != int(syscall.SIGKILL) {
		t.Fatal("immediate cancellation did not retain direct-child termination", result)
	}
}

func TestNativeOwnerLetsTheChildRecordItsCooperativeStop(t *testing.T) {
	c := nativeConfiguration(t)
	c.Limits.CleanupReserve = 3 * time.Second
	c.Limits.ChildCleanupReserve = time.Second
	ready, terminal := filepath.Join(c.Directory, "ready"), filepath.Join(c.Directory, "terminal")
	result := cancelReadyWorker(t, c, "stop "+ready+" "+terminal, ready)
	if result.ExitCode != 0 || result.Signal != 0 {
		t.Fatal("cooperative stop did not reach normal child exit", result)
	}
	if data, err := os.ReadFile(terminal); err != nil || string(data) != "closed" {
		t.Fatal("child terminal marker was not completed", err)
	}
}

func TestNativeOwnerRetainsTerminalEvidenceAfterSuccessAndFailure(t *testing.T) {
	base := nativeConfiguration(t)
	for _, test := range []struct {
		name, input, prefix string
		maximum             int
		wantError, cleanup  bool
	}{
		{"completed", "evidence-complete", "started\ncompleted\n", 4096, false, true},
		{"refused", "evidence-failure", "started\n", 4096, true, true},
		{"readback-overflow", "evidence-complete", "star", 4, true, true},
		{"empty-is-only-data", "echo", "", 4096, false, true},
		{"cleanup-failure", "evidence-cleanup-failure", "started\n", 4096, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := base
			c.Directory = t.TempDir()
			c.TerminalEvidence = EvidenceConfiguration{Directory: t.TempDir(), MaximumBytes: test.maximum}
			o, err := New(context.Background(), c)
			if err != nil {
				t.Fatal(err)
			}
			result, err := o.Run([]byte(test.input))
			if (err != nil) != test.wantError || !result.Started || !result.LimitsApplied || result.CleanupComplete != test.cleanup {
				t.Fatal("worker terminal status changed", result, err)
			}
			overflow := test.maximum == 4
			if result.TerminalEvidence.LimitExceeded != overflow || overflow && !errors.Is(err, ErrEvidenceLimit) {
				t.Fatal("terminal evidence overflow did not refuse publication", result, err)
			}
			assertTerminalPrefix(t, result.TerminalEvidence, test.prefix, !overflow)
			if test.wantError && len(result.Output) != 0 {
				t.Fatal("failure evidence allowed ordinary result publication", result)
			}
			if test.cleanup {
				assertNoPrivateCopies(t, c.Directory)
			}
			if _, err := o.Run([]byte("echo")); !errors.Is(err, ErrSpent) {
				t.Fatal("retained evidence restored a consumed attempt", err)
			}
		})
	}
}

func TestNativeOwnerRetainsTerminalEvidenceAcrossBothCancellationPaths(t *testing.T) {
	base := nativeConfiguration(t)
	for _, cooperative := range []bool{false, true} {
		name, input, expected := "immediate", "evidence-wait ", "started\n"
		c := base
		c.Directory = t.TempDir()
		c.TerminalEvidence = EvidenceConfiguration{Directory: t.TempDir(), MaximumBytes: 4096}
		if cooperative {
			name, input, expected = "cooperative", "evidence-stop ", "started\nstopped\n"
			c.Limits.CleanupReserve, c.Limits.ChildCleanupReserve = 3*time.Second, time.Second
		}
		t.Run(name, func(t *testing.T) {
			ready := filepath.Join(c.Directory, "ready")
			result := cancelReadyWorker(t, c, input+ready, ready)
			assertTerminalPrefix(t, result.TerminalEvidence, expected, true)
			if cooperative && (result.Signal != 0 || result.ExitCode != 0) ||
				!cooperative && result.Signal != int(syscall.SIGKILL) {
				t.Fatal("cancellation changed its owned termination scope", result)
			}
		})
	}
}

func TestNativeOwnerPreservesEmptyEvidenceWhenStartupFails(t *testing.T) {
	c := nativeConfiguration(t)
	c.Launcher = fileAsset(t, filepath.Join(c.Directory, "invalid-launcher"), []byte("not an executable"))
	c.TerminalEvidence = EvidenceConfiguration{Directory: t.TempDir(), MaximumBytes: 4096}
	o, err := New(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	result, err := o.Run([]byte("echo"))
	if err == nil || result.Started || result.LimitsApplied || !result.CleanupComplete || len(result.Output) != 0 {
		t.Fatal("failed startup published completed work", result, err)
	}
	assertTerminalPrefix(t, result.TerminalEvidence, "", true)
	assertNoPrivateCopies(t, c.Directory)
	if _, err := o.Run([]byte("echo")); !errors.Is(err, ErrSpent) {
		t.Fatal("failed startup restored the terminal attempt", err)
	}
}

func cancelReadyWorker(t *testing.T, c Configuration, input, ready string) Result {
	t.Helper()
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	o, err := New(parent, c)
	if err != nil {
		t.Fatal(err)
	}
	type completed struct {
		result Result
		err    error
	}
	done := make(chan completed, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		result, err := o.Run([]byte(input))
		done <- completed{result, err}
	}()
	defer func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(7 * time.Second):
			t.Error("owned worker did not finish before test cleanup")
		}
	}()
	readyDeadline := time.After(3 * time.Second)
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	observed := false
	for !observed {
		select {
		case <-tick.C:
			data, readErr := os.ReadFile(ready)
			observed = readErr == nil && string(data) == "ready"
		case returned := <-done:
			t.Fatalf("worker returned before actual readiness: %+v %v", returned.result, returned.err)
		case <-readyDeadline:
			cancel()
			t.Fatal("worker did not reach its own readiness boundary")
		}
	}
	if _, err := o.Run([]byte("echo")); !errors.Is(err, ErrSpent) || parent.Err() != nil {
		t.Fatal("overlapping invocation replaced or canceled the original", err)
	}
	cancel()
	select {
	case returned := <-done:
		if !errors.Is(returned.err, context.Canceled) || !returned.result.Started || !returned.result.LimitsApplied || !returned.result.CleanupComplete || len(returned.result.Output) != 0 {
			t.Fatalf("active cancellation lost terminal custody: %+v %v", returned.result, returned.err)
		}
		assertNoPrivateCopies(t, c.Directory)
		return returned.result
	case <-time.After(5 * time.Second):
		t.Fatal("canceled worker did not return")
	}
	return Result{}
}

func TestNativeHardStopUsesTheOwnedProcessLifetime(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// One fixed, ordinary five-second wait. No group signaling, PID reuse,
	// hostile-host scenario or unrelated process is involved.
	cmd := exec.Command("/bin/sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	stopped := watchHardStop(ctx, cmd.Process, make(chan struct{}))
	cancel()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("fixed hard-stop boundary did not terminate its child")
	}
	if err := cmd.Wait(); err == nil || platformSignal(cmd.ProcessState) != int(syscall.SIGKILL) {
		t.Fatal("owned wait did not record forced termination", err)
	}
	// The same protected handle is safe after Wait; no stored PID is reused.
	stopped = watchHardStop(ctx, cmd.Process, make(chan struct{}))
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatal("completed child was not recognized", err)
		}
	case <-time.After(time.Second):
		t.Fatal("completed child cancellation did not return")
	}
}
