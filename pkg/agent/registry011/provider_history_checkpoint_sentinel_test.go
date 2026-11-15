package registry011

import (
	"context"
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"
)

// The caller holds the original store mutex. Fixture setup uses its existing
// normal owned TLS exchange; the member failure itself is inert UNIT DATA.
// No oversized reply is sent over a connection, and no attack server is built.
func historyCheckpointReadFailureData(t *testing.T, o *providerHistoryGroupOwner, d *providerHistoryGroupDispatch, extra int) (*providerHistoryDispatchAttempt, context.CancelFunc) {
	t.Helper()
	a, _, cancel, err := o.prepareMemberCandidateLocked(context.Background(), d, 0)
	if err != nil || a == nil || cancel == nil {
		t.Fatal("genuine original preparation before simulated member failure", err)
	}
	member := a.member
	if member.capture.WireBytes != 16<<10 || len(member.request.wire) >= member.capture.WireBytes || extra < 0 || extra > 1 {
		cancel()
		t.Fatal("finite scalar-class fixture and original request boundary")
	}
	// Exactly one funded rejected byte, or one byte beyond that sentinel.
	n := member.capture.WireBytes - len(member.request.wire) + 1 + extra
	a.failure = &providerHistoryCaptureData{knownFailure: true, failureStage: "read", requestWritten: len(member.request.wire), responseRead: n,
		sample: &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: member.request.method, HTTPVersion: "HTTP/1.1",
			ServerName: "rpc.sepolia.ethpandaops.io", TLSVersion: tls.VersionTLS13, CipherSuite: tls.TLS_AES_128_GCM_SHA256,
			VerifiedChains: [][]string{{admissionTestDigest("3")}}, RequestHead: []byte(member.request.head), RequestBody: []byte(member.request.body), ResponseWire: make([]byte, n)}}
	return a, cancel
}

func TestProviderHistoryFailedCheckpointRetainsFundedReadSentinel(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 1)
	s, m := o.store, o.meter
	s.mu.Lock()
	defer s.mu.Unlock()
	pin, ack, held, window, clockStart := o.pin, o.last, s.stored, o.windowStart, o.clockStart
	a, cancel := historyCheckpointReadFailureData(t, o, d, 0)
	defer cancel()
	defer o.failLocked()
	if err := o.retainFailedCaptureLocked(context.Background(), d, a); err != nil {
		t.Fatal("funded rejected byte must remain recordable as terminal failure DATA", err)
	}
	if o.last == nil || o.last.claim == nil {
		t.Fatal("terminal failure acknowledgment was not retained")
	}
	c := o.last.claim
	checkpoint := o.parent.recovery.confirmed
	if o.last == ack || c == nil || !c.used || !c.renamed || c.failure.kind != "failure" || checkpoint == nil || checkpoint != c.checkpoint ||
		!checkpoint.failedS || checkpoint.previous == nil || checkpoint.previous.ack != ack || checkpoint.failureStage != "read" || checkpoint.failureWritten != len(a.member.request.wire) ||
		checkpoint.failureReceived != a.member.capture.WireBytes-len(a.member.request.wire)+1 ||
		o.pin.sequence != pin.sequence+1 || o.pin != s.frontierFiles.pin || s.frontierFiles.historyPending != nil || s.activeMeter.Load() != nil ||
		!o.failed || !m.failed || !m.closed || !o.parent.failed || !s.clock.historyRevoked || s.stored != held || o.windowStart != window || o.clockStart != clockStart ||
		o.order.state != providerHistoryOrderFailed || o.order.sent != 1 || o.order.next != 0 || d.retained != 0 || a.original != nil || a.files != 0 || a.connection != nil ||
		!o.coordinator.terminalDataValidObserved(nil) || !o.custodian.terminalDataValidObserved(nil) || o.coordinator.valid() || o.custodian.valid() {
		t.Fatal("read sentinel became success, lost terminal custody or changed the original allowance/window")
	}
	raw, err := os.ReadFile(filepath.Join(s.config.evidenceDirectory, providerHistoryStoredName(c.failure, nil)))
	if err != nil {
		t.Fatal(err)
	}
	original, err := parseProviderHistoryFailureOriginalObserved(raw, c.failure.digest, d.coordinator.ticket, d.coordinator.consumption, 0,
		historyFailureRoots(t, o), o.parent.tls, s.config.bounds.maximumObjectBytes, nil)
	if err != nil || original == nil || original.stage != "read" || original.received != checkpoint.failureReceived || original.written != checkpoint.failureWritten {
		t.Fatal("terminal checkpoint and retained failure original disagree at the funded boundary", err)
	}
	// This asserts checkpoint consistency for a synthetic private UNIT path;
	// it is not live transport-origin or production-factory qualification.
}

func TestProviderHistoryFailedCheckpointRefusesPastFundedReadSentinel(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 1)
	s := o.store
	s.mu.Lock()
	defer s.mu.Unlock()
	pin, ack, held, retained, window, clockStart := o.pin, o.last, s.stored, o.retention, o.windowStart, o.clockStart
	a, cancel := historyCheckpointReadFailureData(t, o, d, 1)
	defer cancel()
	defer o.failLocked()
	if err := o.retainFailedCaptureLocked(context.Background(), d, a); err == nil || o.last != ack || o.pin != pin || s.frontierFiles.pin != pin ||
		s.frontierFiles.historyPending != nil || o.parent.recovery.pending != nil || o.parent.recovery.confirmed.ack != ack ||
		o.retention != retained || s.stored != held || o.windowStart != window || o.clockStart != clockStart ||
		d.retained != 0 || o.order.next != 0 || a.original != nil || a.files != 0 {
		t.Fatal("bytes beyond the single funded sentinel acquired failed custody or renewed original state")
	}
}
