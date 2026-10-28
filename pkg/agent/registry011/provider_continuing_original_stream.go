package registry011

import (
	"context"
	"crypto/sha256"
	"hash"
	"io"
	"os"
	"sync"
)

const providerHistoryOriginalChunk = 8192

// These are source-call and byte uppers for ONE ordinary-content scan, not
// allocator qualification, elapsed time, semantic replay or an admitted hold.
// The future owner must also fund path/identity/hash-state/native overhead and
// independently bracket the operation with its original qualified clock.
type providerHistoryOriginalWork struct {
	readBytes, hashBytes, readCalls, hashWrites, metadataCalls, bufferBytes uint64
}

// A plan fixes the exact retained object before opening it. It is private DATA,
// not a custody ACK, a consumed permission or a constructor for a Source.
type providerHistoryOriginalPlan struct {
	reference providerIntegrationObjectRef
	maximum   uint64
}

func (*providerHistoryOriginalPlan) UnmarshalJSON([]byte) error { return ErrConfiguration }

func newProviderHistoryOriginalPlan(ref providerIntegrationObjectRef, maximum uint64) (providerHistoryOriginalPlan, error) {
	if maximum == 0 || maximum > 32<<20 || ref.bytes == 0 || ref.bytes > maximum || !lowerHash(ref.digest) {
		return providerHistoryOriginalPlan{}, ErrConfiguration
	}
	switch ref.kind {
	case "request":
		if ref.bytes > 16<<10 {
			return providerHistoryOriginalPlan{}, ErrConfiguration
		}
	case "params":
		if ref.bytes > 8<<10 {
			return providerHistoryOriginalPlan{}, ErrConfiguration
		}
	case "response", "tls":
	default:
		return providerHistoryOriginalPlan{}, ErrConfiguration
	}
	return providerHistoryOriginalPlan{ref, maximum}, nil
}

func (p providerHistoryOriginalPlan) upper() (providerHistoryOriginalWork, error) {
	if _, err := newProviderHistoryOriginalPlan(p.reference, p.maximum); err != nil {
		return providerHistoryOriginalWork{}, err
	}
	// Each ReadAt has one bounded chunk; the final one-byte probe must return
	// EOF. Two checkIdentity calls use four metadata operations each. The
	// object adds Lstat/Open/Stat/Stat/Lstat/Close: fourteen non-read calls.
	// These count Go source calls, not kernel retries or CPU/time guarantees.
	chunks := (p.reference.bytes-1)/providerHistoryOriginalChunk + 1
	return providerHistoryOriginalWork{p.reference.bytes + 1, p.reference.bytes, chunks + 1, chunks, 14, providerHistoryOriginalChunk + 1}, nil
}

// This workspace reuses its explicit byte buffers and fixed SHA-256 state;
// repeated reads do not rely on immediate collection of an object-sized slice.
// File metadata, paths and returned digest allocations still need separate
// whole-program accounting and allocator qualification. It is NOT a hold.
type providerHistoryOriginalReader struct {
	mu     sync.Mutex
	owner  *providerHistoryOriginalReader
	disk   *retentionDisk
	hash   hash.Hash
	buffer [providerHistoryOriginalChunk]byte
	tail   [1]byte
	failed bool
}

func (*providerHistoryOriginalReader) UnmarshalJSON([]byte) error { return ErrConfiguration }

func newProviderHistoryOriginalReader(disk *retentionDisk) (*providerHistoryOriginalReader, error) {
	if disk == nil || disk.root == nil || disk.lock == nil {
		return nil, ErrConfiguration
	}
	r := &providerHistoryOriginalReader{disk: disk, hash: sha256.New()}
	r.owner = r
	return r, nil
}

// scan returns only an integrity result for one ordinary SHA-256 object. It
// does not parse its message/schema or validate a Registry transition. Both
// independent replay owners must perform their own original reads; this result
// cannot replace either reader or any existing custody/final-page checks.
// No existing collector calls this candidate or has its memory bound changed.
func (r *providerHistoryOriginalReader) scan(ctx context.Context, p providerHistoryOriginalPlan) (digest string, err error) {
	if _, err := p.upper(); err != nil || ctx == nil || r == nil {
		return "", ErrConfiguration
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.owner != r || r.failed || r.disk == nil || r.disk.root == nil || r.disk.lock == nil || r.hash == nil {
		return "", ErrConfiguration
	}
	// Any subsequent uncertainty poisons this reader. The full prospective
	// charge stays with the caller; this helper has no refund/retry mechanism.
	r.failed = true
	disk := r.disk
	if ctx.Err() != nil || disk.checkIdentity() != nil {
		return "", ErrCandidateRetention
	}
	name := providerObjectName(p.reference.kind, p.reference.digest)
	info, err := disk.root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || uint64(info.Size()) != p.reference.bytes {
		return "", ErrCandidateRetention
	}
	f, err := disk.root.Open(name)
	if err != nil {
		return "", ErrCandidateRetention
	}
	closed := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
	}()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) || opened.Size() != info.Size() || !opened.Mode().IsRegular() {
		return "", ErrCandidateRetention
	}
	r.hash.Reset() // Reuses hash memory; does not reset any resource lifetime.
	// The constructor enforces <=32MiB. Keep the bound explicit at this cast
	// too; neither a large platform word nor a forged plan can widen it.
	if p.reference.bytes > 32<<20 {
		return "", ErrConfiguration
	}
	size := int64(p.reference.bytes)
	for offset := int64(0); offset < size; {
		if ctx.Err() != nil {
			return "", ErrCandidateRetention
		}
		want := len(r.buffer)
		if remaining := size - offset; remaining > 0 && remaining <= providerHistoryOriginalChunk {
			want = int(remaining)
		}
		n, err := f.ReadAt(r.buffer[:want], offset)
		if err != nil || n != want {
			return "", ErrCandidateRetention
		}
		_, _ = r.hash.Write(r.buffer[:want]) // The fixed SHA-256 writer cannot fail.
		offset += int64(want)
	}
	if n, err := f.ReadAt(r.tail[:], size); n != 0 || err != io.EOF {
		return "", ErrCandidateRetention
	}
	after, statErr := f.Stat()
	path, pathErr := disk.root.Lstat(name)
	if statErr != nil || pathErr != nil || !os.SameFile(info, after) || !os.SameFile(info, path) || after.Size() != size || path.Size() != size || !after.Mode().IsRegular() || !path.Mode().IsRegular() || !after.ModTime().Equal(info.ModTime()) || !path.ModTime().Equal(info.ModTime()) {
		return "", ErrCandidateRetention
	}
	closed = true
	if closeErr := f.Close(); closeErr != nil || ctx.Err() != nil || disk.checkIdentity() != nil {
		return "", ErrCandidateRetention
	}
	digest = nodeHex(r.hash.Sum(nil))
	if digest != p.reference.digest {
		return "", ErrCandidateRetention
	}
	r.failed = false
	return digest, nil
}
