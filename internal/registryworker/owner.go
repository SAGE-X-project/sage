package registryworker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrConfiguration = errors.New("registry worker: invalid configuration")
	ErrUnavailable   = errors.New("registry worker: attempt unavailable")
	ErrSpent         = errors.New("registry worker: attempt already spent")
	ErrUnsupported   = errors.New("registry worker: unsupported platform")
	ErrOutputLimit   = errors.New("registry worker: output limit exceeded")
)

// Asset identifies a previously built executable. Copies live in this owner's
// private directory. Its caller-selected identity is DATA, not an authenticated
// compiler/runtime identity or sufficient source/native qualification.
type Asset struct {
	Path   string
	Bytes  int64
	SHA256 string
}

// Limits distinguish actual kernel controls from bounded parent I/O. CPUSeconds
// is cumulative process CPU time; FileBytes is PER FILE, not total storage.
// Neither is a minimum service guarantee. There is no physical-memory promise.
type Limits struct {
	CPUSeconds, FileBytes, Descriptors  uint64
	InputBytes, ResultBytes, ErrorBytes int
	Duration, CleanupReserve            time.Duration
	// ChildCleanupReserve is part of CleanupReserve, not an extension. Zero
	// retains immediate cancellation; a positive value permits a cooperative
	// child stop before the fixed hard-stop and parent cleanup boundaries.
	ChildCleanupReserve time.Duration
}

type Configuration struct {
	Launcher, Worker Asset
	Directory        string
	Limits           Limits
}

// Result is detached completed-attempt DATA. Success here never certifies the
// complete profile or creates a Registry derivation, ACK or dispatch permission.
type Result struct {
	Started, LimitsApplied bool
	ExitCode               int
	Signal                 int
	UserCPU, SystemCPU     time.Duration
	Output, Diagnostics    []byte
	OutputExceeded         bool
	// CleanupComplete covers the parent's private executable copies only.
	// It never certifies a child journal, Registry custody or child Go defers.
	CleanupComplete bool
}

type Owner struct {
	self           *Owner
	mu             sync.Mutex
	spent          bool
	configuration  Configuration
	parent         context.Context
	whole, action  context.Context
	hardStop       context.Context
	cancelWhole    context.CancelFunc
	cancelAction   context.CancelFunc
	cancelHardStop context.CancelFunc
}

func (*Owner) UnmarshalJSON([]byte) error { return ErrUnavailable }

// New fixes a common lifetime before copies, input consumption or process
// startup. The action must stop before the separately held cleanup window.
// Existing resource/cost qualification and explicit host selection remain the
// caller's obligations; constructing this mechanism does not satisfy them.
func New(ctx context.Context, c Configuration) (*Owner, error) {
	if ctx == nil || ctx.Err() != nil || !validConfiguration(c) {
		return nil, ErrConfiguration
	}
	if !platformSupported() {
		return nil, ErrUnsupported
	}
	now := time.Now()
	deadline := now.Add(c.Limits.Duration)
	if inherited, ok := ctx.Deadline(); ok && inherited.Before(deadline) {
		deadline = inherited
	}
	actionDeadline := deadline.Add(-c.Limits.CleanupReserve)
	if !actionDeadline.After(now) {
		return nil, ErrConfiguration
	}
	// User cancellation stops work, not the separately held terminal cleanup.
	// The common reservation still ends at its ORIGINAL finite deadline. It is
	// never extended after cancellation, native failure or a smaller actual run.
	whole, cancelWhole := context.WithDeadline(context.WithoutCancel(ctx), deadline)
	action, cancelAction := context.WithDeadline(ctx, actionDeadline)
	hardStop, cancelHardStop := action, cancelAction
	if c.Limits.ChildCleanupReserve > 0 {
		hardStop, cancelHardStop = context.WithDeadline(context.WithoutCancel(ctx), actionDeadline.Add(c.Limits.ChildCleanupReserve))
	}
	c.Directory = strings.Clone(c.Directory)
	c.Launcher.Path, c.Launcher.SHA256 = strings.Clone(c.Launcher.Path), strings.Clone(c.Launcher.SHA256)
	c.Worker.Path, c.Worker.SHA256 = strings.Clone(c.Worker.Path), strings.Clone(c.Worker.SHA256)
	o := &Owner{configuration: c, parent: ctx, whole: whole, action: action, hardStop: hardStop,
		cancelWhole: cancelWhole, cancelAction: cancelAction, cancelHardStop: cancelHardStop}
	o.self = o
	return o, nil
}

func validConfiguration(c Configuration) bool {
	l := c.Limits
	return validAsset(c.Launcher) && validAsset(c.Worker) &&
		filepath.IsAbs(c.Directory) && len(c.Directory) <= 4096 && !strings.ContainsRune(c.Directory, 0) &&
		l.CPUSeconds > 0 && l.CPUSeconds <= 31*24*60*60 && l.FileBytes > 0 && l.FileBytes < 1<<63-1 &&
		l.Descriptors >= 16 && l.Descriptors <= 65536 &&
		l.InputBytes > 0 && l.InputBytes <= 64<<20 && l.ResultBytes > 0 && l.ResultBytes <= 64<<20 &&
		l.ErrorBytes > 0 && l.ErrorBytes <= 64<<10 && l.CleanupReserve > 0 &&
		l.ChildCleanupReserve >= 0 && l.ChildCleanupReserve < l.CleanupReserve &&
		l.Duration > l.CleanupReserve && l.Duration <= 31*24*time.Hour
}

func validAsset(a Asset) bool {
	if !filepath.IsAbs(a.Path) || len(a.Path) > 4096 || strings.ContainsRune(a.Path, 0) || a.Bytes <= 0 || a.Bytes > 128<<20 || len(a.SHA256) != 64 {
		return false
	}
	for _, c := range a.SHA256 {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// Run consumes the attempt before checking the input. A refusal, cancellation,
// native error or cleanup failure cannot restore it. Only this creator removes
// its two private executable copies and their directory, exactly once.
func (o *Owner) Run(input []byte) (result Result, err error) {
	result.ExitCode = -1
	if o == nil || o.self != o {
		return result, ErrUnavailable
	}
	o.mu.Lock()
	if o.spent {
		o.mu.Unlock()
		return result, ErrSpent
	}
	o.spent = true
	o.mu.Unlock()
	defer o.cancelWhole()
	defer o.cancelAction()
	defer o.cancelHardStop()
	c := o.configuration
	if len(input) == 0 || len(input) > c.Limits.InputBytes || o.action.Err() != nil {
		return result, errors.Join(ErrUnavailable, o.action.Err())
	}
	// No caller slice survives this boundary. Input mutation during this copy
	// is forbidden, just as it is for the original config snapshot constructor.
	ownedInput := bytes.Clone(input)
	directory, err := os.MkdirTemp(c.Directory, ".sage-registry-worker-")
	if err != nil {
		return result, errors.Join(ErrUnavailable, o.action.Err())
	}
	launcher, worker := filepath.Join(directory, "launcher"), filepath.Join(directory, "worker")
	defer func() {
		cleanupErr := errors.Join(removeCreated(o.whole, launcher, worker, directory), o.whole.Err())
		result.CleanupComplete = cleanupErr == nil
		err = errors.Join(err, cleanupErr, o.parent.Err())
		if err != nil {
			result.Output = nil
		}
	}()
	// Capture the action's outcome before entering the separately held cleanup
	// phase. An action deadline that expires DURING valid cleanup does not
	// retroactively invalidate an action that already completed in time.
	defer func() { err = errors.Join(err, o.action.Err()) }()
	if err = copyAsset(o.action, c.Launcher, launcher); err != nil {
		return result, err
	}
	if err = copyAsset(o.action, c.Worker, worker); err != nil {
		return result, err
	}
	readyRead, readyWrite, err := os.Pipe()
	if err != nil {
		return result, ErrUnavailable
	}
	readClosed := false
	defer func() {
		if !readClosed {
			err = errors.Join(err, readyRead.Close())
		}
	}()
	cmd, err := platformCommand(o.action, launcher, worker, c.Limits)
	if err != nil {
		return result, errors.Join(err, readyWrite.Close())
	}
	stdout := limitedOutput{limit: c.Limits.ResultBytes, cancel: o.cancelAction}
	stderr := limitedOutput{limit: c.Limits.ErrorBytes, cancel: o.cancelAction}
	readiness := limitedOutput{limit: 256, cancel: o.cancelAction}
	cmd.Dir, cmd.Env = directory, []string{"LANG=C", "LC_ALL=C"}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = bytes.NewReader(ownedInput), &stdout, &stderr
	cmd.ExtraFiles = []*os.File{readyWrite}
	cmd.WaitDelay = time.Second
	if c.Limits.ChildCleanupReserve > 0 {
		cmd.WaitDelay = c.Limits.ChildCleanupReserve
	}
	if err = cmd.Start(); err != nil {
		return result, errors.Join(ErrUnavailable, readyWrite.Close())
	}
	result.Started = true
	var processExited chan struct{}
	var hardStopped <-chan error
	if c.Limits.ChildCleanupReserve > 0 {
		processExited = make(chan struct{})
		hardStopped = watchHardStop(o.hardStop, cmd.Process, processExited)
	}
	closeWriteErr := readyWrite.Close()
	readyDone := make(chan error, 1)
	go func() {
		_, readErr := io.Copy(&readiness, readyRead)
		readyDone <- readErr
	}()
	waitErr := cmd.Wait()
	var hardStopErr error
	if hardStopped != nil {
		close(processExited)
		hardStopErr = <-hardStopped
	}
	// The launcher closes the control descriptor on exec. If the worker dies
	// before exec, process exit also closes it. The selected program is a single
	// child; arbitrary descendant processes are outside this candidate contract.
	var readyErr, closeReadErr error
	select {
	case readyErr = <-readyDone:
	case <-o.action.Done():
		closeReadErr = readyRead.Close()
		readClosed = true
		readyErr = <-readyDone
	}
	result.LimitsApplied = platformReadiness(readiness.data.Bytes(), c.Limits)
	result.OutputExceeded = stdout.exceeded || stderr.exceeded || readiness.exceeded
	result.Output, result.Diagnostics = bytes.Clone(stdout.data.Bytes()), bytes.Clone(stderr.data.Bytes())
	if state := cmd.ProcessState; state != nil {
		result.ExitCode = state.ExitCode()
		result.UserCPU, result.SystemCPU = state.UserTime(), state.SystemTime()
		result.Signal = platformSignal(state)
	}
	err = errors.Join(waitErr, hardStopErr, closeWriteErr, closeReadErr, readyErr)
	if !result.LimitsApplied || result.OutputExceeded {
		err = errors.Join(err, ErrUnavailable)
	}
	return result, err
}

// The fixed boundary is chosen before bootstrap. Even an early cooperative
// cancellation cannot move it later. Use the owning Process handle throughout,
// never a cached numeric PID or process-group identifier.
func watchHardStop(ctx context.Context, process *os.Process, exited <-chan struct{}) <-chan error {
	done := make(chan error, 1)
	go func() {
		select {
		case <-ctx.Done():
			err := process.Kill()
			if errors.Is(err, os.ErrProcessDone) {
				err = nil
			}
			done <- err
		case <-exited:
			done <- nil
		}
	}()
	return done
}

func copyAsset(ctx context.Context, asset Asset, destination string) (err error) {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	info, err := os.Lstat(asset.Path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != asset.Bytes {
		return ErrUnavailable
	}
	// #nosec G304 -- Caller-selected source: regular size/opened identity and
	// bounded exact hash are checked before use; this is not deployment trust.
	f, err := os.Open(asset.Path)
	if err != nil {
		return ErrUnavailable
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() || opened.Size() != asset.Bytes {
		return ErrUnavailable
	}
	// #nosec G304 -- Run owns the private directory and fixed basename;
	// exclusive creation cannot replace an existing destination.
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return ErrUnavailable
	}
	defer func() { err = errors.Join(err, out.Close()) }()
	h := sha256.New()
	r := io.LimitReader(contextReader{ctx: ctx, reader: f}, asset.Bytes+1)
	n, err := io.Copy(io.MultiWriter(out, h), r)
	if err != nil || n != asset.Bytes || hex.EncodeToString(h.Sum(nil)) != asset.SHA256 {
		return ErrUnavailable
	}
	if err = out.Sync(); err != nil {
		return ErrUnavailable
	}
	return out.Chmod(0o500)
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if ownerErr := r.ctx.Err(); ownerErr != nil {
		return n, errors.Join(err, ownerErr)
	}
	return n, err
}

type limitedOutput struct {
	data     bytes.Buffer
	limit    int
	exceeded bool
	cancel   context.CancelFunc
}

func (w *limitedOutput) Write(p []byte) (int, error) {
	remaining := w.limit - w.data.Len()
	if len(p) > remaining {
		_, _ = w.data.Write(p[:remaining])
		w.exceeded = true
		w.cancel()
		return remaining, ErrOutputLimit
	}
	return w.data.Write(p)
}

func removeCreated(ctx context.Context, paths ...string) error {
	var err error
	for _, path := range paths {
		if ctx.Err() != nil {
			return errors.Join(err, ctx.Err())
		}
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, ErrUnavailable)
		}
	}
	// A final native deletion may return after the original cleanup lifetime.
	// Physical removal alone must not certify timely cleanup completion.
	return errors.Join(err, ctx.Err())
}
