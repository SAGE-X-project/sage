package didsource010

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sage-x-project/did-go/advanced/sage"
	"github.com/sage-x-project/did-go/consumer"
	"github.com/sage-x-project/did-go/producer"
	"github.com/sage-x-project/did-go/resolution"
)

// These tests exercise the core's complete-record Source with its selected
// public DID dependencies. The injected reader is not an operational registry.
func TestDIDConsumerMetadata(t *testing.T) {
	for _, state := range []string{"active", "created", "deactivated"} {
		t.Run(state, func(t *testing.T) {
			src, reader, _ := newDIDComplete(t)
			var record map[string]any
			if err := json.Unmarshal(reader.snapshot.Record, &record); err != nil {
				t.Fatal(err)
			}
			record["state"] = state
			raw, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			reader.snapshot.Record = raw
			result := sage.Respond(t.Context(), src, sage.Request{DID: mustDID(t, eipID), Accept: sage.MediaTypeEnvelope}, producer.New())
			if result.Problem != nil || reader.calls != 1 {
				t.Fatalf("response %+v; reads=%d", result, reader.calls)
			}
			checkConsumerMetadata(t, result.Body, state == "deactivated")
		})
	}
}

func checkConsumerMetadata(t *testing.T, raw []byte, deactivated bool) {
	t.Helper()
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(envelope["didDocumentMetadata"], &metadata); err != nil {
		t.Fatal(err)
	}
	want := "false"
	if deactivated {
		want = "true"
	}
	if string(metadata["deactivated"]) != want || string(metadata["versionId"]) != `"3"` || metadata["sage"] == nil {
		t.Fatalf("metadata presence or provenance lost: %s", envelope["didDocumentMetadata"])
	}
}

func TestDIDConsumerAcceptDiagnostics(t *testing.T) {
	for _, name := range []string{"supported", "unsupported", "malformed", "extension", "source-failure"} {
		t.Run(name, func(t *testing.T) {
			src, reader, _ := newDIDComplete(t)
			r, err := sage.NewResolver(src, consumer.New(), producer.New())
			if err != nil {
				t.Fatal(err)
			}
			opts := resolution.ResolutionOptions{Accept: "application/did"}
			var want sage.Code
			var standard resolution.Code
			reads := 0
			typed := false
			switch name {
			case "supported":
				reads = 1
			case "unsupported":
				opts.Accept = "text/html"
				want, standard, typed = sage.CodeRepresentationUnsupported, resolution.CodeFeatureNotSupported, true
			case "malformed":
				opts.Accept = "application/did;q=invalid"
				standard = resolution.CodeInvalidOptions
			case "extension":
				opts.Extra = map[string]any{"unknown": true}
				standard = resolution.CodeFeatureNotSupported
			case "source-failure":
				reader.err = errors.New("fixture reader unavailable")
				want, standard, reads = sage.CodeRecordUnreachable, resolution.CodeInternalError, 1
			}
			result, code := r.ResolveWithCode(t.Context(), mustDID(t, eipID), opts)
			if code != want || reader.calls != reads {
				t.Fatalf("code=%s reads=%d; want %s/%d", code, reader.calls, want, reads)
			}
			problem := result.ResolutionMetadata.Error
			if standard == "" {
				if problem != nil || result.Document == nil || !result.DocumentMetadata.DeactivatedPresent || result.DocumentMetadata.Deactivated {
					t.Fatalf("success %+v", result)
				}
			} else if problem == nil || problem.Type != standard.TypeURL() || problem.IsAcceptNotSupported() != typed || result.Document != nil || !result.DocumentMetadata.IsEmpty() {
				t.Fatalf("failure %+v", result)
			}
		})
	}
}

func TestDIDConsumerTLS(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			for _, name := range []string{"envelope", "core10", "did11", "unsupported", "malformed", "source-failure"} {
				t.Run(name, func(t *testing.T) {
					src, reader, _ := newDIDComplete(t)
					r, err := sage.NewResolver(src, consumer.New(), producer.New())
					if err != nil {
						t.Fatal(err)
					}
					server := httptest.NewTLSServer(sage.NewHandler(r, "/identifiers/"))
					defer server.Close()
					accept, contentType, status, reads := "application/did-resolution", "application/did-resolution", http.StatusOK, 1
					wantCode, wantType := "", ""
					switch name {
					case "core10":
						accept, contentType = "application/did+json", "application/did+json"
					case "did11":
						accept, contentType = "application/did", "application/did"
					case "unsupported":
						accept, contentType, status, reads = "text/html", "application/problem+json", http.StatusNotImplemented, 0
						wantCode, wantType = "representation.unsupported", resolution.CodeFeatureNotSupported.TypeURL()
					case "malformed":
						accept, contentType, status, reads = "application/did;q=invalid", "application/problem+json", http.StatusBadRequest, 0
						wantType = resolution.CodeInvalidOptions.TypeURL()
					case "source-failure":
						reader.err = errors.New("fixture reader unavailable")
						accept, contentType, status = "application/did", "application/problem+json", http.StatusInternalServerError
						wantCode, wantType = "record.unreachable", resolution.CodeInternalError.TypeURL()
					}
					req, err := http.NewRequestWithContext(t.Context(), method, server.URL+"/identifiers/"+eipID, strings.NewReader("{}"))
					if err != nil {
						t.Fatal(err)
					}
					req.Header.Set("Accept", accept)
					req.Header.Set("Content-Type", "application/json")
					res, err := server.Client().Do(req)
					if err != nil {
						t.Fatal(err)
					}
					defer func() {
						if err := res.Body.Close(); err != nil {
							t.Error(err)
						}
					}()
					var wire map[string]json.RawMessage
					if err := json.NewDecoder(res.Body).Decode(&wire); err != nil {
						t.Fatal(err)
					}
					if res.StatusCode != status || res.Header.Get("Content-Type") != contentType || reader.calls != reads {
						t.Fatalf("status=%d content-type=%s reads=%d; want %d/%s/%d: %v", res.StatusCode, res.Header.Get("Content-Type"), reader.calls, status, contentType, reads, wire)
					}
					if wantType != "" {
						if string(wire["type"]) != `"`+wantType+`"` || (wantCode == "" && wire["sageCode"] != nil) || (wantCode != "" && string(wire["sageCode"]) != `"`+wantCode+`"`) || wire["didDocument"] != nil || wire["didDocumentMetadata"] != nil {
							t.Fatalf("problem %v", wire)
						}
					} else if name == "envelope" {
						raw, err := json.Marshal(wire)
						if err != nil {
							t.Fatal(err)
						}
						checkConsumerMetadata(t, raw, false)
					} else if string(wire["id"]) != `"`+eipID+`"` || wire["authentication"] == nil || wire["keyAgreement"] == nil {
						t.Fatalf("document %v", wire)
					}
				})
			}
		})
	}
}
