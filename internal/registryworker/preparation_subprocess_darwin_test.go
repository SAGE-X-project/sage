//go:build darwin && registry_native_limits

package registryworker

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// This test runs only in the test-owned subprocess below. Instrumented test
// counters are separate from the fixed production child's runtime result.
func TestPreparationInheritedPipeProcess(t *testing.T) {
	scenario := os.Getenv("SAGE_PREPARATION_PROCESS_TEST")
	if scenario == "" {
		return
	}
	ctx := context.Background()
	if scenario == "nil-context" {
		ctx = nil
	}
	if scenario == "closed-descriptor" {
		// Only the descriptor explicitly inherited by this owned test child.
		if err := syscall.Close(5); err != nil {
			t.Fatal(err)
		}
	}
	lifetime, err := OpenPreparationLifetime(ctx)
	if scenario != "close-live-parent" && scenario != "parent-cancel" {
		if err == nil || lifetime != nil {
			t.Fatal("invalid inherited input accepted")
		}
		if _, again := OpenPreparationLifetime(context.Background()); !errors.Is(again, ErrSpent) {
			t.Fatal("failed descriptor claim was refunded")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := lifetime.Snapshot()
	if err != nil || !snapshot.valid() {
		t.Fatal("original diagnostic window missing")
	}
	action, err := lifetime.Context()
	if err != nil {
		t.Fatal(err)
	}
	cleanup, err := lifetime.CleanupContext()
	if err != nil {
		t.Fatal(err)
	}
	parentDone, err := lifetime.ParentDone()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(os.Stdout, "PREPARATION_OPEN\n"); err != nil {
		t.Fatal(err)
	}
	if scenario == "parent-cancel" {
		<-action.Done()
		if !errors.Is(action.Err(), context.Canceled) {
			t.Fatal("pipe cancellation was replaced by action deadline")
		}
		<-parentDone
	}
	if err := lifetime.ReleaseControl(); err != nil {
		t.Fatal(err)
	}
	if cleanup.Err() != nil {
		t.Fatal("control release erased the held original cleanup window")
	}
	if _, err := lifetime.Snapshot(); err == nil {
		t.Fatal("released snapshot remained available")
	}
	if _, err := lifetime.Context(); err == nil {
		t.Fatal("released action remained available")
	}
	if _, err := lifetime.ParentDone(); err == nil {
		t.Fatal("released control remained available")
	}
	if _, err := lifetime.CleanupContext(); err == nil {
		t.Fatal("released object recreated a cleanup context")
	}
	if err := lifetime.Close(); err != nil || !errors.Is(cleanup.Err(), context.Canceled) {
		t.Fatalf("final cleanup failed: %v", err)
	}
	if err := lifetime.Close(); err != nil {
		t.Fatal("idempotent final close failed")
	}
	if _, again := OpenPreparationLifetime(context.Background()); !errors.Is(again, ErrSpent) {
		t.Fatal("successful descriptor claim was refunded")
	}
}

func TestPreparationInheritedPipeLifecycle(t *testing.T) {
	for _, scenario := range []string{"close-live-parent", "parent-cancel", "nil-context", "bad-environment", "bad-header", "empty-header", "expired-action", "regular-file", "missing-descriptor", "closed-descriptor"} {
		t.Run(scenario, func(t *testing.T) {
			origin := time.Now()
			snapshot := PreparationSnapshot{Origin: origin, ActionDeadline: origin.Add(5 * time.Second),
				HardStopDeadline: origin.Add(6 * time.Second), WholeDeadline: origin.Add(7 * time.Second),
				MaximumInputBytes: 1024, MaximumResultBytes: 2048}
			binding := preparationTestBinding()
			if scenario == "expired-action" {
				snapshot.Origin = origin.Add(-3 * time.Second)
				snapshot.ActionDeadline = origin.Add(-2 * time.Second)
			}
			read, write, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = read.Close() }()
			defer func() { _ = write.Close() }()
			header := encodePreparationHeader(snapshot, binding)
			if scenario == "bad-header" {
				header = "invalid\n"
			}
			if scenario == "empty-header" {
				if err := write.Close(); err != nil {
					t.Fatal(err)
				}
			} else if _, err := io.WriteString(write, header); err != nil {
				t.Fatal(err)
			}
			regular, err := os.OpenFile(filepath.Join(t.TempDir(), "inherited"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = regular.Close() }()
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			arguments := []string{"-test.run=^TestPreparationInheritedPipeProcess$", "-test.timeout=7s"}
			if testing.CoverMode() != "" {
				directory := t.TempDir()
				if root, ok := os.LookupEnv("SAGE_NATIVE_PRIMITIVE_EVIDENCE_DIR"); ok {
					directory = filepath.Join(root, "worker", "child-profiles", scenario)
					if err := os.MkdirAll(directory, 0o700); err != nil {
						t.Fatal(err)
					}
				}
				arguments = append(arguments, "-test.gocoverdir="+directory, "-test.coverprofile="+filepath.Join(directory, "coverage.out"))
			}
			cmd := exec.CommandContext(ctx, os.Args[0], arguments...)
			environment := binding.environment()
			if scenario == "bad-environment" {
				environment = "invalid"
			}
			cmd.Env = []string{"LANG=C", "LC_ALL=C", "SAGE_PREPARATION_PROCESS_TEST=" + scenario, journalEnvironment + "=" + environment}
			cmd.ExtraFiles = []*os.File{regular, regular, read}
			if scenario == "regular-file" {
				cmd.ExtraFiles[2] = regular
			} else if scenario == "missing-descriptor" {
				cmd.ExtraFiles = cmd.ExtraFiles[:2]
			}
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			if scenario == "close-live-parent" || scenario == "parent-cancel" {
				marker, err := bufio.NewReader(stdout).ReadString('\n')
				if err != nil || marker != "PREPARATION_OPEN\n" {
					_ = write.Close()
					_ = cmd.Wait()
					t.Fatalf("child did not open actual inherited pipe: %q %v", marker, err)
				}
				if scenario == "parent-cancel" {
					if err := write.Close(); err != nil {
						t.Fatal(err)
					}
				}
			}
			// In the normal case the parent writer stays OPEN through child
			// ReleaseControl and exit. Closing it first would conceal F1.
			if err := cmd.Wait(); err != nil || ctx.Err() != nil {
				t.Fatalf("owned pipe lifecycle failed: err=%v deadline=%v", err, ctx.Err())
			}
		})
	}
}
