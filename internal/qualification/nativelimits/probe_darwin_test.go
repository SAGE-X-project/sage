//go:build darwin && registry_native_limits

package nativelimits

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

const outputLimit = 64 << 10

type boundedOutput struct {
	data     bytes.Buffer
	overflow bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	n := len(p)
	left := outputLimit - b.data.Len()
	if len(p) > left {
		b.overflow = true
		p = p[:left]
	}
	_, _ = b.data.Write(p)
	return n, nil
}

type attempt struct {
	Name        string `json:"name"`
	Output      string `json:"output"`
	Error       string `json:"error,omitempty"`
	Deadline    bool   `json:"deadline"`
	Canceled    bool   `json:"canceled"`
	Started     bool   `json:"started"`
	Overflow    bool   `json:"output_overflow"`
	ElapsedNS   int64  `json:"elapsed_ns"`
	UserCPUNS   int64  `json:"user_cpu_ns"`
	SystemCPUNS int64  `json:"system_cpu_ns"`
	ExitCode    int    `json:"exit_code"`
	Signal      int    `json:"signal"`
}

// Every command uses fixed authored arguments and an owning deadline. Only
// the disposable C child lowers its own limits; the Go process is unchanged.
func invoke(owner context.Context, name, binary string, timeout time.Duration, args ...string) attempt {
	if err := owner.Err(); err != nil {
		return attempt{Name: name, Error: err.Error(), Deadline: errors.Is(err, context.DeadlineExceeded), Canceled: errors.Is(err, context.Canceled), ExitCode: -1}
	}
	ctx, cancel := context.WithTimeout(owner, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = time.Second
	var output boundedOutput
	cmd.Stdout, cmd.Stderr = &output, &output
	started := time.Now()
	err := cmd.Run()
	r := attempt{Name: name, Output: output.data.String(), Deadline: errors.Is(ctx.Err(), context.DeadlineExceeded), Canceled: errors.Is(ctx.Err(), context.Canceled), Started: cmd.Process != nil, Overflow: output.overflow, ElapsedNS: time.Since(started).Nanoseconds(), ExitCode: -1}
	if err != nil {
		r.Error = err.Error()
	}
	if state := cmd.ProcessState; state != nil {
		r.ExitCode = state.ExitCode()
		r.UserCPUNS, r.SystemCPUNS = state.UserTime().Nanoseconds(), state.SystemTime().Nanoseconds()
		if status, ok := state.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			r.Signal = int(status.Signal())
		}
	}
	return r
}

type ownedReader struct {
	owner  context.Context
	reader io.Reader
}

func (r ownedReader) Read(p []byte) (int, error) {
	if err := r.owner.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if ownerErr := r.owner.Err(); ownerErr != nil {
		return n, ownerErr
	}
	return n, err
}

func fileHash(owner context.Context, path string) (string, error) {
	if err := owner.Err(); err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	_, copyErr := io.Copy(h, ownedReader{owner: owner, reader: f})
	closeErr := f.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeJournal(owner context.Context, path string, record map[string]any) error {
	if err := owner.Err(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".observations-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	n, writeErr := f.Write(append(data, '\n'))
	if writeErr == nil && n != len(data)+1 {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = owner.Err()
	}
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := owner.Err(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func boundaryObservation(owner context.Context, path, relative string) map[string]any {
	r := map[string]any{"path": relative, "exists": false, "read_complete": false}
	if err := owner.Err(); err != nil {
		r["error"] = err.Error()
		return r
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		r["error"] = err.Error()
		return r
	}
	r["exists"] = true
	info, statErr := f.Stat()
	if statErr != nil || !info.Mode().IsRegular() {
		_ = f.Close()
		r["error"] = fmt.Sprintf("regular boundary file: %v", statErr)
		return r
	}
	r["bytes"] = info.Size()
	data, readErr := io.ReadAll(io.LimitReader(ownedReader{owner: owner, reader: f}, 8193))
	closeErr := f.Close()
	h := sha256.Sum256(data)
	if readErr != nil || closeErr != nil || len(data) > 8192 || int64(len(data)) != info.Size() {
		r["prefix_sha256"] = hex.EncodeToString(h[:])
		r["error"] = fmt.Sprintf("boundary read: read=%v close=%v retained=%d", readErr, closeErr, len(data))
		return r
	}
	r["sha256"], r["read_complete"] = hex.EncodeToString(h[:]), true
	return r
}

func TestDarwinResourcePrimitives(t *testing.T) {
	// The operation phase leaves a journal window inside this common owner,
	// with headroom before the external package timeout. Forced host loss or
	// blocked kernel I/O can still leave incomplete, unqualified observations.
	ownerDeadline := time.Now().Add(2 * time.Minute)
	if deadline, ok := t.Deadline(); ok && deadline.Add(-30*time.Second).Before(ownerDeadline) {
		ownerDeadline = deadline.Add(-30 * time.Second)
	}
	owner, cancelOwner := context.WithDeadline(context.Background(), ownerDeadline)
	t.Cleanup(cancelOwner)
	operations, cancelOperations := context.WithDeadline(owner, ownerDeadline.Add(-10*time.Second))
	t.Cleanup(cancelOperations)
	dir := t.TempDir()
	evidence := os.Getenv("SAGE_NATIVE_PRIMITIVE_EVIDENCE_DIR")
	if evidence == "" {
		evidence = filepath.Join(dir, "evidence")
	}
	if err := os.MkdirAll(evidence, 0o700); err != nil {
		t.Fatal(err)
	}
	record := map[string]any{
		"scope":                   "finite disposable-child primitive observations only",
		"whole_profile_qualified": false, "lease_issued": false,
		"witness_issued": false, "operating_authority_issued": false,
		"go_version": runtime.Version(), "go_os": runtime.GOOS, "go_arch": runtime.GOARCH,
		"collection_state":           "IN_PROGRESS",
		"owner_deadline_unix_ns":     ownerDeadline.UnixNano(),
		"operation_deadline_unix_ns": ownerDeadline.Add(-10 * time.Second).UnixNano(),
	}
	for _, name := range []string{"ImageOS", "ImageVersion", "GITHUB_SHA", "GITHUB_RUN_ID", "GITHUB_RUN_ATTEMPT", "GITHUB_JOB", "GITHUB_REF", "GITHUB_REPOSITORY"} {
		record[name] = os.Getenv(name)
	}
	var attempts []attempt
	persist := func() error {
		record["attempts"] = attempts
		return writeJournal(owner, filepath.Join(evidence, "observations.json"), record)
	}
	retainAttempt := func(r attempt) error {
		attempts = append(attempts, r)
		return persist()
	}
	t.Cleanup(func() {
		record["collection_state"], record["tests_failed"] = "RETURNED", t.Failed()
		if err := persist(); err != nil {
			t.Error(err)
		}
	})
	if err := persist(); err != nil {
		t.Fatal(err)
	}
	for _, spec := range []struct {
		name, binary string
		args         []string
	}{
		{"os_version", "/usr/bin/sw_vers", nil},
		{"kernel_release", "/usr/bin/uname", []string{"-r"}},
		{"compiler_path", "/usr/bin/xcrun", []string{"--find", "clang"}},
	} {
		r := invoke(operations, spec.name, spec.binary, 20*time.Second, spec.args...)
		if err := retainAttempt(r); err != nil {
			t.Fatal(err)
		}
		if r.Error != "" || r.Overflow || r.Deadline || r.Canceled {
			t.Fatalf("%s: %+v", spec.name, r)
		}
	}
	compiler, err := filepath.EvalSymlinks(strings.TrimSpace(attempts[len(attempts)-1].Output))
	if err != nil || !filepath.IsAbs(compiler) {
		t.Fatalf("absolute compiler path: %q: %v", compiler, err)
	}
	compilerHash, err := fileHash(operations, compiler)
	if err != nil {
		t.Fatal(err)
	}
	record["compiler_path"], record["compiler_sha256"] = compiler, compilerHash
	version := invoke(operations, "compiler_version", compiler, 20*time.Second, "--version")
	if err := retainAttempt(version); err != nil {
		t.Fatal(err)
	}
	if version.Error != "" || version.Overflow || version.Deadline || version.Canceled {
		t.Fatalf("compiler version: %+v", version)
	}
	source := filepath.Join("testdata", "resource_limits.c")
	sourceHash, err := fileHash(operations, source)
	if err != nil {
		t.Fatal(err)
	}
	record["probe_source_sha256"] = sourceHash
	binary := filepath.Join(dir, "resource-limits")
	build := invoke(operations, "compile_probe", compiler, time.Minute, "-std=c11", "-O2", "-Wall", "-Wextra", "-Werror", "-pedantic", source, "-o", binary)
	if err := retainAttempt(build); err != nil {
		t.Fatal(err)
	}
	if build.Error != "" || build.Overflow || build.Deadline || build.Canceled {
		t.Fatalf("compile probe: %+v", build)
	}
	binaryHash, err := fileHash(operations, binary)
	if err != nil {
		t.Fatal(err)
	}
	record["probe_binary_sha256"] = binaryHash
	probeFile, err := os.Open(binary)
	if err != nil {
		t.Fatal(err)
	}
	probeBytes, readErr := io.ReadAll(io.LimitReader(ownedReader{owner: operations, reader: probeFile}, (8<<20)+1))
	closeErr := probeFile.Close()
	if readErr != nil || closeErr != nil || len(probeBytes) > 8<<20 {
		t.Fatalf("retain finite probe binary: read=%v close=%v bytes=%d", readErr, closeErr, len(probeBytes))
	}
	retainedHash := sha256.Sum256(probeBytes)
	if hex.EncodeToString(retainedHash[:]) != binaryHash {
		t.Fatal("probe binary changed before retention")
	}
	if err := os.WriteFile(filepath.Join(evidence, "resource-limits.bin"), probeBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	boundaryDir, err := os.MkdirTemp(evidence, "file-boundary-")
	if err != nil {
		t.Fatal(err)
	}
	boundaryPath := filepath.Join(boundaryDir, "original.bin")
	boundaryRelative, err := filepath.Rel(evidence, boundaryPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range []struct {
		name   string
		signal syscall.Signal
	}{
		{"address-space", 0}, {"open-files", 0},
		{"file-size", syscall.SIGXFSZ}, {"cpu-time", syscall.SIGXCPU},
	} {
		t.Run(spec.name, func(t *testing.T) {
			r := invoke(operations, spec.name, binary, 20*time.Second, spec.name, boundaryPath)
			if spec.name == "file-size" {
				record["boundary_file"] = boundaryObservation(owner, boundaryPath, boundaryRelative)
			}
			if err := retainAttempt(r); err != nil {
				t.Fatal(err)
			}
			if r.Deadline || r.Canceled || r.Overflow {
				t.Fatalf("primitive did not terminate inside owned limits: %+v", r)
			}
			if spec.signal == 0 {
				if r.Error != "" || r.ExitCode != 0 || r.Signal != 0 {
					t.Fatalf("primitive refused: %+v", r)
				}
				var value struct {
					Primitive string `json:"primitive"`
					Passed    bool   `json:"passed"`
				}
				if err := json.Unmarshal([]byte(r.Output), &value); err != nil || value.Primitive != spec.name || !value.Passed {
					t.Fatalf("primitive result: %q: %v", r.Output, err)
				}
				return
			}
			marker := fmt.Sprintf("ready:%s\n", spec.name)
			if r.Error == "" || r.Signal != int(spec.signal) || r.Output != marker {
				t.Fatalf("expected owned limit signal %v: %+v", spec.signal, r)
			}
			if spec.name == "file-size" {
				boundary := record["boundary_file"].(map[string]any)
				if boundary["bytes"] != int64(4096) || boundary["read_complete"] != true || boundary["error"] != nil {
					t.Fatalf("retained file boundary: %v", boundary)
				}
			}
			if spec.name == "cpu-time" && (r.UserCPUNS+r.SystemCPUNS < int64(500*time.Millisecond) || r.UserCPUNS+r.SystemCPUNS > int64(3*time.Second)) {
				t.Fatalf("unexpected CPU accounting; no qualification: %+v", r)
			}
		})
		if err := operations.Err(); err != nil {
			record["remaining_controls_refused"] = err.Error()
			if persistErr := persist(); persistErr != nil {
				t.Error(persistErr)
			}
			t.Errorf("common operation owner expired: %v", err)
			break
		}
	}
}

func TestCommandRefusesExpiredOwnerBeforeStart(t *testing.T) {
	owner, cancel := context.WithCancel(context.Background())
	cancel()
	r := invoke(owner, "refused", "/usr/bin/true", time.Second)
	if !r.Canceled || r.Deadline || r.Started || r.ExitCode != -1 || r.Error == "" || r.Output != "" {
		t.Fatalf("owner refusal: %+v", r)
	}
	const original = "unconsumed"
	untouched := bytes.NewBufferString(original)
	reader := ownedReader{owner: owner, reader: untouched}
	if n, err := reader.Read(make([]byte, 1)); n != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("owned input refusal: n=%d err=%v", n, err)
	}
	if untouched.Len() != len(original) || untouched.String() != original {
		t.Fatalf("canceled owner consumed original input: %q", untouched.String())
	}
}

func TestJournalKeepsCompletedRecordAfterOwnerRefusal(t *testing.T) {
	dir := t.TempDir()
	owner, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	path := filepath.Join(dir, "observations.json")
	if err := writeJournal(owner, path, map[string]any{"attempts": []string{"completed"}, "collection_state": "IN_PROGRESS"}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := writeJournal(owner, path, map[string]any{"replacement": true}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expired writer: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("completed journal changed: %q: %v", after, err)
	}
}

func TestBoundaryObservationKeepsPartialAndAbsentOriginals(t *testing.T) {
	dir := t.TempDir()
	owner, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	path := filepath.Join(dir, "partial.bin")
	partial := []byte("partial")
	if err := os.WriteFile(path, partial, 0o600); err != nil {
		t.Fatal(err)
	}
	r := boundaryObservation(owner, path, "partial.bin")
	h := sha256.Sum256(partial)
	if r["bytes"] != int64(len(partial)) || r["sha256"] != hex.EncodeToString(h[:]) || r["read_complete"] != true {
		t.Fatalf("partial original observation: %v", r)
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, partial) {
		t.Fatalf("partial original changed: %q: %v", actual, err)
	}
	absent := filepath.Join(dir, "absent.bin")
	r = boundaryObservation(owner, absent, "absent.bin")
	if r["exists"] != false || r["read_complete"] != false || r["sha256"] != nil || r["error"] == nil {
		t.Fatalf("absent original observation: %v", r)
	}
	if _, err := os.Stat(absent); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("absent original was fabricated: %v", err)
	}
}

func TestBoundedOutputRetainsFinitePrefix(t *testing.T) {
	var b boundedOutput
	input := bytes.Repeat([]byte{'x'}, outputLimit+10)
	n, err := b.Write(input)
	if err != nil || n != len(input) || !b.overflow || b.data.Len() != outputLimit {
		t.Fatalf("first write: n=%d err=%v overflow=%v size=%d", n, err, b.overflow, b.data.Len())
	}
	n, err = b.Write([]byte("ignored"))
	if err != nil || n != 7 || b.data.Len() != outputLimit {
		t.Fatalf("full write: n=%d err=%v size=%d", n, err, b.data.Len())
	}
}
