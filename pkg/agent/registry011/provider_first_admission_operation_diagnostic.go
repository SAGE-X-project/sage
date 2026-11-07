package registry011

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
)

// Fixed diagnostic DATA only. These tags never authorize work, select a
// transport, supply an accepted E measurement or change the failed S schema.
// The first tag and pre-cleanup snapshot cannot be replaced by later failures.
type providerFirstFailureTag uint8

const (
	providerFirstFailureUnclassified providerFirstFailureTag = iota
	providerFirstFailureEntry
	providerFirstFailureConsume
	providerFirstFailureBurn
	providerFirstFailureDial
	providerFirstFailureCaptureEntry
	providerFirstFailureCapturePreflight
	providerFirstFailureTLS
	providerFirstFailureDeadline
	providerFirstFailureRequestCopy
	providerFirstFailureWrite
	providerFirstFailureHead
	providerFirstFailureHeaders
	providerFirstFailureEncoding
	providerFirstFailureLength
	providerFirstFailureTransfer
	providerFirstFailureHTTP
	providerFirstFailureStatus
	providerFirstFailureBody
	providerFirstFailureEOF
	providerFirstFailureRPC
	providerFirstFailureRead
	providerFirstFailureCaptureWork
	providerFirstFailureCaptureCopies
	providerFirstFailureExecutionClock
	providerFirstFailureExchange
	providerFirstFailureMeasurement
	providerFirstFailureRemaining
	providerFirstFailureSettlementClock
	providerFirstFailureSettlementData
	providerFirstFailureSettlementLedger
	providerFirstFailureSettlementCommit
	providerFirstFailureSettlementVerify
	providerFirstFailureResultCopy
	providerFirstFailureModel
	providerFirstFailureHandoff
)

type providerFirstFailureReason uint8

const (
	providerFirstReasonUnknown providerFirstFailureReason = iota
	providerFirstReasonRefused
	providerFirstReasonCanceled
	providerFirstReasonTimeout
	providerFirstReasonDNS
	providerFirstReasonCertificate
	providerFirstReasonNetwork
	providerFirstReasonLimit
	providerFirstReasonFormat
	providerFirstReasonTrailer
)

type providerFirstOperationDiagnostic struct {
	Tag            providerFirstFailureTag
	Reason         providerFirstFailureReason
	Detail         uint64
	RequestWritten int
	ResponseRead   int
	Clocks         uint64
	Failed, Closed bool
	Frozen         bool
}

func (d *providerFirstOperationDiagnostic) reject(tag providerFirstFailureTag, reason providerFirstFailureReason, detail uint64) {
	if d != nil && !d.Frozen && d.Tag == providerFirstFailureUnclassified {
		d.Tag, d.Reason, d.Detail = tag, reason, detail
	}
}

// Called under the existing store lock, outside meter.mu, before terminal
// cleanup. Copy existing scalar values only: no validation, I/O, clock sample,
// observer event or budget change. The owner's static layout quote includes d.
func (d *providerFirstOperationDiagnostic) freeze(m *providerWorkMeter) {
	if d == nil || d.Frozen {
		return
	}
	if m != nil {
		m.mu.Lock()
		d.Clocks, d.Failed, d.Closed = m.clocks, m.failed, m.closed
		m.mu.Unlock()
	}
	d.Frozen = true
}

// Classify the already returned native error, without retaining its text,
// resolved addresses, certificates or arbitrary error-carried payloads.
func providerFirstNativeFailureReason(err error) providerFirstFailureReason {
	if err == nil {
		return providerFirstReasonUnknown
	}
	if errors.Is(err, context.Canceled) {
		return providerFirstReasonCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return providerFirstReasonTimeout
	}
	var certificate *tls.CertificateVerificationError
	if errors.As(err, &certificate) {
		return providerFirstReasonCertificate
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return providerFirstReasonDNS
	}
	var network net.Error
	if errors.As(err, &network) {
		if network.Timeout() {
			return providerFirstReasonTimeout
		}
		return providerFirstReasonNetwork
	}
	return providerFirstReasonUnknown
}
