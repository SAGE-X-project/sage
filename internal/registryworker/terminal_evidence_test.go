//go:build darwin

package registryworker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func terminalRecord(t *testing.T, value string, maximum int) *terminalEvidence {
	t.Helper()
	e, err := openTerminalEvidence(context.Background(), EvidenceConfiguration{Directory: t.TempDir(), MaximumBytes: maximum})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.file.WriteString(value); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if !e.spent {
			_, _ = e.finish(context.Background())
		}
	})
	return e
}

func assertTerminalPrefix(t *testing.T, result EvidenceResult, value string, complete bool) {
	t.Helper()
	digest := sha256.Sum256([]byte(value))
	if string(result.Prefix) != value || result.PrefixSHA256 != hex.EncodeToString(digest[:]) || result.ReadbackComplete != complete {
		t.Fatal("retained bytes, digest or readback status changed", result)
	}
	retained, err := os.ReadFile(result.Path)
	if err != nil || len(retained) < len(value) || string(retained[:len(value)]) != value {
		t.Fatal("independent file read lost the retained prefix", err)
	}
}

func TestTerminalEvidenceRetainsExactBytesAfterTheExecutableDirectoryIsGone(t *testing.T) {
	for _, value := range []string{"", "started\n", "started\ncompleted\n"} {
		t.Run(value, func(t *testing.T) {
			e := terminalRecord(t, value, 4096)
			executables := t.TempDir()
			if err := os.Remove(executables); err != nil {
				t.Fatal(err)
			}
			result, err := e.finish(context.Background())
			if err != nil || result.ObservedBytes != int64(len(value)) || result.LimitExceeded {
				t.Fatal("ordinary readback failed", result, err)
			}
			assertTerminalPrefix(t, result, value, true)
			info, statErr := os.Stat(result.Path)
			if statErr != nil || info.Mode().Perm() != 0o600 || filepath.Dir(result.Path) == executables {
				t.Fatal("terminal file was not private and separate", statErr)
			}
			if _, statErr := e.file.Stat(); !errors.Is(statErr, os.ErrClosed) {
				t.Fatal("readback left the file handle open", statErr)
			}
			if _, statErr := e.directory.Stat(); !errors.Is(statErr, os.ErrClosed) {
				t.Fatal("readback left the directory handle open", statErr)
			}
			if _, err := e.finish(context.Background()); !errors.Is(err, ErrSpent) {
				t.Fatal("terminal readback was repeated", err)
			}
		})
	}
}

func TestTerminalEvidenceBoundsReadbackWithoutDeletingFailureBytes(t *testing.T) {
	e := terminalRecord(t, "started\nfailed\n", 4)
	result, err := e.finish(context.Background())
	if !errors.Is(err, ErrEvidenceLimit) || !result.LimitExceeded || result.ObservedBytes != 15 {
		t.Fatal("oversized evidence was accepted", result, err)
	}
	assertTerminalPrefix(t, result, "star", false)
	if data, err := os.ReadFile(result.Path); err != nil || string(data) != "started\nfailed\n" {
		t.Fatal("readback refusal deleted the original failure bytes", err)
	}
}

func TestTerminalEvidenceCannotCertifyExpiredOrUnavailableReadback(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*testing.T, *terminalEvidence) context.Context
	}{
		{"missing-lifetime", func(*testing.T, *terminalEvidence) context.Context { return nil }},
		{"expired-lifetime", func(*testing.T, *terminalEvidence) context.Context { return canceledContext() }},
		{"closed-file", func(t *testing.T, e *terminalEvidence) context.Context {
			if err := e.file.Close(); err != nil {
				t.Fatal(err)
			}
			return context.Background()
		}},
		{"closed-directory", func(t *testing.T, e *terminalEvidence) context.Context {
			if err := e.directory.Close(); err != nil {
				t.Fatal(err)
			}
			return context.Background()
		}},
		{"closed-root", func(t *testing.T, e *terminalEvidence) context.Context {
			if err := e.root.Close(); err != nil {
				t.Fatal(err)
			}
			return context.Background()
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := terminalRecord(t, "failure-prefix", 4096)
			result, err := e.finish(test.edit(t, e))
			if err == nil || result.ReadbackComplete || result.Path == "" {
				t.Fatal("unavailable cleanup certified terminal readback", result, err)
			}
			if data, err := os.ReadFile(result.Path); err != nil || string(data) != "failure-prefix" {
				t.Fatal("failed readback removed the existing bytes", err)
			}
		})
	}
}

func TestTerminalEvidenceCreationRefusesExpiredActionAndUnavailableRoot(t *testing.T) {
	for _, unavailableRoot := range []bool{false, true} {
		path := t.TempDir()
		root, err := os.OpenRoot(path)
		if err != nil {
			t.Fatal(err)
		}
		directory, err := root.Open(".")
		if err != nil {
			_ = root.Close()
			t.Fatal(err)
		}
		ctx := canceledContext()
		if unavailableRoot {
			ctx = context.Background()
			if err := root.Close(); err != nil {
				t.Fatal(err)
			}
		}
		record, err := createTerminalEvidenceInRoot(ctx, EvidenceConfiguration{Directory: path, MaximumBytes: 4096}, root, directory)
		closeErr := directory.Close()
		_ = root.Close()
		if record != nil || err == nil || closeErr != nil {
			t.Fatal("unavailable owner created a terminal file", err, closeErr)
		}
		if entries, err := os.ReadDir(path); err != nil || len(entries) != 0 {
			t.Fatal("refused creation changed its store", entries, err)
		}
	}
}

func TestTerminalEvidenceKeepsCreationAndSyncOnTheOpenedRoot(t *testing.T) {
	for _, moveBeforeCreation := range []bool{true, false} {
		name := "move-after-creation"
		if moveBeforeCreation {
			name = "move-before-creation"
		}
		t.Run(name, func(t *testing.T) {
			parent := t.TempDir()
			original, moved := filepath.Join(parent, "original"), filepath.Join(parent, "moved")
			if err := os.Mkdir(original, 0o700); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(original)
			if err != nil {
				t.Fatal(err)
			}
			directory, err := root.Open(".")
			if err != nil {
				_ = root.Close()
				t.Fatal(err)
			}
			move := func() {
				if err := os.Rename(original, moved); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(original, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if moveBeforeCreation {
				move()
			}
			c := EvidenceConfiguration{Directory: original, MaximumBytes: 4096}
			record, err := createTerminalEvidenceInRoot(context.Background(), c, root, directory)
			if err != nil {
				_ = directory.Close()
				_ = root.Close()
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if !record.spent {
					_, _ = record.finish(context.Background())
				}
			})
			if _, err := record.file.WriteString("owned-prefix"); err != nil {
				t.Fatal(err)
			}
			if !moveBeforeCreation {
				move()
			}
			result, err := record.finish(context.Background())
			if !errors.Is(err, ErrUnavailable) || result.ReadbackComplete || string(result.Prefix) != "owned-prefix" {
				t.Fatal("moved name certified a different retention parent", result, err)
			}
			if entries, err := os.ReadDir(original); err != nil || len(entries) != 0 {
				t.Fatal("creation escaped the original open root", entries, err)
			}
			retained := filepath.Join(moved, filepath.Base(result.Path))
			if data, err := os.ReadFile(retained); err != nil || string(data) != "owned-prefix" {
				t.Fatal("namespace change removed the original retained bytes", err)
			}
		})
	}
}

// A deterministic unit boundary: the action expires only when terminal
// cleanup has closed the owned file. This is not host-clock qualification.
type actionExpiryAfterTerminalClose struct {
	context.Context
	file *os.File
}

func (c actionExpiryAfterTerminalClose) Err() error {
	if _, err := c.file.Stat(); errors.Is(err, os.ErrClosed) {
		return context.DeadlineExceeded
	}
	return c.Context.Err()
}

func TestTerminalCleanupDoesNotRetroactivelyExpireACompletedAction(t *testing.T) {
	e := terminalRecord(t, "completed", 4096)
	action := actionExpiryAfterTerminalClose{Context: context.Background(), file: e.file}
	result, err := finishWorkerAction(action, context.Background(), e)
	if err != nil || !errors.Is(action.Err(), context.DeadlineExceeded) {
		t.Fatal("cleanup-only action expiry invalidated completed work", result, err)
	}
	assertTerminalPrefix(t, result, "completed", true)
	for _, expiredWhole := range []bool{false, true} {
		e := terminalRecord(t, "prefix", 4096)
		action, whole := canceledContext(), context.Background()
		if expiredWhole {
			action, whole = context.Background(), canceledContext()
		}
		result, err := finishWorkerAction(action, whole, e)
		wantComplete := !expiredWhole
		if !errors.Is(err, context.Canceled) || result.ReadbackComplete != wantComplete {
			t.Fatal("original action or whole expiry was ignored", result, err)
		}
	}
}

func TestTerminalEvidenceRequiresAnExistingPrivateStore(t *testing.T) {
	for _, test := range []struct {
		name string
		path func(*testing.T) string
	}{
		{"absent", func(t *testing.T) string { return filepath.Join(t.TempDir(), "absent") }},
		{"ordinary-file", func(t *testing.T) string {
			return fileAsset(t, filepath.Join(t.TempDir(), "file"), []byte("owned")).Path
		}},
		{"public-directory", func(t *testing.T) string {
			path := t.TempDir()
			if err := os.Chmod(path, 0o755); err != nil {
				t.Fatal(err)
			}
			return path
		}},
		{"linked-directory", func(t *testing.T) string {
			path := filepath.Join(t.TempDir(), "link")
			if err := os.Symlink(t.TempDir(), path); err != nil {
				t.Fatal(err)
			}
			return path
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := EvidenceConfiguration{Directory: test.path(t), MaximumBytes: 4096}
			if record, err := openTerminalEvidence(context.Background(), c); record != nil || !errors.Is(err, ErrUnavailable) {
				t.Fatal("unqualified namespace opened a terminal channel", err)
			}
		})
	}
	c := EvidenceConfiguration{Directory: t.TempDir(), MaximumBytes: 4096}
	for _, ctx := range []context.Context{nil, canceledContext()} {
		if record, err := openTerminalEvidence(ctx, c); record != nil || !errors.Is(err, ErrUnavailable) {
			t.Fatal("terminal store opened outside its original action", err)
		}
	}
	if record, err := openTerminalEvidence(context.Background(), EvidenceConfiguration{}); record != nil || err != nil {
		t.Fatal("disabled terminal channel changed existing worker behavior", err)
	}
	var absent *terminalEvidence
	if result, err := absent.finish(context.Background()); result.Path != "" || result.ReadbackComplete || err != nil {
		t.Fatal("disabled terminal channel claimed a readback", err)
	}
}
