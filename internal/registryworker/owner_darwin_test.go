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
	cmd, err := platformCommand(context.Background(), "/usr/bin/true", "/unused", Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Cancel(); !errors.Is(err, os.ErrProcessDone) {
		t.Fatal("completed child cancellation lost process lifetime protection", err)
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
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	o, err := New(parent, c)
	if err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(c.Directory, "readiness-"+strconv.Itoa(os.Getpid()))
	type completed struct {
		result Result
		err    error
	}
	done := make(chan completed, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		result, err := o.Run([]byte("wait " + ready))
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
		if !errors.Is(returned.err, context.Canceled) || !returned.result.Started || !returned.result.LimitsApplied || !returned.result.CleanupComplete || returned.result.Signal != int(syscall.SIGKILL) || len(returned.result.Output) != 0 {
			t.Fatalf("active cancellation lost terminal custody: %+v %v", returned.result, returned.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("canceled worker did not return")
	}
	assertNoPrivateCopies(t, c.Directory)
}
