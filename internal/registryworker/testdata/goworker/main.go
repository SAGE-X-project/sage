//go:build darwin

// Safe finite worker fixture. Its stock Go runtime starts after the real C
// launcher's limits were installed. This is no Registry compiler or issuer.
package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "--sage-registry-history-worker" {
		os.Exit(2)
	}
	for _, expected := range []struct {
		resource int
		value    uint64
	}{{syscall.RLIMIT_CPU, 10}, {syscall.RLIMIT_FSIZE, 1 << 20}, {syscall.RLIMIT_NOFILE, 64}, {syscall.RLIMIT_CORE, 0}} {
		var actual syscall.Rlimit
		if syscall.Getrlimit(expected.resource, &actual) != nil || actual.Cur != expected.value || actual.Max != expected.value {
			os.Exit(3)
		}
	}
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 4096))
	if err != nil || len(raw) == 4096 {
		os.Exit(4)
	}
	input := string(raw)
	switch input {
	case "echo":
		if _, err := io.WriteString(os.Stdout, "completed"); err != nil {
			os.Exit(5)
		}
	case "overflow":
		if _, err := io.WriteString(os.Stdout, strings.Repeat("0123456789", 4)); err != nil {
			os.Exit(5)
		}
	case "unexpected-file":
		if err := os.WriteFile("unexpected", []byte("preserved"), 0o600); err != nil {
			os.Exit(5)
		}
		if _, err := io.WriteString(os.Stdout, "completed"); err != nil {
			os.Exit(5)
		}
	default:
		if strings.HasPrefix(input, "evidence-") {
			terminalEvidence(input)
			return
		}
		if strings.HasPrefix(input, "stop ") {
			cooperativeStop(input)
			return
		}
		if !strings.HasPrefix(input, "wait /") {
			os.Exit(6)
		}
		f, err := os.OpenFile(strings.TrimPrefix(input, "wait "), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			os.Exit(7)
		}
		_, writeErr := f.WriteString("ready")
		if closeErr := f.Close(); writeErr != nil || closeErr != nil {
			os.Exit(8)
		}
		// An owned cancellation must end this ordinary Go wait. The finite
		// fallback also bounds fixture lifetime if its parent disappears.
		<-time.After(5 * time.Second)
		os.Exit(9)
	}
}

// The inherited descriptor addresses a separate parent-owned test file. Only
// fixed finite marker strings are written; these are not Registry journals.
func terminalEvidence(input string) {
	f := os.NewFile(4, "parent-terminal-evidence")
	if f == nil {
		os.Exit(10)
	}
	defer func() {
		if f.Close() != nil {
			os.Exit(11)
		}
	}()
	write := func(value string) {
		if _, err := io.WriteString(f, value); err != nil || f.Sync() != nil {
			os.Exit(11)
		}
	}
	write("started\n")
	switch input {
	case "evidence-complete":
		write("completed\n")
		if _, err := io.WriteString(os.Stdout, "completed"); err != nil {
			os.Exit(5)
		}
		return
	case "evidence-failure":
		os.Exit(12)
	case "evidence-cleanup-failure":
		if !writeMarker("unexpected", "preserved") {
			os.Exit(7)
		}
		return
	}
	fields := strings.Fields(input)
	if len(fields) != 2 || !filepath.IsAbs(fields[1]) ||
		fields[0] != "evidence-wait" && fields[0] != "evidence-stop" {
		os.Exit(6)
	}
	if fields[0] == "evidence-wait" {
		if !writeMarker(fields[1], "ready") {
			os.Exit(7)
		}
		<-time.After(5 * time.Second)
		os.Exit(9)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()
	if !writeMarker(fields[1], "ready") {
		os.Exit(7)
	}
	fallback := time.NewTimer(5 * time.Second)
	defer fallback.Stop()
	select {
	case <-ctx.Done():
	case <-fallback.C:
		os.Exit(9)
	}
	write("stopped\n")
}

// This finite test fixture writes its terminal marker only after an ordinary
// termination request. Neither marker is a Registry journal or qualification.
func cooperativeStop(input string) {
	fields := strings.Fields(input)
	if len(fields) != 3 || !filepath.IsAbs(fields[1]) || !filepath.IsAbs(fields[2]) {
		os.Exit(6)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()
	if !writeMarker(fields[1], "ready") {
		os.Exit(7)
	}
	fallback := time.NewTimer(5 * time.Second)
	defer fallback.Stop()
	select {
	case <-ctx.Done():
	case <-fallback.C:
		os.Exit(9)
	}
	if !writeMarker(fields[2], "closed") {
		os.Exit(8)
	}
	if _, err := io.WriteString(os.Stdout, "stopped"); err != nil {
		os.Exit(5)
	}
}

func writeMarker(path, value string) bool {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return false
	}
	_, writeErr := f.WriteString(value)
	closeErr := f.Close()
	return writeErr == nil && closeErr == nil
}
