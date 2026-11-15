//go:build darwin && registry_native_limits

package nativelimits

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

const capabilityOutputLimit = 4096

type capabilityOutput struct {
	prefix    []byte
	observed  uint64
	saturated bool
	overflow  bool
	cancel    context.CancelFunc
}

func (o *capabilityOutput) Write(p []byte) (int, error) {
	if uint64(len(p)) > ^uint64(0)-o.observed {
		o.observed, o.saturated = ^uint64(0), true
	} else {
		o.observed += uint64(len(p))
	}
	n := min(len(p), capabilityOutputLimit-len(o.prefix))
	o.prefix = append(o.prefix, p[:n]...)
	if n < len(p) && !o.overflow {
		o.overflow = true
		if o.cancel != nil {
			o.cancel()
		}
	}
	return len(p), nil
}

type capabilityFileIdentity struct {
	Path     string `json:"path"`
	Resolved string `json:"resolved_path"`
	Bytes    int64  `json:"bytes"`
	SHA256   string `json:"sha256"`
	Error    string `json:"error,omitempty"`
}

func capabilityIdentity(path string) capabilityFileIdentity {
	out := capabilityFileIdentity{Path: path}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.Resolved = resolved
	f, err := os.Open(resolved)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	defer f.Close()
	info, err := f.Stat()
	const limit = int64(256 << 20)
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		out.Error = "identity requires a regular file within 256 MiB"
		return out
	}
	h := sha256.New()
	out.Bytes, err = io.Copy(h, io.LimitReader(f, limit+1))
	if err != nil || out.Bytes > limit || out.Bytes != info.Size() {
		out.Error = "identity read failed or file size changed"
		return out
	}
	out.SHA256 = fmt.Sprintf("%x", h.Sum(nil))
	return out
}

type capabilityBuildIdentity struct {
	Source                capabilityFileIdentity `json:"source"`
	Binary                capabilityFileIdentity `json:"binary"`
	Compiler              capabilityFileIdentity `json:"compiler"`
	CompilerVersionRecord string                 `json:"compiler_version_record"`
}

type capabilityOutputRecord struct {
	ObservedBytes uint64 `json:"observed_bytes"`
	Saturated     bool   `json:"observed_bytes_saturated"`
	RetainedBytes int    `json:"retained_bytes"`
	Overflow      bool   `json:"overflow"`
	SHA256        string `json:"retained_sha256"`
}

func (o *capabilityOutput) record() capabilityOutputRecord {
	return capabilityOutputRecord{o.observed, o.saturated, len(o.prefix), o.overflow, fmt.Sprintf("%x", sha256.Sum256(o.prefix))}
}

type capabilityAttempt struct {
	Schema        int                     `json:"schema"`
	Arguments     []string                `json:"arguments"`
	Build         capabilityBuildIdentity `json:"observed_build_identity"`
	Started       bool                    `json:"started"`
	ProcessState  bool                    `json:"process_state_available"`
	Exited        bool                    `json:"exited"`
	ExitCode      int                     `json:"exit_code"`
	Signal        int                     `json:"signal"`
	RunError      string                  `json:"run_error"`
	ParentContext string                  `json:"parent_context_error"`
	ChildContext  string                  `json:"child_context_error"`
	Stdout        capabilityOutputRecord  `json:"stdout"`
	Stderr        capabilityOutputRecord  `json:"stderr"`
}

func preserveCapabilityAttempt(dir, name string, record capabilityAttempt, stdout, stderr *capabilityOutput) error {
	record.Stdout, record.Stderr = stdout.record(), stderr.record()
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return errors.Join(
		os.WriteFile(filepath.Join(dir, name+".stdout.bin"), stdout.prefix, 0600),
		os.WriteFile(filepath.Join(dir, name+".stderr.bin"), stderr.prefix, 0600),
		os.WriteFile(filepath.Join(dir, name+".attempt.json"), raw, 0600),
	)
}

// The child context inherits the one original deadline. Output overflow can only
// shorten it. Raw prefixes and process state are saved before interpreting output.
func runCapabilityAttempt(ctx context.Context, dir, name, executable string, args []string, files []*os.File, identity capabilityBuildIdentity) ([]byte, error) {
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	stdout, stderr := &capabilityOutput{cancel: cancel}, &capabilityOutput{cancel: cancel}
	cmd := exec.CommandContext(child, executable, args...)
	cmd.Stdout, cmd.Stderr, cmd.ExtraFiles = stdout, stderr, files
	cmd.WaitDelay = time.Second
	runErr := cmd.Run()
	identity.Binary = capabilityIdentity(identity.Binary.Path)
	record := capabilityAttempt{Schema: 1, Arguments: cmd.Args, Build: identity, Started: cmd.Process != nil, ExitCode: -1}
	if cmd.ProcessState != nil {
		record.ProcessState, record.Exited, record.ExitCode = true, cmd.ProcessState.Exited(), cmd.ProcessState.ExitCode()
		if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			record.Signal = int(status.Signal())
		}
	}
	if runErr != nil {
		record.RunError = runErr.Error()
	}
	if err := ctx.Err(); err != nil {
		record.ParentContext = err.Error()
	}
	if err := child.Err(); err != nil {
		record.ChildContext = err.Error()
	}
	if err := preserveCapabilityAttempt(dir, name, record, stdout, stderr); err != nil {
		return nil, fmt.Errorf("preserve capability attempt: %w", err)
	}
	if stdout.overflow || stderr.overflow {
		return nil, errors.New("capability command exceeded bounded output")
	}
	if runErr != nil {
		return nil, runErr
	}
	if err := child.Err(); err != nil {
		return nil, err
	}
	return stdout.prefix, nil
}

func TestDarwinCapabilityEvidencePreservesFailedAttempts(t *testing.T) {
	for _, name := range []string{"nonzero-exit", "bounded-output", "custody-refused"} {
		t.Run(name, func(t *testing.T) {
			cancelled := 0
			stdout := &capabilityOutput{cancel: func() { cancelled++ }}
			stderr := &capabilityOutput{}
			payload := []byte("partial observer output")
			if name == "bounded-output" {
				payload = bytes.Repeat([]byte("x"), capabilityOutputLimit+17)
			}
			if n, err := stdout.Write(payload); n != len(payload) || err != nil {
				t.Fatal("capture did not report consumed bytes", n, err)
			}
			if _, err := stderr.Write([]byte("failure detail")); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if name == "custody-refused" {
				dir = filepath.Join(dir, "missing-parent")
			}
			record := capabilityAttempt{Schema: 1, ExitCode: 7, RunError: "synthetic child failure"}
			err := preserveCapabilityAttempt(dir, "failed", record, stdout, stderr)
			if name == "custody-refused" {
				if err == nil {
					t.Fatal("missing evidence directory was accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join(dir, "failed.attempt.json"))
			if err != nil {
				t.Fatal(err)
			}
			var retained capabilityAttempt
			if err := json.Unmarshal(raw, &retained); err != nil || retained.ExitCode != 7 || retained.RunError != record.RunError || retained.Stdout != stdout.record() || retained.Stderr != stderr.record() {
				t.Fatal("failed attempt state was lost", retained, err)
			}
			for stream, want := range map[string][]byte{"stdout": stdout.prefix, "stderr": stderr.prefix} {
				got, err := os.ReadFile(filepath.Join(dir, "failed."+stream+".bin"))
				if err != nil || !bytes.Equal(got, want) {
					t.Fatal("failed raw output was lost", stream, err)
				}
			}
			if name == "bounded-output" && (len(stdout.prefix) != capabilityOutputLimit || stdout.observed != uint64(len(payload)) || !stdout.overflow || cancelled != 1) {
				t.Fatal("output was not bounded with cancellation and observed length")
			}
		})
	}
}

type resourceCapabilityObservation struct {
	Schema                  int    `json:"schema"`
	Scope                   string `json:"scope"`
	FullNativeQualification bool   `json:"full_native_qualification"`
	RootContext             bool   `json:"root_context"`
	MemoryAPIPresent        bool   `json:"memory_api_present"`
	MemoryQueryInvoked      bool   `json:"memory_query_invoked"`
	MemoryReturn            int    `json:"memory_return"`
	MemoryErrno             int    `json:"memory_errno"`
	ActiveMB                int32  `json:"active_mb"`
	ActiveAttributes        uint32 `json:"active_attributes"`
	InactiveMB              int32  `json:"inactive_mb"`
	InactiveAttributes      uint32 `json:"inactive_attributes"`
	AddressReturn           int    `json:"address_return"`
	AddressErrno            int    `json:"address_errno"`
	AddressSoft             string `json:"address_soft"`
	AddressHard             string `json:"address_hard"`
	FileEligible            bool   `json:"file_eligible"`
	AllocationReturn        int    `json:"allocation_return"`
	AllocationErrno         int    `json:"allocation_errno"`
	AllocatedBytes          int64  `json:"allocated_bytes"`
	SyncReturn              int    `json:"sync_return"`
	SyncErrno               int    `json:"sync_errno"`
	AfterStatReturn         int    `json:"after_stat_return"`
	AfterSize               int64  `json:"after_size"`
	AfterBlocks             int64  `json:"after_blocks"`
	SameFile                bool   `json:"same_file"`
}

func readResourceCapabilityObservation(raw []byte) (resourceCapabilityObservation, error) {
	var out resourceCapabilityObservation
	if len(raw) == 0 || len(raw) > 4096 {
		return out, fmt.Errorf("unbounded resource observation")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&out); err != nil {
		return resourceCapabilityObservation{}, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return resourceCapabilityObservation{}, fmt.Errorf("trailing resource observation")
	}
	if out.Schema != 1 || out.Scope != "ordinary-self-query-owned-file" || out.FullNativeQualification ||
		out.RootContext && out.MemoryQueryInvoked || !out.MemoryAPIPresent && out.MemoryQueryInvoked ||
		!out.MemoryQueryInvoked && out.MemoryReturn == 0 ||
		out.MemoryReturn != 0 && out.MemoryReturn != -1 ||
		out.MemoryReturn == 0 && out.MemoryErrno != 0 || out.MemoryReturn == -1 && out.MemoryErrno <= 0 || out.MemoryReturn != 0 &&
		(out.ActiveMB != 0 || out.InactiveMB != 0 || out.ActiveAttributes != 0 || out.InactiveAttributes != 0) {
		return resourceCapabilityObservation{}, fmt.Errorf("inconsistent or overclaimed resource observation")
	}
	for _, value := range []string{out.AddressSoft, out.AddressHard} {
		if _, err := strconv.ParseUint(value, 10, 64); err != nil {
			return resourceCapabilityObservation{}, fmt.Errorf("invalid address-space observation")
		}
	}
	return out, nil
}

func TestDarwinResourceCapabilityObservationKeepsPartialResultsSeparate(t *testing.T) {
	base := resourceCapabilityObservation{Schema: 1, Scope: "ordinary-self-query-owned-file",
		MemoryAPIPresent: true, MemoryQueryInvoked: true, MemoryReturn: -1, MemoryErrno: 1,
		AddressSoft: "18446744073709551615", AddressHard: "18446744073709551615"}
	for _, name := range []string{"denied-query", "observed-query", "unavailable-api", "overclaim", "partial-query", "root-query", "wrong-schema", "wide-address", "unknown-field", "trailing-value", "overlong", "wrong-return", "success-with-error", "failure-without-error", "uninvoked-success"} {
		t.Run(name, func(t *testing.T) {
			in := base
			good := name == "denied-query" || name == "observed-query" || name == "unavailable-api"
			switch name {
			case "observed-query":
				in.MemoryReturn, in.MemoryErrno, in.ActiveMB, in.InactiveMB = 0, 0, 256, 128
			case "unavailable-api":
				in.MemoryAPIPresent, in.MemoryQueryInvoked, in.MemoryErrno = false, false, 78
			case "overclaim":
				in.FullNativeQualification = true
			case "partial-query":
				in.ActiveMB = 256
			case "root-query":
				in.RootContext = true
			case "wrong-schema":
				in.Schema++
			case "wide-address":
				in.AddressSoft = "18446744073709551616"
			case "wrong-return":
				in.MemoryReturn = 1
			case "success-with-error":
				in.MemoryReturn = 0
			case "failure-without-error":
				in.MemoryErrno = 0
			case "uninvoked-success":
				in.MemoryQueryInvoked, in.MemoryReturn, in.MemoryErrno = false, 0, 0
			}
			raw, err := json.Marshal(in)
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "unknown-field":
				raw = append(append(raw[:len(raw)-1:len(raw)-1], []byte(",\"lease\":true")...), '}')
			case "trailing-value":
				raw = append(raw, []byte(" {}")...)
			case "overlong":
				raw = bytes.Repeat([]byte(" "), 4097)
			}
			got, err := readResourceCapabilityObservation(raw)
			if good && (err != nil || got != in) || !good && (err == nil || got != (resourceCapabilityObservation{})) {
				t.Fatal("query refusal, partial output or authority claim was conflated", got, err)
			}
		})
	}
}

func TestDarwinOwnedResourceCapabilitiesDoNotQualifyWholeExecution(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("ordinary-user observation is not exercised by a privileged test process")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "resource-capabilities")
	parent := os.Getenv("SAGE_NATIVE_PRIMITIVE_EVIDENCE_DIR")
	if parent == "" {
		parent = t.TempDir()
	}
	evidence, err := os.MkdirTemp(parent, "capability-observation-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("bounded capability attempt evidence: %s", evidence)
	compiler, err := exec.LookPath("cc")
	if err != nil {
		compiler = "cc"
	}
	identity := capabilityBuildIdentity{Source: capabilityIdentity(filepath.Join("testdata", "resource_capabilities.c")),
		Binary: capabilityFileIdentity{Path: binary}, Compiler: capabilityIdentity(compiler), CompilerVersionRecord: "compiler-version.attempt.json"}
	if _, err := runCapabilityAttempt(ctx, evidence, "compiler-version", compiler, []string{"--version"}, nil, identity); err != nil {
		t.Fatalf("observe compiler identity after preserving attempt: %v", err)
	}
	if _, err := runCapabilityAttempt(ctx, evidence, "compile", compiler, []string{"-std=c11", "-Wall", "-Wextra", "-Werror", "-O2",
		"-o", binary, filepath.Join("testdata", "resource_capabilities.c")}, nil, identity); err != nil {
		t.Fatalf("compile owned capability observer after preserving attempt: %v", err)
	}
	identity.Binary = capabilityIdentity(binary)
	if identity.Source.Error != "" || identity.Compiler.Error != "" || identity.Binary.Error != "" {
		t.Fatal("observed source, compiler or new binary identity is incomplete", identity)
	}
	for _, name := range []string{"empty-private-file", "populated-file-refused"} {
		t.Run(name, func(t *testing.T) {
			f, err := os.CreateTemp(t.TempDir(), "owned-allocation-")
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			original := []byte(nil)
			if name == "populated-file-refused" {
				original = []byte("owned fixture content remains unchanged")
				if _, err := f.Write(original); err != nil {
					t.Fatal(err)
				}
			}
			before, err := f.Stat()
			if err != nil {
				t.Fatal(err)
			}
			raw, err := runCapabilityAttempt(ctx, evidence, name, binary, nil, []*os.File{f}, identity)
			if err != nil {
				t.Fatalf("observe owned capabilities: %v", err)
			}
			got, err := readResourceCapabilityObservation(raw)
			if err != nil || got.RootContext || got.FullNativeQualification || got.AfterStatReturn != 0 || !got.SameFile {
				t.Fatal("invalid or overclaimed native observation", got, err)
			}
			after, err := f.Stat()
			if err != nil || !os.SameFile(before, after) || after.Size() != before.Size() {
				t.Fatal("observer replaced or resized the owned file", err)
			}
			retained, err := os.ReadFile(f.Name())
			if err != nil || !bytes.Equal(retained, original) {
				t.Fatal("observer changed original file contents", err)
			}
			if len(original) == 0 {
				if !got.FileEligible || got.AllocationReturn != 0 || got.AllocationErrno != 0 ||
					got.AllocatedBytes < 32*1024 || got.SyncReturn != 0 || got.SyncErrno != 0 ||
					got.AfterSize != 0 || got.AfterBlocks < 64 {
					t.Fatal("owned file backing was not observed", got)
				}
			} else if got.FileEligible || got.AllocationReturn == 0 || got.AllocatedBytes != 0 || got.AfterSize != int64(len(original)) {
				t.Fatal("populated file was eligible for preallocation", got)
			}
			t.Logf("memory query rc=%d errno=%d, file allocation rc=%d; whole native qualification=false", got.MemoryReturn, got.MemoryErrno, got.AllocationReturn)
		})
	}
}
