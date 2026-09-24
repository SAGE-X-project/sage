package guard010

import (
	"context"
	"testing"
)

func TestMCPParentAdmissionExistsOnlyDuringAdmittedWorker(t *testing.T) {
	for _, retire := range []bool{false, true} {
		f := newAdmissionFixture(t, 1)
		f.executor.entered = make(chan struct{})
		f.executor.release = make(chan struct{})
		if receipt, err := f.admit(t); err != nil || !receipt.Committed() {
			t.Fatal("parent was not admitted", err)
		}
		f.gate.mu.Lock()
		preClaim := f.gate.slots[0].invocation.ParentAdmission()
		f.gate.mu.Unlock()
		if preClaim != nil {
			t.Fatal("admission escaped before worker claim")
		}
		done := make(chan error, 1)
		go func() { _, err := f.gate.runOne(context.Background()); done <- err }()
		<-f.executor.entered
		f.executor.mu.Lock()
		invocation := f.executor.observed
		f.executor.mu.Unlock()
		parent := invocation.ParentAdmission()
		if parent == nil || parent.Authorized(context.Background(), f.intent) != nil {
			t.Fatal("running worker lacks exact parent admission")
		}
		changed := append([]byte(nil), f.intent...)
		changed[len(changed)-1] ^= 1
		if parent.Authorized(context.Background(), changed) == nil {
			t.Fatal("different parent envelope was admitted")
		}
		if retire {
			f.gate.retire()
			if parent.Authorized(context.Background(), f.intent) == nil {
				t.Fatal("retired parent remained authorized")
			}
		}
		close(f.executor.release)
		result := <-done
		if !retire && (result != nil || f.state(t) != "COMPLETED") {
			t.Fatal("authorized parent worker did not complete", result)
		}
		if parent.Authorized(context.Background(), f.intent) == nil {
			t.Fatal("finished parent remained authorized")
		}
	}
}
