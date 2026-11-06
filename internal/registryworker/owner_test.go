package registryworker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func candidateConfiguration(directory string) Configuration {
	asset := Asset{Path: filepath.Join(directory, "input"), Bytes: 4, SHA256: strings.Repeat("a", 64)}
	return Configuration{
		Launcher: asset, Worker: asset, Directory: directory,
		Limits: Limits{CPUSeconds: 10, FileBytes: 1 << 20, Descriptors: 64,
			InputBytes: 4096, ResultBytes: 4096, ErrorBytes: 4096,
			Duration: 10 * time.Second, CleanupReserve: time.Second},
	}
}

func TestConfigurationRequiresFiniteDistinctResourceScopes(t *testing.T) {
	base := candidateConfiguration(t.TempDir())
	if !validConfiguration(base) {
		t.Fatal("finite candidate refused")
	}
	for _, test := range []struct {
		name string
		edit func(*Configuration)
	}{
		{"relative-directory", func(c *Configuration) { c.Directory = "relative" }},
		{"directory-nul", func(c *Configuration) { c.Directory += "\x00" }},
		{"directory-size", func(c *Configuration) { c.Directory = "/" + strings.Repeat("x", 4096) }},
		{"relative-executable", func(c *Configuration) { c.Worker.Path = "relative" }},
		{"executable-nul", func(c *Configuration) { c.Worker.Path += "\x00" }},
		{"executable-size", func(c *Configuration) { c.Worker.Path = "/" + strings.Repeat("x", 4096) }},
		{"empty-asset", func(c *Configuration) { c.Worker.Bytes = 0 }},
		{"large-asset", func(c *Configuration) { c.Worker.Bytes = 128<<20 + 1 }},
		{"short-hash", func(c *Configuration) { c.Worker.SHA256 = "a" }},
		{"nonhex-hash", func(c *Configuration) { c.Worker.SHA256 = strings.Repeat("g", 64) }},
		{"uppercase-hash", func(c *Configuration) { c.Worker.SHA256 = strings.Repeat("A", 64) }},
		{"zero-cpu", func(c *Configuration) { c.Limits.CPUSeconds = 0 }},
		{"large-cpu", func(c *Configuration) { c.Limits.CPUSeconds = 31*24*60*60 + 1 }},
		{"zero-file", func(c *Configuration) { c.Limits.FileBytes = 0 }},
		{"infinite-file", func(c *Configuration) { c.Limits.FileBytes = 1<<63 - 1 }},
		{"too-few-descriptors", func(c *Configuration) { c.Limits.Descriptors = 15 }},
		{"too-many-descriptors", func(c *Configuration) { c.Limits.Descriptors = 65537 }},
		{"zero-input", func(c *Configuration) { c.Limits.InputBytes = 0 }},
		{"large-input", func(c *Configuration) { c.Limits.InputBytes = 64<<20 + 1 }},
		{"zero-result", func(c *Configuration) { c.Limits.ResultBytes = 0 }},
		{"large-result", func(c *Configuration) { c.Limits.ResultBytes = 64<<20 + 1 }},
		{"zero-diagnostic", func(c *Configuration) { c.Limits.ErrorBytes = 0 }},
		{"large-diagnostic", func(c *Configuration) { c.Limits.ErrorBytes = 64<<10 + 1 }},
		{"no-cleanup-window", func(c *Configuration) { c.Limits.CleanupReserve = 0 }},
		{"no-action-window", func(c *Configuration) { c.Limits.Duration = c.Limits.CleanupReserve }},
		{"large-duration", func(c *Configuration) { c.Limits.Duration = 31*24*time.Hour + 1 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := base
			test.edit(&changed)
			if validConfiguration(changed) {
				t.Fatal("invalid configuration admitted")
			}
			if o, err := New(context.Background(), changed); o != nil || !errors.Is(err, ErrConfiguration) {
				t.Fatal("invalid configuration created an owner", err)
			}
		})
	}
	for _, ctx := range []context.Context{nil, canceledContext()} {
		if o, err := New(ctx, base); o != nil || !errors.Is(err, ErrConfiguration) {
			t.Fatal("missing original lifetime admitted", err)
		}
	}
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestNewOwnerKeepsItsOriginalCleanupDeadline(t *testing.T) {
	c := candidateConfiguration(t.TempDir())
	shortConfiguration := c
	shortConfiguration.Limits.CleanupReserve = time.Minute
	shortConfiguration.Limits.Duration = 2 * time.Minute
	short, cancelShort := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShort()
	if owner, err := New(short, shortConfiguration); owner != nil ||
		platformSupported() && !errors.Is(err, ErrConfiguration) ||
		!platformSupported() && !errors.Is(err, ErrUnsupported) {
		t.Fatal("unfunded cleanup interval or unsupported host admitted", err)
	}
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	o, err := New(parent, c)
	if !platformSupported() {
		if o != nil || !errors.Is(err, ErrUnsupported) {
			t.Fatal("unsupported host returned a live owner", err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	defer o.cancelWhole()
	defer o.cancelAction()
	wholeDeadline, wholeOK := o.whole.Deadline()
	actionDeadline, actionOK := o.action.Deadline()
	if !wholeOK || !actionOK || wholeDeadline.Sub(actionDeadline) != c.Limits.CleanupReserve {
		t.Fatal("action and cleanup do not share the original finite reservation")
	}
	cancel()
	if o.action.Err() != context.Canceled || o.whole.Err() != nil {
		t.Fatal("caller cancellation consumed the cleanup reservation")
	}
	if _, err := o.Run([]byte("echo")); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled action proceeded to an executable copy", err)
	}
	assertEntries, readErr := os.ReadDir(c.Directory)
	if readErr != nil || len(assertEntries) != 0 {
		t.Fatal("canceled owner touched its selected directory", readErr)
	}
}

func TestOutputBoundaryCancelsWithoutKeepingTheExcess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := limitedOutput{limit: 4, cancel: cancel}
	if n, err := w.Write([]byte("abcd")); n != 4 || err != nil || ctx.Err() != nil {
		t.Fatal("exact boundary rejected", n, err)
	}
	if n, err := w.Write([]byte("secret")); n != 0 || !errors.Is(err, ErrOutputLimit) || !w.exceeded || ctx.Err() != context.Canceled || w.data.String() != "abcd" {
		t.Fatal("overflow was retained or did not revoke the action", n, err)
	}
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	partial := limitedOutput{limit: 4, cancel: cancel2}
	if n, err := partial.Write([]byte("abcde")); n != 4 || !errors.Is(err, ErrOutputLimit) || partial.data.String() != "abcd" || ctx2.Err() != context.Canceled {
		t.Fatal("partial boundary changed its accepted prefix", n, err)
	}
}

func fileAsset(t *testing.T, path string, data []byte) Asset {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(data)
	return Asset{Path: path, Bytes: int64(len(data)), SHA256: hex.EncodeToString(h[:])}
}

func TestAssetCopyUsesExactPinnedBytesAndExclusiveDestination(t *testing.T) {
	directory := t.TempDir()
	asset := fileAsset(t, filepath.Join(directory, "original"), []byte("original"))
	destination := filepath.Join(directory, "copy")
	if err := copyAsset(context.Background(), asset, destination); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	info, statErr := os.Stat(destination)
	if err != nil || statErr != nil || string(data) != "original" || info.Mode().Perm() != 0o500 {
		t.Fatal("copy lost its own bytes or final mode", err, statErr)
	}
	if err := copyAsset(context.Background(), asset, destination); !errors.Is(err, ErrUnavailable) {
		t.Fatal("existing destination replaced", err)
	}
	if err := copyAsset(canceledContext(), asset, filepath.Join(directory, "canceled")); err != context.Canceled {
		t.Fatal("canceled owner copied input", err)
	}
}

func TestAssetCopyRefusesChangedAbsentAndLinkedInputs(t *testing.T) {
	directory := t.TempDir()
	asset := fileAsset(t, filepath.Join(directory, "original"), []byte("original"))
	for _, test := range []struct {
		name string
		edit func(*Asset)
	}{
		{"wrong-length", func(a *Asset) { a.Bytes++ }},
		{"wrong-digest", func(a *Asset) { a.SHA256 = strings.Repeat("0", 64) }},
		{"absent", func(a *Asset) { a.Path += ".absent" }},
		{"directory", func(a *Asset) { a.Path = directory }},
		{"linked", func(a *Asset) {
			link := filepath.Join(directory, "link")
			if err := os.Symlink(a.Path, link); err != nil {
				t.Fatal(err)
			}
			a.Path = link
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := asset
			test.edit(&changed)
			if err := copyAsset(context.Background(), changed, filepath.Join(directory, "copy-"+test.name)); !errors.Is(err, ErrUnavailable) {
				t.Fatal("changed source copied", err)
			}
		})
	}
}

func TestCanceledReaderCannotConsumeTheOriginal(t *testing.T) {
	original := bytes.NewBufferString("unchanged")
	r := contextReader{ctx: canceledContext(), reader: original}
	if data, err := io.ReadAll(r); len(data) != 0 || err != context.Canceled || original.String() != "unchanged" {
		t.Fatal("canceled reader consumed original", err)
	}
	valid := contextReader{ctx: context.Background(), reader: strings.NewReader("owned")}
	if data, err := io.ReadAll(valid); string(data) != "owned" || err != nil {
		t.Fatal("ordinary EOF changed", string(data), err)
	}
}

func TestCleanupRemovesOnlyExplicitCreatedPaths(t *testing.T) {
	directory := t.TempDir()
	owned := filepath.Join(directory, "owned")
	if err := os.Mkdir(owned, 0o700); err != nil {
		t.Fatal(err)
	}
	one, two := filepath.Join(owned, "launcher"), filepath.Join(owned, "worker")
	fileAsset(t, one, []byte("one"))
	fileAsset(t, two, []byte("two"))
	if err := removeCreated(context.Background(), one, two, owned); err != nil {
		t.Fatal(err)
	}
	// An unexpected file must remain, and its directory cannot be certified
	// as cleaned. There is no recursive deletion of an unexamined namespace.
	if err := os.Mkdir(owned, 0o700); err != nil {
		t.Fatal(err)
	}
	unexpected := filepath.Join(owned, "unexpected")
	fileAsset(t, unexpected, []byte("preserved"))
	if err := removeCreated(context.Background(), one, two, owned); !errors.Is(err, ErrUnavailable) {
		t.Fatal("nonempty directory certified as cleaned", err)
	}
	if data, err := os.ReadFile(unexpected); err != nil || string(data) != "preserved" {
		t.Fatal("unexpected path deleted", err)
	}
	if err := removeCreated(canceledContext(), unexpected); !errors.Is(err, context.Canceled) {
		t.Fatal("expired cleanup owner performed I/O", err)
	}
	if data, err := os.ReadFile(unexpected); err != nil || string(data) != "preserved" {
		t.Fatal("expired cleanup consumed its retained input", err)
	}
}

// expiryAfterRemoval is a synthetic unit context: the original finite deadline
// is retained, but the cleanup boundary is advanced only after the observed
// private directory disappears. It does not measure a host clock or filesystem
// service bound. Only tests supply this context, never Configuration callers.
type expiryAfterRemoval struct {
	context.Context
	mu                sync.Mutex
	pattern           string
	observed, expired bool
	done              chan struct{}
}

func (c *expiryAfterRemoval) Done() <-chan struct{} { return c.done }

func (c *expiryAfterRemoval) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.expired {
		return context.DeadlineExceeded
	}
	if err := c.Context.Err(); err != nil {
		return err
	}
	paths, err := filepath.Glob(c.pattern)
	if err != nil {
		return nil
	}
	if len(paths) > 0 {
		c.observed = true
	} else if c.observed {
		c.expired = true
		close(c.done)
		return context.DeadlineExceeded
	}
	return nil
}

func TestCleanupRechecksItsBoundaryAfterTheLastDeletion(t *testing.T) {
	owned := filepath.Join(t.TempDir(), ".sage-registry-worker-unit")
	if err := os.Mkdir(owned, 0o700); err != nil {
		t.Fatal(err)
	}
	ctx := &expiryAfterRemoval{Context: context.Background(), pattern: owned, done: make(chan struct{})}
	if err := removeCreated(ctx, owned); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("deletion past the cleanup boundary certified completion", err)
	}
	if _, err := os.Stat(owned); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("final deletion was not exercised", err)
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("synthetic cleanup boundary was not crossed")
	}
}

func TestCopiedAndSpentOwnersCannotRun(t *testing.T) {
	var missing *Owner
	if _, err := missing.Run([]byte("input")); !errors.Is(err, ErrUnavailable) {
		t.Fatal("missing owner ran", err)
	}
	o := &Owner{spent: true}
	o.self = o
	if _, err := o.Run([]byte("input")); !errors.Is(err, ErrSpent) {
		t.Fatal("spent owner ran", err)
	}
	// Construct a distinct value pointing at the original, without copying a
	// sync.Mutex. Identity is checked before either object's mutex is used.
	copied := &Owner{self: o}
	if _, err := copied.Run([]byte("input")); !errors.Is(err, ErrUnavailable) {
		t.Fatal("copied owner ran", err)
	}
	for _, raw := range []string{"{}", "null", `{"spent":false}`} {
		if err := json.Unmarshal([]byte(raw), o); !errors.Is(err, ErrUnavailable) || !o.spent {
			t.Fatal("serialized DATA changed or restored the owner", err)
		}
	}
}
