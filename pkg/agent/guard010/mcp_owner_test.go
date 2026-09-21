package guard010

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const ownerID1 = "00000000-0000-4000-8000-000000000001"
const ownerID2 = "00000000-0000-4000-8000-000000000002"

func testOwner(t *testing.T, server bool) *testMCPOwner {
	t.Helper()
	o, e := newTestMCPOwner(server, 0, 0, 30*time.Second)
	if e != nil {
		t.Fatal(e)
	}
	return o
}
func ownerEvent(t *testing.T, o *testMCPOwner, e mcpEvent, id string) *mcpOutput {
	t.Helper()
	p, err := o.transition(e, id, time.Second, 40*time.Second, true)
	if err != nil {
		t.Fatal(e, err)
	}
	return p
}
func ownerComplete(t *testing.T, o *testMCPOwner, p *mcpOutput) {
	t.Helper()
	if _, e := o.complete(p, true, time.Second, true); e != nil {
		t.Fatal(e)
	}
}
func readyOwner(t *testing.T, server bool) *testMCPOwner {
	t.Helper()
	o := testOwner(t, server)
	if server {
		ownerComplete(t, o, ownerEvent(t, o, mcpReplyInitialize, ownerID1))
		ownerComplete(t, o, ownerEvent(t, o, mcpReplyInitialized, ""))
		ownerComplete(t, o, ownerEvent(t, o, mcpReplyList, ownerID2))
	} else {
		ownerComplete(t, o, ownerEvent(t, o, mcpSendInitialize, ownerID1))
		ownerEvent(t, o, mcpAcceptInitialize, "")
		ownerComplete(t, o, ownerEvent(t, o, mcpSendInitialized, ""))
		ownerEvent(t, o, mcpAcceptInitialized, "")
		ownerComplete(t, o, ownerEvent(t, o, mcpSendList, ownerID2))
		ownerEvent(t, o, mcpAcceptList, "")
	}
	return o
}
func TestMCPOwnerSetupAndRetainedHistory(t *testing.T) {
	for _, server := range []bool{false, true} {
		o := readyOwner(t, server)
		if o.phase != mcpReady || len(o.seen) != 2 {
			t.Fatal("setup history or phase")
		}
		// READY retires the setup deadline, not the lifetime request-ID history.
		if e := o.reserveProtected("00000000-0000-4000-8000-000000000003", 31*time.Second, true); e != nil {
			t.Fatal(e)
		}
		if e := o.reserveProtected(ownerID1, 31*time.Second, true); e == nil || o.phase != mcpClosed || len(o.seen) != 3 {
			t.Fatal("setup ID reused")
		}
	}
}
func TestMCPOwnerHistoryLimit(t *testing.T) {
	o := readyOwner(t, true)
	for i := 3; i <= 1024; i++ {
		if e := o.reserveProtected(fmt.Sprintf("00000000-0000-4000-8000-%012x", i), time.Second, true); e != nil {
			t.Fatal(i, e)
		}
	}
	if len(o.seen) != 1024 {
		t.Fatal("history")
	}
	if e := o.reserveProtected("00000000-0000-4000-8000-000000001000", time.Second, true); e == nil || o.phase != mcpClosed || len(o.seen) != 1024 {
		t.Fatal("history exhaustion")
	}
	// This isolated unit does not claim a real session can exceed 1000 records.
}
func TestMCPOwnerOutputBarrierAndOwnedDeferredFrame(t *testing.T) {
	o := testOwner(t, false)
	p := ownerEvent(t, o, mcpSendInitialize, ownerID1)
	if o.phase != mcpClientStart {
		t.Fatal("published before full send")
	}
	b := []byte("wire")
	if e := o.deferFrame(b, time.Second, true); e != nil {
		t.Fatal(e)
	}
	b[0] = 'X'
	ownerComplete(t, o, p)
	if o.phase != mcpClientWaitInitialize {
		t.Fatal("not published")
	}
	if _, e := o.complete(p, false, time.Second, true); e == nil || o.phase == mcpClosed {
		t.Fatal("stale completion mutated state")
	}
	if got := o.takeDeferred(); string(got) != "wire" {
		t.Fatal("mutable deferred frame")
	}
	if o.takeDeferred() != nil {
		t.Fatal("duplicate deferred frame")
	}
	ownerEvent(t, o, mcpAcceptInitialize, "")
}
func TestMCPOwnerOutputFailureSchedules(t *testing.T) {
	for _, kind := range []string{"partial", "close", "deadline", "rollback", "session", "overflow", "oversize", "second-output", "reenter", "pending-deadline"} {
		t.Run(kind, func(t *testing.T) {
			o := testOwner(t, false)
			p := ownerEvent(t, o, mcpSendInitialize, ownerID1)
			if e := o.deferFrame([]byte("wire"), time.Second, true); e != nil {
				t.Fatal(e)
			}
			now := 2 * time.Second
			valid, full := true, true
			switch kind {
			case "partial":
				full = false
			case "close":
				o.close()
			case "deadline":
				now = 30 * time.Second
			case "rollback":
				now = 0
			case "session":
				valid = false
			case "overflow":
				_ = o.deferFrame([]byte("extra"), now, true)
			case "oversize":
				_ = o.deferFrame(make([]byte, 32769), now, true)
			case "second-output":
				_, _ = o.transition(mcpSendInitialize, ownerID2, now, 0, true)
			case "reenter":
				_, _ = o.transition(mcpAcceptInitialize, "", now, 0, true)
			case "pending-deadline":
				_ = o.deferFrame([]byte("late"), 30*time.Second, true)
			}
			if _, e := o.complete(p, full, now, valid); e == nil || o.phase != mcpClosed || o.takeDeferred() != nil || len(o.seen) != 1 {
				t.Fatal("failure reopened or released state")
			}
		})
	}
}
func TestMCPOwnerFixedDeadlinesAndStaleIdentity(t *testing.T) {
	for _, q := range []struct{ created, now, limit time.Duration }{{-1, 0, time.Second}, {2, 1, time.Second}, {0, 0, 0}, {0, 0, 31 * time.Second}, {0, 30 * time.Second, 30 * time.Second}, {time.Duration(1<<63 - 2), time.Duration(1<<63 - 2), time.Second}} {
		if _, e := newTestMCPOwner(false, q.created, q.now, q.limit); e == nil {
			t.Fatal("invalid origin/deadline")
		}
	}
	o, e := newTestMCPOwner(false, 0, 29*time.Second, 30*time.Second)
	if e != nil {
		t.Fatal(e)
	}
	p, e := o.transition(mcpSendInitialize, ownerID1, 29*time.Second, 0, true)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = o.complete(p, true, 30*time.Second, true); e == nil {
		t.Fatal("confirmation restarted deadline")
	}
	o = readyOwner(t, false)
	p, e = o.transition(mcpProtectedOutput, "", 31*time.Second, 32*time.Second, true)
	if e != nil {
		t.Fatal(e)
	}
	other := readyOwner(t, false)
	if _, e = other.complete(p, true, 31*time.Second, true); e == nil || other.phase != mcpReady {
		t.Fatal("cross-owner publication")
	}
	if _, e = o.complete(p, true, 32*time.Second, true); e == nil || o.phase != mcpClosed {
		t.Fatal("protected deadline equality")
	}
}
func TestMCPOwnerRejectsOutOfOrderEvents(t *testing.T) {
	for event := mcpAcceptInitialize; event <= mcpProtectedOutput+1; event++ {
		o := testOwner(t, false)
		if _, e := o.transition(event, "", time.Second, 2*time.Second, true); e == nil || o.phase != mcpClosed {
			t.Fatal(event, "out of order")
		}
	}
	o := readyOwner(t, true)
	if e := o.reserveProtected("not-a-uuid", time.Second, true); e == nil || o.phase != mcpClosed {
		t.Fatal("bad ID")
	}
}

// Native goroutine schedules exercise the real coordinator mutex; no network,
// host bypass, tool effect or protocol conformance is implied by these races.
func TestMCPOwnerConcurrentCloseAndPublication(t *testing.T) {
	for i := 0; i < 100; i++ {
		o := testOwner(t, true)
		p := ownerEvent(t, o, mcpReplyInitialize, ownerID1)
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); <-start; o.close() }()
		go func() { defer wg.Done(); <-start; _, _ = o.complete(p, true, time.Second, true) }()
		close(start)
		wg.Wait()
		if o.phase != mcpClosed || o.pending != nil {
			t.Fatal("close lost")
		}
	}
}
func TestMCPOwnerBlockedSendDoesNotBlockClose(t *testing.T) {
	o := testOwner(t, true)
	p := ownerEvent(t, o, mcpReplyInitialize, ownerID1)
	started, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() { close(started); <-release; _, _ = o.complete(p, true, time.Second, true); close(finished) }()
	<-started
	closed := make(chan struct{})
	go func() { o.close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("close blocked on send")
	}
	close(release)
	<-finished
	if o.phase != mcpClosed {
		t.Fatal("late send reopened owner")
	}
}

// Deterministic trusted local clock for isolated lifecycle units.
type testMCPOwner struct {
	*mcpOwner
	now atomic.Int64
}

func newTestMCPOwner(server bool, created, now, limit time.Duration) (*testMCPOwner, error) {
	o := &testMCPOwner{}
	o.now.Store(int64(now))
	var err error
	o.mcpOwner, err = newMCPOwner(server, created, limit, func() (time.Duration, error) { return time.Duration(o.now.Load()), nil })
	return o, err
}
func (o *testMCPOwner) transition(e mcpEvent, id string, now, end time.Duration, valid bool) (*mcpOutput, error) {
	o.now.Store(int64(now))
	return o.mcpOwner.transition(e, id, end, valid)
}
func (o *testMCPOwner) reserveProtected(id string, now time.Duration, valid bool) error {
	o.now.Store(int64(now))
	return o.mcpOwner.reserveProtected(id, valid)
}
func (o *testMCPOwner) deferFrame(b []byte, now time.Duration, valid bool) error {
	o.now.Store(int64(now))
	return o.mcpOwner.deferFrame(b, valid)
}
func (o *testMCPOwner) complete(p *mcpOutput, full bool, now time.Duration, valid bool) ([]byte, error) {
	o.now.Store(int64(now))
	return nil, o.mcpOwner.complete(p, full, valid)
}

func TestMCPOwnerSamplesClockAfterLockAcquisition(t *testing.T) {
	o := testOwner(t, true)
	ownerComplete(t, o, ownerEvent(t, o, mcpReplyInitialize, ownerID1))
	ownerComplete(t, o, ownerEvent(t, o, mcpReplyInitialized, ""))
	p := ownerEvent(t, o, mcpReplyList, ownerID2)
	if err := o.deferFrame([]byte("deferred"), time.Second, true); err != nil {
		t.Fatal(err)
	}
	o.mu.Lock()
	o.now.Store(int64(29 * time.Second))
	started, done := make(chan struct{}), make(chan error, 1)
	go func() { close(started); done <- o.mcpOwner.complete(p, true, true) }()
	<-started
	o.now.Store(int64(30 * time.Second))
	o.mu.Unlock()
	if err := <-done; err == nil || o.phase != mcpClosed || o.takeDeferred() != nil {
		t.Fatal("publication used a pre-lock sample")
	}
}
func TestMCPOwnerClockFailure(t *testing.T) {
	for _, clock := range []func() (time.Duration, error){nil, func() (time.Duration, error) { return 0, ErrInvalid }, func() (time.Duration, error) { panic("clock unavailable") }} {
		if _, err := newMCPOwner(false, 0, time.Second, clock); err == nil {
			t.Fatal("bad constructor clock")
		}
		o := testOwner(t, true)
		p := ownerEvent(t, o, mcpReplyInitialize, ownerID1)
		o.clock = clock
		if err := o.mcpOwner.complete(p, true, true); err == nil || o.phase != mcpClosed {
			t.Fatal("bad publication clock")
		}
	}
}
