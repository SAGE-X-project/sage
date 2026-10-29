package registry011

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"testing"
)

func TestProviderIntegrationCertificateBoundsUseFiniteDataOnly(t *testing.T) {
	owner, _, _, _ := providerMeterFixture(t)
	for _, mode := range []string{"normal", "bounds", "meter", "chains", "certificates", "nil-certificate", "bytes", "quota"} {
		t.Run(mode, func(t *testing.T) {
			b := providerTLSBounds{1, 1, 8}
			// Deliberately not a real certificate or trusted TLS state. This
			// bounded data cannot authenticate an origin or reach a connection.
			state := tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{Raw: []byte("unit")}}}}
			meter := newProviderWorkMeter(owner, providerOperationReservation{execution: candidateWorkVector{2: 4, 10: 5}})
			switch mode {
			case "bounds":
				b.chains = 0
			case "meter":
				meter = nil
			case "chains":
				state.VerifiedChains = append(state.VerifiedChains, state.VerifiedChains[0])
			case "certificates":
				state.VerifiedChains[0] = append(state.VerifiedChains[0], state.VerifiedChains[0][0])
			case "nil-certificate":
				state.VerifiedChains[0][0] = nil
			case "bytes":
				b.certificateBytes = 3
			case "quota":
				meter.upper[10] = 4
			}
			ok := providerIntegrationCertificates(state, b, meter)
			if ok != (mode == "normal") {
				t.Fatal("certificate-data bound result", mode)
			}
			if mode == "normal" && (meter.actual[2] != 0 || meter.actual[10] != 5 || meter.actual[0] != 0 || meter.actual[1] != 0) {
				t.Fatal("DER input cost was omitted or invented network work")
			}
		})
	}
}

func TestProviderIntegrationTransportRejectsInvalidDataBeforeDial(t *testing.T) {
	owner, _, _, _ := providerMeterFixture(t)
	meter := newProviderWorkMeter(owner, providerOperationReservation{})
	request, err := newProviderIntegrationRequest("eth_chainId", []any{})
	if err != nil {
		t.Fatal(err)
	}
	limits := providerTestLimits()
	for _, mode := range []string{"context", "connection", "request", "meter", "limits", "wire", "certificates"} {
		t.Run(mode, func(t *testing.T) {
			ctx, conn, req, m, l, b := context.Background(), &tls.Conn{}, request, meter, limits, providerTLSBounds{1, 1, 8}
			switch mode {
			case "context":
				ctx = nil
			case "connection":
				conn = nil
			case "request":
				req = nil
			case "meter":
				m = nil
			case "limits":
				l.Timeout = 0
			case "wire":
				copy := *request
				copy.wire += "extra"
				req = &copy
			case "certificates":
				b.chains = 0
			}
			if sample, err := providerIntegrationCaptureWithBounds(ctx, conn, l, req, m, b); err == nil || sample != nil {
				t.Fatal("invalid finite transport data started a capture")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, selected := range []context.Context{nil, ctx} {
		if conn, err := providerIntegrationDial(selected, limits); err == nil || conn != nil {
			t.Fatal("invalid/cancelled context started a dial")
		}
	}
}

func TestProviderIntegrationNormalOwnedTLSRefusesUnfundedOrClosedCapture(t *testing.T) {
	for _, mode := range []string{"der-budget", "deadline", "closed", "wire-budget", "result-budget"} {
		t.Run(mode, func(t *testing.T) {
			owner, _, _, _ := providerMeterFixture(t)
			request, err := newProviderIntegrationRequest("eth_chainId", []any{})
			if err != nil {
				t.Fatal(err)
			}
			limits := providerTestLimits()
			meter := newProviderWorkMeter(owner, providerOperationReservation{execution: candidateWorkVector{1: 1 << 20, 2: 1 << 20, 10: 1 << 50}})
			conn := providerIntegrationOwnedTLS(t, request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
			ctx, cancel := context.WithTimeout(context.Background(), limits.Timeout)
			defer cancel()
			switch mode {
			case "der-budget":
				meter.upper[10] = 0
			case "deadline":
				ctx = context.Background()
			case "closed":
				_ = conn.Close()
			case "wire-budget":
				// The peer stays normal; this client selected an insufficient
				// fixed receive allowance. No malformed producer is installed.
				limits.HeaderBytes, limits.WireBytes = 1, len(request.wire)+1
			case "result-budget":
				meter.upper[1] = 0
			}
			sample, err := providerIntegrationCapture(ctx, conn, limits, request, meter)
			if err == nil || sample != nil && sample.Complete {
				t.Fatal("unfunded/closed normal capture became complete", mode)
			}
			if mode == "wire-budget" && (sample == nil || sample.CaptureError == "" || !meter.failed) {
				t.Fatal("bounded incomplete response lost its diagnostic refusal")
			}
		})
	}
}
