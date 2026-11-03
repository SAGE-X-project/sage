package registry011

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestProviderFirstFailureDiagnosticPreservesItsOriginalSnapshot(t *testing.T) {
	m := providerReplayUnitMeter()
	if !m.event(providerReadEvent, 7) {
		t.Fatal("independent finite observer precursor")
	}
	var missing *providerFirstOperationDiagnostic
	missing.reject(providerFirstFailureDial, providerFirstReasonDNS, 0)
	missing.freeze(m)
	d := providerFirstOperationDiagnostic{RequestWritten: 5, ResponseRead: 7}
	d.reject(providerFirstFailureLength, providerFirstReasonLimit, 4097)
	d.reject(providerFirstFailureRead, providerFirstReasonUnknown, 0)
	d.freeze(m)
	want := d
	if !d.Frozen || d.Tag != providerFirstFailureLength || d.Reason != providerFirstReasonLimit || d.Detail != 4097 ||
		d.Clocks != m.clocks || d.Failed || d.Closed ||
		d.RequestWritten != 5 || d.ResponseRead != 7 {
		t.Fatal("first failure or pre-cleanup scalar snapshot lost")
	}
	m.refuse()
	d.reject(providerFirstFailureSettlementVerify, providerFirstReasonRefused, 1)
	d.freeze(m)
	if d != want {
		t.Fatal("secondary cleanup changed the first diagnostic")
	}
	var unknown providerFirstOperationDiagnostic
	unknown.freeze(nil)
	unknown.reject(providerFirstFailureDial, providerFirstReasonDNS, 1)
	if !unknown.Frozen || unknown.Tag != providerFirstFailureUnclassified || unknown.Reason != providerFirstReasonUnknown {
		t.Fatal("unknown diagnostic was promoted after freezing")
	}
}

func TestProviderFirstNativeFailureReasonRetainsOnlyBoundedCategories(t *testing.T) {
	for _, row := range []struct {
		name string
		err  error
		want providerFirstFailureReason
	}{
		{"nil", nil, providerFirstReasonUnknown},
		{"canceled", fmt.Errorf("wrapped: %w", context.Canceled), providerFirstReasonCanceled},
		{"deadline", context.DeadlineExceeded, providerFirstReasonTimeout},
		{"certificate", &tls.CertificateVerificationError{Err: errors.New("finite certificate failure DATA")}, providerFirstReasonCertificate},
		{"dns", &net.DNSError{Err: "finite resolver failure DATA", Name: "diagnostic.invalid"}, providerFirstReasonDNS},
		{"network", &net.OpError{Op: "read", Net: "tcp", Err: io.EOF}, providerFirstReasonNetwork},
		{"network-timeout", &net.OpError{Op: "read", Net: "tcp", Err: os.ErrDeadlineExceeded}, providerFirstReasonTimeout},
		{"other", errors.New("finite non-native failure DATA"), providerFirstReasonUnknown},
	} {
		t.Run(row.name, func(t *testing.T) {
			if got := providerFirstNativeFailureReason(row.err); got != row.want {
				t.Fatal("native error category changed", got, row.want)
			}
		})
	}
}

func TestProviderFirstHTTPFailureTagsPreserveTheExistingParserResult(t *testing.T) {
	for _, mode := range []string{"same", "head-limit", "encoding", "length-limit", "length-format", "transfer", "status", "body-read", "rpc", "extra-byte"} {
		t.Run(mode, func(t *testing.T) {
			limits := providerTestLimits()
			body := `{"jsonrpc":"2.0","id":1,"result":"ok"}`
			status, extra, length, tail := "200 OK", "", strconv.Itoa(len(body)), ""
			want := providerFirstFailureUnclassified
			switch mode {
			case "head-limit":
				limits.HeaderBytes, want = 1, providerFirstFailureHead
			case "encoding":
				extra, want = "Content-Encoding: gzip\r\n", providerFirstFailureEncoding
			case "length-limit":
				limits.BodyBytes, want = 1, providerFirstFailureLength
			case "length-format":
				length, want = "0"+length, providerFirstFailureLength
			case "transfer":
				extra, want = "Transfer-Encoding: chunked\r\n", providerFirstFailureTransfer
			case "status":
				status, want = "503 Service Unavailable", providerFirstFailureHeaders
			case "body-read":
				length, want = strconv.Itoa(len(body)+1), providerFirstFailureBody
			case "rpc":
				body, length, want = `{}`, "2", providerFirstFailureRPC
			case "extra-byte":
				tail, want = "x", providerFirstFailureEOF
			}
			// Invalid bytes here are finite local parser DATA, never a peer,
			// network responder, arbitrary endpoint or reproduction program.
			wire := "HTTP/1.1 " + status + "\r\nContent-Type: application/json\r\n" + extra + "Content-Length: " + length + "\r\nConnection: close\r\n\r\n" + body + tail
			var d providerFirstOperationDiagnostic
			head, headers, parsed, result, err := readProviderHTTPDiagnosed(bufio.NewReader(strings.NewReader(wire)), limits, nil, 0, &d)
			oldHead, oldHeaders, oldBody, oldResult, oldErr := readProviderHTTP(bufio.NewReader(strings.NewReader(wire)), limits)
			if d.Tag != want || d.Frozen || (err == nil) != (mode == "same") || (err == nil) != (oldErr == nil) ||
				!reflect.DeepEqual(head, oldHead) || !reflect.DeepEqual(headers, oldHeaders) || !reflect.DeepEqual(parsed, oldBody) || !reflect.DeepEqual(result, oldResult) ||
				err != nil && err.Error() != oldErr.Error() {
				t.Fatal("bounded failure tag changed the parser result or accepted a refused input", mode, d, err, oldErr)
			}
		})
	}
}

func TestProviderFirstFailureDiagnosticCannotRenewAnOriginalInvocation(t *testing.T) {
	for _, mode := range []string{"normal", "closed-connection"} {
		t.Run(mode, func(t *testing.T) {
			s, o := providerFirstOperationFixture(t)
			conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
			if mode == "closed-connection" {
				if err := conn.Close(); err != nil {
					t.Fatal(err)
				}
			}
			held, limits := s.stored, o.plan
			out, err := o.executeCandidateWithConnection(context.Background(), conn)
			if mode == "normal" {
				if err != nil || out == nil || o.diagnostic.Tag != providerFirstFailureUnclassified || o.diagnostic.Reason != providerFirstReasonUnknown || o.diagnostic.Frozen ||
					o.diagnostic.RequestWritten != len(providerFirstOperationRequestWire) || o.diagnostic.ResponseRead <= 0 ||
					s.failed || o.execution.failed ||
					o.state != providerFirstOperationDone || s.stored != held || o.plan != limits {
					t.Fatal("diagnostics changed a normal original operation", err)
				}
				return
			}
			d := o.diagnostic
			if err == nil || out != nil || d.Tag != providerFirstFailureDeadline || d.Reason != providerFirstReasonRefused || !d.Frozen ||
				d.Failed || d.Closed || o.execution.actual[0] != 1 || o.execution.actual[1] != 0 || o.execution.actual[10] == 0 || o.execution.upper != limits.reservation.execution ||
				d.RequestWritten != 0 || d.ResponseRead != 0 || !o.consumption || !o.settlement || s.sequence != 4 ||
				!o.execution.failed || !o.execution.closed || !o.remaining.failed || !o.remaining.closed || o.state != providerFirstOperationFailed ||
				!s.failed || s.activeMeter.Load() != nil || s.stored != held || o.plan != limits ||
				o.integration.ledger.current.totals.bootstrap.outstanding != (candidateWorkVector{}) {
				t.Fatal("failure diagnostic replaced original refusal, reservation, failed S or cleanup", d, err)
			}
			work, remaining, reads := o.execution.actual, o.remaining.actual, s.clock.read
			if again, err := o.executeCandidateWithConnection(context.Background(), nil); err == nil || again != nil || o.diagnostic != d ||
				o.execution.actual != work || o.remaining.actual != remaining || s.clock.read != reads {
				t.Fatal("diagnostic restored a closed original or performed another request")
			}
		})
	}
}
