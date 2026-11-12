package compilermemory

// These units simulate native outcomes with small Go buffers. They establish
// control/error behavior only, never physical backing or native qualification.

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

type simulatedRegionOperations struct {
	mu                                sync.Mutex
	calls                             []string
	mapErr, lockErr, unlockErr, unmapErr error
	returnBytesOnMapError              bool
	wrongMapLength                    bool
	onMap, onLock, onUnlock, onUnmap    func()
}

func (s *simulatedRegionOperations) record(name string, hook func()) {
	s.mu.Lock()
	s.calls = append(s.calls, name)
	s.mu.Unlock()
	if hook != nil {
		hook()
	}
}

func (s *simulatedRegionOperations) mapBytes(n int) ([]byte, error) {
	s.record("map", s.onMap)
	if s.mapErr != nil && !s.returnBytesOnMapError {
		return nil, s.mapErr
	}
	if s.wrongMapLength {
		n--
	}
	return make([]byte, n), s.mapErr
}

func (s *simulatedRegionOperations) lockBytes([]byte) error {
	s.record("lock", s.onLock)
	return s.lockErr
}

func (s *simulatedRegionOperations) unlockBytes([]byte) error {
	s.record("unlock", s.onUnlock)
	return s.unlockErr
}

func (s *simulatedRegionOperations) unmapBytes([]byte) error {
	s.record("unmap", s.onUnmap)
	return s.unmapErr
}

func (s *simulatedRegionOperations) recorded() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.calls)
}

type controlledRegionContext struct {
	context.Context
	mu           sync.Mutex
	deadline     time.Time
	calls        int
	failAt       int
	blockAt      int
	entered      chan struct{}
	release      chan struct{}
}

func (c *controlledRegionContext) Deadline() (time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.deadline.IsZero() {
		return c.deadline, true
	}
	return c.Context.Deadline()
}

func (c *controlledRegionContext) Err() error {
	c.mu.Lock()
	c.calls++
	call, failAt, blockAt := c.calls, c.failAt, c.blockAt
	entered, release := c.entered, c.release
	c.mu.Unlock()
	if blockAt > 0 && call == blockAt {
		close(entered)
		<-release
	}
	if failAt > 0 && call >= failAt {
		return context.Canceled
	}
	return c.Context.Err()
}

func (c *controlledRegionContext) changeDeadline() {
	d, _ := c.Deadline()
	c.mu.Lock()
	c.deadline = d.Add(time.Nanosecond)
	c.mu.Unlock()
}

func simulatedRegionContexts(t *testing.T) (*controlledRegionContext, *controlledRegionContext, context.CancelFunc, context.CancelFunc) {
	t.Helper()
	a, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	c, cancelCleanup := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
	t.Cleanup(cancelCleanup)
	t.Cleanup(cancel)
	return &controlledRegionContext{Context: a}, &controlledRegionContext{Context: c}, cancel, cancelCleanup
}

func TestSimulatedRegionAcquisitionFailuresKeepOwnership(t *testing.T) {
	mapError, lockError := errors.New("simulated map"), errors.New("simulated lock")
	unlockError, unmapError := errors.New("simulated unlock"), errors.New("simulated unmap")
	for _, tc := range []struct {
		name                  string
		operations            *simulatedRegionOperations
		transition            string
		want                  []error
		calls                 []string
		retained, stillLocked bool
	}{
		{"map-error", &simulatedRegionOperations{mapErr: mapError}, "", []error{mapError}, []string{"map"}, false, false},
		{"partial-map-error-and-cleanup-error", &simulatedRegionOperations{mapErr: mapError, returnBytesOnMapError: true, unmapErr: unmapError}, "", []error{mapError, unmapError}, []string{"map", "unmap"}, true, false},
		{"wrong-map-length", &simulatedRegionOperations{wrongMapLength: true}, "", []error{ErrUnavailable}, []string{"map", "unmap"}, false, false},
		{"action-after-map", &simulatedRegionOperations{}, "cancel-map", []error{context.Canceled}, []string{"map", "unmap"}, false, false},
		{"deadline-after-map", &simulatedRegionOperations{}, "deadline-map", []error{ErrConfiguration}, []string{"map", "unmap"}, false, false},
		{"lock-error", &simulatedRegionOperations{lockErr: lockError}, "", []error{lockError}, []string{"map", "lock", "unmap"}, false, false},
		{"lock-and-cleanup-error", &simulatedRegionOperations{lockErr: lockError, unmapErr: unmapError}, "", []error{lockError, unmapError}, []string{"map", "lock", "unmap"}, true, false},
		{"action-after-lock", &simulatedRegionOperations{}, "cancel-lock", []error{context.Canceled}, []string{"map", "lock", "unlock", "unmap"}, false, false},
		{"post-lock-and-both-release-errors", &simulatedRegionOperations{unlockErr: unlockError, unmapErr: unmapError}, "cancel-lock", []error{context.Canceled, unlockError, unmapError}, []string{"map", "lock", "unlock", "unmap"}, true, true},
		{"cleanup-unavailable-after-map", &simulatedRegionOperations{}, "cancel-both-map", []error{context.Canceled}, []string{"map"}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			action, cleanup, cancelAction, cancelCleanup := simulatedRegionContexts(t)
			ops := tc.operations
			switch tc.transition {
			case "cancel-map":
				ops.onMap = cancelAction
			case "deadline-map":
				ops.onMap = action.changeDeadline
			case "cancel-lock":
				ops.onLock = cancelAction
			case "cancel-both-map":
				ops.onMap = func() {
					cancelAction()
					cancelCleanup()
				}
			}
			region, err := acquireWithOperations(action, cleanup, 64, ops)
			if (region != nil) != tc.retained || !errors.Is(err, ErrUnavailable) {
				t.Fatal("acquisition result", region != nil, err)
			}
			for _, want := range tc.want {
				if !errors.Is(err, want) {
					t.Fatal("lost original failure", want, err)
				}
			}
			if !slices.Equal(ops.recorded(), tc.calls) {
				t.Fatal("acquisition cleanup order", ops.recorded(), tc.calls)
			}
			if region != nil {
				if region.usable || !region.closeCalled || region.data == nil || region.locked != tc.stillLocked {
					t.Fatal("lost partial ownership or exposed failed region")
				}
				cached := region.closeErr
				if cached == nil || region.Close() != cached || !slices.Equal(ops.recorded(), tc.calls) {
					t.Fatal("cleanup repeated or original failure replaced")
				}
				if _, err := region.Store([]byte("later")); !errors.Is(err, ErrClosed) {
					t.Fatal("failed acquisition admitted work", err)
				}
			}
		})
	}
}

func TestSimulatedRegionCleanupFailuresAreTerminal(t *testing.T) {
	unlockError, unmapError := errors.New("simulated unlock"), errors.New("simulated unmap")
	for _, tc := range []struct {
		name, transition       string
		unlockErr, unmapErr     error
		want                   []error
		calls                  []string
		mapped, stillLocked    bool
	}{
		{"expired-before-close", "cancel-before", nil, nil, []error{context.Canceled}, nil, true, true},
		{"changed-before-close", "deadline-before", nil, nil, []error{ErrConfiguration}, nil, true, true},
		{"unlock-error", "", unlockError, nil, []error{unlockError}, []string{"unlock", "unmap"}, false, false},
		{"unmap-error", "", nil, unmapError, []error{unmapError}, []string{"unlock", "unmap"}, true, false},
		{"both-native-errors", "", unlockError, unmapError, []error{unlockError, unmapError}, []string{"unlock", "unmap"}, true, true},
		{"expired-after-unlock", "cancel-unlock", nil, nil, []error{context.Canceled}, []string{"unlock"}, true, false},
		{"changed-after-unlock", "deadline-unlock", nil, nil, []error{ErrConfiguration}, []string{"unlock"}, true, false},
		{"expired-after-unmap", "cancel-unmap", nil, nil, []error{context.Canceled}, []string{"unlock", "unmap"}, false, false},
		{"changed-after-unmap", "deadline-unmap", nil, nil, []error{ErrConfiguration}, []string{"unlock", "unmap"}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			action, cleanup, _, cancelCleanup := simulatedRegionContexts(t)
			ops := &simulatedRegionOperations{}
			region, err := acquireWithOperations(action, cleanup, 64, ops)
			if err != nil {
				t.Fatal("simulated acquisition", err)
			}
			span, err := region.Store([]byte("retained"))
			if err != nil {
				t.Fatal(err)
			}
			ops.calls = nil
			ops.unlockErr, ops.unmapErr = tc.unlockErr, tc.unmapErr
			switch tc.transition {
			case "cancel-before":
				cancelCleanup()
			case "deadline-before":
				cleanup.changeDeadline()
			case "cancel-unlock":
				ops.onUnlock = cancelCleanup
			case "deadline-unlock":
				ops.onUnlock = cleanup.changeDeadline
			case "cancel-unmap":
				ops.onUnmap = cancelCleanup
			case "deadline-unmap":
				ops.onUnmap = cleanup.changeDeadline
			}
			err = region.Close()
			for _, want := range tc.want {
				if !errors.Is(err, want) {
					t.Fatal("lost cleanup failure", want, err)
				}
			}
			if region.usable || !region.closeCalled || (region.data != nil) != tc.mapped || region.locked != tc.stillLocked {
				t.Fatal("wrong terminal mapping state")
			}
			if !slices.Equal(ops.recorded(), tc.calls) || region.Close() != err || !slices.Equal(ops.recorded(), tc.calls) {
				t.Fatal("cleanup was retried or reordered", ops.recorded(), tc.calls)
			}
			if err := region.Read(span, make([]byte, 8)); !errors.Is(err, ErrClosed) {
				t.Fatal("failed cleanup admitted read", err)
			}
			if _, err := region.Slice(span, 0, 1); !errors.Is(err, ErrClosed) {
				t.Fatal("failed cleanup admitted view", err)
			}
			if _, err := region.Equal(span, []byte("retained")); !errors.Is(err, ErrClosed) {
				t.Fatal("failed cleanup admitted comparison", err)
			}
			if err := region.Seal(); !errors.Is(err, ErrClosed) {
				t.Fatal("failed cleanup admitted sealing", err)
			}
		})
	}
}

func TestSimulatedRegionPreMapAndPostAccessRefusals(t *testing.T) {
	t.Run("before-map", func(t *testing.T) {
		action, cleanup, _, _ := simulatedRegionContexts(t)
		action.failAt = 2
		ops := &simulatedRegionOperations{}
		region, err := acquireWithOperations(action, cleanup, 64, ops)
		if region != nil || !errors.Is(err, context.Canceled) || len(ops.recorded()) != 0 {
			t.Fatal("pre-map refusal entered native work", region != nil, err, ops.recorded())
		}
	})
	for _, operation := range []string{"store", "read", "equal"} {
		t.Run(operation, func(t *testing.T) {
			action, cleanup, _, _ := simulatedRegionContexts(t)
			region, err := acquireWithOperations(action, cleanup, 64, &simulatedRegionOperations{})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = region.Close() })
			span, err := region.Store([]byte("data"))
			if err != nil {
				t.Fatal(err)
			}
			action.mu.Lock()
			action.calls, action.failAt = 0, 2
			action.mu.Unlock()
			switch operation {
			case "store":
				used := region.used
				span, err = region.Store([]byte("later"))
				if !errors.Is(err, context.Canceled) || span.owner != nil || region.used != used+5 || region.usable {
					t.Fatal("entered copy was refunded or published", err)
				}
			case "read":
				result := []byte("xxxx")
				if err := region.Read(span, result); !errors.Is(err, context.Canceled) || !slices.Equal(result, []byte{0, 0, 0, 0}) {
					t.Fatal("post-copy refusal kept published bytes", result, err)
				}
			case "equal":
				if equal, err := region.Equal(span, []byte("data")); equal || !errors.Is(err, context.Canceled) {
					t.Fatal("post-check refusal published equality", equal, err)
				}
			}
		})
	}
}

func TestSimulatedRegionOriginalOwnerDeadlineAndRanges(t *testing.T) {
	action, cleanup, _, _ := simulatedRegionContexts(t)
	ops := &simulatedRegionOperations{}
	if region, err := acquireWithOperations(action, cleanup, 64, nil); region != nil || !errors.Is(err, ErrConfiguration) {
		t.Fatal("missing implementation accepted", err)
	}
	region, err := acquireWithOperations(action, cleanup, 64, ops)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = region.Close() })
	span, err := region.Store([]byte("data"))
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []Span{
		{},
		{owner: &Region{}, offset: 0, length: 1},
		{owner: region, offset: -1, length: 1},
		{owner: region, offset: region.used + 1, length: 0},
		{owner: region, offset: 0, length: region.used + 1},
	} {
		if err := region.Read(invalid, make([]byte, max(0, invalid.length))); !errors.Is(err, ErrRange) {
			t.Fatal("invalid committed range read", err)
		}
		if _, err := region.Equal(invalid, nil); !errors.Is(err, ErrRange) {
			t.Fatal("invalid committed range compared", err)
		}
		if _, err := region.Slice(invalid, 0, 0); !errors.Is(err, ErrRange) {
			t.Fatal("invalid committed view derived", err)
		}
	}
	copyOwner := &Region{self: region}
	if _, err := copyOwner.Store([]byte("copy")); !errors.Is(err, ErrConfiguration) {
		t.Fatal("copied owner admitted work", err)
	}
	if err := copyOwner.Read(span, make([]byte, 4)); !errors.Is(err, ErrConfiguration) {
		t.Fatal("copied owner admitted read", err)
	}
	empty, err := region.Store(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := region.Read(empty, nil); err != nil {
		t.Fatal("valid empty span", err)
	}
	action.changeDeadline()
	if _, err := region.Store([]byte("later")); !errors.Is(err, ErrConfiguration) {
		t.Fatal("changed action deadline accepted", err)
	}
	if _, err := region.Slice(span, 0, 1); !errors.Is(err, ErrConfiguration) {
		t.Fatal("changed action deadline view", err)
	}
	if err := region.Read(span, make([]byte, 4)); !errors.Is(err, ErrConfiguration) {
		t.Fatal("changed action deadline read", err)
	}
	if _, err := region.Equal(span, []byte("data")); !errors.Is(err, ErrConfiguration) {
		t.Fatal("changed action deadline comparison", err)
	}
	if err := region.Seal(); !errors.Is(err, ErrConfiguration) {
		t.Fatal("changed action deadline sealed", err)
	}
}

func TestSimulatedRegionCloseWaitsForActiveRead(t *testing.T) {
	action, cleanup, _, _ := simulatedRegionContexts(t)
	ops := &simulatedRegionOperations{}
	region, err := acquireWithOperations(action, cleanup, 64, ops)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = region.Close() })
	span, err := region.Store([]byte("data"))
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	releaseRead := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseRead()
	action.mu.Lock()
	action.calls, action.blockAt, action.entered, action.release = 0, 1, entered, release
	action.mu.Unlock()
	readDone := make(chan error, 1)
	result := make([]byte, 4)
	go func() { readDone <- region.Read(span, result) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("controlled read did not enter")
	}
	cleanupEntered := make(chan struct{}, 2)
	ops.onUnlock = func() { cleanupEntered <- struct{}{} }
	ops.onUnmap = func() { cleanupEntered <- struct{}{} }
	closeStarted, closeDone := make(chan struct{}), make(chan error, 1)
	go func() {
		close(closeStarted)
		closeDone <- region.Close()
	}()
	<-closeStarted
	// This bounded observation is not a scheduler service guarantee. The read
	// has entered and remains held while the original Close is invoked.
	select {
	case <-cleanupEntered:
		t.Fatal("native cleanup entered during active read")
	case err := <-closeDone:
		t.Fatal("close finished during active read", err)
	case <-time.After(20 * time.Millisecond):
	}
	releaseRead()
	select {
	case err := <-readDone:
		if err != nil || string(result) != "data" {
			t.Fatal("active read lost its backing", result, err)
		}
	case <-time.After(time.Second):
		t.Fatal("controlled read did not return")
	}
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("close did not finish after read released")
	}
	if !slices.Equal(ops.recorded(), []string{"map", "lock", "unlock", "unmap"}) {
		t.Fatal("controlled lifecycle order", ops.recorded())
	}
}
