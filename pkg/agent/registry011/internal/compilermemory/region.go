// Package compilermemory owns the bytes used by an internal compiler
// representation. It does not qualify a complete compiler or issue authority.
package compilermemory

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"time"
)

var (
	ErrConfiguration = errors.New("compiler memory: invalid configuration")
	ErrUnavailable   = errors.New("compiler memory: unavailable")
	ErrClosed        = errors.New("compiler memory: closed")
	ErrRange         = errors.New("compiler memory: invalid range")
	ErrUnsupported   = errors.New("compiler memory: unsupported platform")
)

// Span is checked DATA referring to committed bytes of one original region.
// Copying a valid span does not extend that region's lifetime.
type Span struct {
	owner          *Region
	offset, length int
}

// This private seam supports simulated failure units. Public acquisition fixes
// the implementation to nativeRegionOperations; callers cannot supply it.
type regionOperations interface {
	mapBytes(int) ([]byte, error)
	lockBytes([]byte) error
	unlockBytes([]byte) error
	unmapBytes([]byte) error
}

type nativeRegionOperations struct{}

func (nativeRegionOperations) mapBytes(n int) ([]byte, error) { return mapRegion(n) }
func (nativeRegionOperations) lockBytes(b []byte) error       { return lockRegion(b) }
func (nativeRegionOperations) unlockBytes(b []byte) error     { return unlockRegion(b) }
func (nativeRegionOperations) unmapBytes(b []byte) error      { return unmapRegion(b) }

func (Span) UnmarshalJSON([]byte) error { return ErrConfiguration }

// Region owns one mapping. Its Go wrapper and synchronization objects are
// separate allocations; mapping success is not their resource qualification.
type Region struct {
	self            *Region
	operations      regionOperations
	mu              sync.RWMutex
	action, cleanup context.Context
	actionDeadline  time.Time
	cleanupDeadline time.Time
	data            []byte
	capacity        int
	used            int
	locked          bool
	usable          bool
	sealed          bool
	closeCalled     bool
	closeErr        error
}

func (*Region) UnmarshalJSON([]byte) error { return ErrConfiguration }

// Acquire obtains the actual mapping before returning usable storage. The
// caller supplies a derived capacity and the original action/cleanup contexts.
// It must retain the fixed process owner's hard stop. An error can return a
// non-nil region only when cleanup failed; no access is permitted in that state.
func Acquire(action, cleanup context.Context, capacity int) (*Region, error) {
	return acquireWithOperations(action, cleanup, capacity, nativeRegionOperations{})
}

func acquireWithOperations(action, cleanup context.Context, capacity int, operations regionOperations) (*Region, error) {
	if action == nil || cleanup == nil || operations == nil || capacity <= 0 || action.Err() != nil || cleanup.Err() != nil {
		return nil, ErrConfiguration
	}
	deadline, finite := action.Deadline()
	cleanupDeadline, cleanupFinite := cleanup.Deadline()
	if !finite || !cleanupFinite || cleanupDeadline.Before(deadline) || !deadline.After(time.Now()) {
		return nil, ErrConfiguration
	}
	page := os.Getpagesize()
	padding := (page - capacity%page) % page
	if capacity > int(^uint(0)>>1)-padding {
		return nil, ErrConfiguration
	}
	r := &Region{action: action, cleanup: cleanup, actionDeadline: deadline, cleanupDeadline: cleanupDeadline, capacity: capacity, operations: operations}
	r.self = r
	if err := r.actionError(); err != nil {
		return nil, err
	}
	data, err := operations.mapBytes(capacity + padding)
	r.data = data
	if err == nil {
		if len(data) != capacity+padding {
			err = ErrUnavailable
		} else {
			err = r.actionError()
		}
	}
	if err == nil {
		err = operations.lockBytes(data)
		if err == nil {
			r.locked = true
			err = r.actionError()
		}
	}
	if err != nil {
		cleanupErr := r.Close()
		if cleanupErr != nil {
			return r, errors.Join(ErrUnavailable, err, cleanupErr)
		}
		return nil, errors.Join(ErrUnavailable, err)
	}
	r.usable = true
	return r, nil
}

func (r *Region) actionError() error {
	if err := r.action.Err(); err != nil {
		return err
	}
	deadline, finite := r.action.Deadline()
	if !finite || deadline != r.actionDeadline {
		return ErrConfiguration
	}
	return nil
}

func (r *Region) accessError() error {
	if r.closeCalled || !r.usable {
		return ErrClosed
	}
	return r.actionError()
}

func (r *Region) cleanupError() error {
	if err := r.cleanup.Err(); err != nil {
		return err
	}
	deadline, finite := r.cleanup.Deadline()
	if !finite || deadline != r.cleanupDeadline {
		return ErrConfiguration
	}
	return nil
}

func (r *Region) contains(s Span) bool {
	return s.owner == r && s.offset >= 0 && s.length >= 0 && s.offset <= r.used && s.length <= r.used-s.offset
}

// Store copies into the actual region. An entered copy consumes its range even
// if a later context check fails. Sealing prevents further mutation by Store.
func (r *Region) Store(value []byte) (Span, error) {
	if r == nil || r.self != r {
		return Span{}, ErrConfiguration
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.accessError(); err != nil {
		return Span{}, err
	}
	if r.sealed || len(value) > r.capacity-r.used {
		return Span{}, ErrRange
	}
	s := Span{owner: r, offset: r.used, length: len(value)}
	r.used += len(value)
	copy(r.data[s.offset:r.used], value)
	if err := r.actionError(); err != nil {
		r.usable = false
		return Span{}, err
	}
	return s, nil
}

// Slice derives a bounded view without exporting a native pointer.
func (r *Region) Slice(s Span, offset, length int) (Span, error) {
	if r == nil || r.self != r {
		return Span{}, ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return Span{}, err
	}
	if !r.contains(s) || offset < 0 || length < 0 || offset > s.length || length > s.length-offset {
		return Span{}, ErrRange
	}
	return Span{owner: r, offset: s.offset + offset, length: length}, nil
}

// Read copies a complete span into an already supplied destination. No native
// slice escapes the access lock, so Close cannot unmap an active access.
func (r *Region) Read(s Span, destination []byte) error {
	if r == nil || r.self != r {
		return ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return err
	}
	if !r.contains(s) || len(destination) != s.length {
		return ErrRange
	}
	copy(destination, r.data[s.offset:s.offset+s.length])
	if err := r.actionError(); err != nil {
		clear(destination)
		return err
	}
	return nil
}

func (r *Region) Equal(s Span, value []byte) (bool, error) {
	if r == nil || r.self != r {
		return false, ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return false, err
	}
	if !r.contains(s) {
		return false, ErrRange
	}
	equal := bytes.Equal(r.data[s.offset:s.offset+s.length], value)
	if err := r.actionError(); err != nil {
		return false, err
	}
	return equal, nil
}

func (r *Region) Seal() error {
	if r == nil || r.self != r {
		return ErrConfiguration
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.accessError(); err != nil {
		return err
	}
	r.sealed = true
	return nil
}

// Close spends one cleanup attempt. A repeated call returns its original
// outcome without repeating native operations or extending the deadline.
func (r *Region) Close() error {
	if r == nil || r.self != r {
		return ErrConfiguration
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closeCalled {
		return r.closeErr
	}
	r.closeCalled, r.usable = true, false
	if len(r.data) == 0 {
		return nil
	}
	if err := r.cleanupError(); err != nil {
		r.closeErr = err
		return err
	}
	var unlockErr error
	if r.locked {
		unlockErr = r.operations.unlockBytes(r.data)
		if unlockErr == nil {
			r.locked = false
		}
	}
	if err := r.cleanupError(); err != nil {
		r.closeErr = errors.Join(unlockErr, err)
		return r.closeErr
	}
	unmapErr := r.operations.unmapBytes(r.data)
	if unmapErr == nil {
		r.data, r.locked = nil, false
	}
	r.closeErr = errors.Join(unlockErr, unmapErr, r.cleanupError())
	return r.closeErr
}
