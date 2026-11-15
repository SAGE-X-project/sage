//go:build darwin

package registryworker

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

func platformSupported() bool { return true }

func platformCommand(ctx context.Context, launcher, worker string, l Limits) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, launcher, worker, strconv.FormatUint(l.CPUSeconds, 10),
		strconv.FormatUint(l.FileBytes, 10), strconv.FormatUint(l.Descriptors, 10))
	// Keep CommandContext's lifecycle-protected Process.Kill cancellation.
	// The selected launcher/worker is one process, with no descendants.
	if l.ChildCleanupReserve > 0 {
		// Exec calls Cancel only after Start. Signal uses the same protected
		// Process handle; the original hard-stop owner separately bounds grace.
		cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	}
	return cmd, nil
}

func platformReadiness(data []byte, l Limits) bool {
	expected := "sage-registry-worker-v1 " + strconv.FormatUint(l.CPUSeconds, 10) + " " +
		strconv.FormatUint(l.FileBytes, 10) + " " + strconv.FormatUint(l.Descriptors, 10) + "\n"
	return bytes.Equal(data, []byte(expected))
}

func platformSignal(state *os.ProcessState) int {
	if status, ok := state.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return int(status.Signal())
	}
	return 0
}
