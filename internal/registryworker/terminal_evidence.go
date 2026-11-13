package registryworker

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// EvidenceConfiguration selects a parent-owned file outside disposable
// executable copies. Zero disables the channel. MaximumBytes bounds parent
// readback, not a total-storage reservation or the child's per-file limit.
type EvidenceConfiguration struct {
	Directory    string
	MaximumBytes int
}

// EvidenceResult is raw completed-attempt DATA, including a failure prefix.
// ReadbackComplete means these exact bytes were synced, read and closed within
// the original cleanup lifetime. It does not interpret or authenticate a child
// journal, certify a protocol terminal state, or approve retention operations.
type EvidenceResult struct {
	// Path is the creation name. A refused readback may leave that name
	// unavailable after a host namespace change; it is never a live handle.
	Path             string
	ObservedBytes    int64
	Prefix           []byte
	PrefixSHA256     string
	LimitExceeded    bool
	ReadbackComplete bool
}

type terminalEvidence struct {
	root            *os.Root
	file, directory *os.File
	path, name      string
	maximum         int
	spent           bool
}

func validEvidenceConfiguration(c EvidenceConfiguration, l Limits) bool {
	if c == (EvidenceConfiguration{}) {
		return true
	}
	return filepath.IsAbs(c.Directory) && len(c.Directory) <= 4096 &&
		!strings.ContainsRune(c.Directory, 0) && c.MaximumBytes > 0 &&
		c.MaximumBytes <= 64<<20 && uint64(c.MaximumBytes) <= l.FileBytes
}

// The selected directory is part of the trusted host/store assumption. It
// must already exist and be private. Creating a file does not qualify that
// host, allocate total storage, or select an operating retention policy.
func openTerminalEvidence(ctx context.Context, c EvidenceConfiguration) (_ *terminalEvidence, err error) {
	if c == (EvidenceConfiguration{}) {
		return nil, nil
	}
	if ctx == nil || ctx.Err() != nil {
		return nil, ErrUnavailable
	}
	info, err := os.Lstat(c.Directory)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return nil, ErrUnavailable
	}
	root, err := os.OpenRoot(c.Directory)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, root.Close())
		}
	}()
	directory, err := root.Open(".")
	if err != nil {
		return nil, ErrUnavailable
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, directory.Close())
		}
	}()
	opened, err := directory.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.IsDir() || opened.Mode().Perm()&0o077 != 0 || ctx.Err() != nil {
		return nil, ErrUnavailable
	}
	return createTerminalEvidenceInRoot(ctx, c, root, directory)
}

// Both creation and entry sync use this exact already-open root. There is
// one exclusive creation, with no namespace fallback or collision retry.
func createTerminalEvidenceInRoot(ctx context.Context, c EvidenceConfiguration, root *os.Root, directory *os.File) (*terminalEvidence, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	name := ".sage-registry-terminal-" + rand.Text()
	file, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, ErrUnavailable
	}
	return &terminalEvidence{root: root, file: file, directory: directory, maximum: c.MaximumBytes,
		path: filepath.Join(c.Directory, name), name: name}, nil
}

// Freeze the action outcome before any terminal readback uses the separate
// cleanup window. A later action expiry cannot invalidate a timely action;
// expiry of the original whole lifetime still refuses cleanup completion.
func finishWorkerAction(action, whole context.Context, evidence *terminalEvidence) (EvidenceResult, error) {
	actionErr := action.Err()
	result, err := evidence.finish(whole)
	return result, errors.Join(actionErr, err)
}

// finish runs after Wait (or failed startup). Only executable copies are
// disposable: this file is deliberately retained on success, cancellation,
// overflow, incomplete readback and later executable-cleanup failure.
func (e *terminalEvidence) finish(ctx context.Context) (out EvidenceResult, err error) {
	if e == nil {
		return out, nil
	}
	if e.spent {
		return out, ErrSpent
	}
	e.spent = true
	out.Path = e.path
	defer func() {
		err = errors.Join(err, e.file.Close(), e.directory.Close(), e.root.Close())
		if ctx != nil {
			err = errors.Join(err, ctx.Err())
		}
		out.ReadbackComplete = err == nil
	}()
	if ctx == nil || ctx.Err() != nil {
		return out, ErrUnavailable
	}
	if err = e.file.Sync(); err != nil {
		return out, ErrUnavailable
	}
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	before, err := e.file.Stat()
	if err != nil || !before.Mode().IsRegular() || before.Size() < 0 {
		return out, ErrUnavailable
	}
	out.ObservedBytes = before.Size()
	size := before.Size()
	if size > int64(e.maximum) {
		size = int64(e.maximum)
		out.LimitExceeded = true
	}
	data := make([]byte, int(size))
	n, readErr := io.ReadFull(contextReader{ctx: ctx, reader: io.NewSectionReader(e.file, 0, size)}, data)
	out.Prefix = data[:n:n]
	digest := sha256.Sum256(out.Prefix)
	out.PrefixSHA256 = hex.EncodeToString(digest[:])
	if readErr != nil {
		return out, errors.Join(ErrUnavailable, readErr)
	}
	after, err := e.file.Stat()
	if err != nil || after.Size() != before.Size() || after.ModTime() != before.ModTime() {
		return out, ErrUnavailable
	}
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	entry, entryErr := e.root.Lstat(e.name)
	current, currentErr := os.Lstat(e.path)
	directory, directoryErr := e.directory.Stat()
	parent, parentErr := os.Lstat(filepath.Dir(e.path))
	if entryErr != nil || currentErr != nil || directoryErr != nil || parentErr != nil ||
		!entry.Mode().IsRegular() || !current.Mode().IsRegular() || !parent.IsDir() ||
		!os.SameFile(before, entry) || !os.SameFile(before, current) || !os.SameFile(directory, parent) {
		return out, ErrUnavailable
	}
	if err = e.directory.Sync(); err != nil {
		return out, ErrUnavailable
	}
	if out.LimitExceeded {
		return out, ErrEvidenceLimit
	}
	return out, nil
}
