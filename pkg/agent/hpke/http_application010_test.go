package hpke

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type applicationReads010 struct {
	control *completionControl
	calls   atomic.Int64
	mu      sync.Mutex
	edit    func()
}

func (s *applicationReads010) setEdit(edit func()) { s.mu.Lock(); s.edit = edit; s.mu.Unlock() }
func (s *applicationReads010) Read(ctx context.Context, did string) (registry010.Snapshot, error) {
	s.calls.Add(1)
	s.mu.Lock()
	edit := s.edit
	s.edit = nil
	s.mu.Unlock()
	if edit != nil {
		edit()
	}
	return s.control.Read(ctx, did)
}

type applicationFixture010 struct {
	left, right       *HTTPApplicationOwner010
	oldLeft, oldRight *AuthenticatedCompletion010
	control           *completionControl
	tuple             map[string]string
	reads             *applicationReads010
}

func applicationPair010(t *testing.T, httpHandshake bool) applicationFixture010 {
	return applicationPairTarget010(t, httpHandshake, "https://agent.example/messages")
}
func applicationPairTarget010(t *testing.T, httpHandshake bool, target string) applicationFixture010 {
	t.Helper()
	a, b, c := completionPair(t)
	reads := &applicationReads010{control: c}
	// Install the passive counting Source before any handshake observation.
	for _, endpoint := range []*CompletionEndpoint010{a, b} {
		journal, err := registry010.OpenJournal(filepath.Join(t.TempDir(), "registry"), true)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = journal.Close() })
		gate, err := registry010.NewGate(registry010.Config{Source: "fixture-authority", Registry: completionRegistry, Network: "local"}, reads, c, journal)
		if err != nil {
			t.Fatal(err)
		}
		endpoint.registry = gate
	}
	ctx := context.Background()
	var left, right *AuthenticatedCompletion010
	if httpHandshake {
		for _, endpoint := range []*CompletionEndpoint010{a, b} {
			if err := endpoint.BindHTTP(target); err != nil {
				t.Fatal(err)
			}
		}
		pending, hello, err := a.StartHTTP(ctx, completionBob, completionBob+"#signing-1", 300)
		if err != nil {
			t.Fatal(err)
		}
		var reply HTTPMessage010
		right, reply, err = b.RespondHTTP(ctx, hello, 300)
		if err != nil {
			t.Fatal(err)
		}
		left, err = pending.CompleteHTTP(ctx, reply)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		pending, hello, err := a.Start(ctx, completionBob, completionBob+"#signing-1", 300)
		if err != nil {
			t.Fatal(err)
		}
		var reply []byte
		right, reply, err = b.Respond(ctx, hello, 300)
		if err != nil {
			t.Fatal(err)
		}
		left, err = pending.Complete(ctx, reply)
		if err != nil {
			t.Fatal(err)
		}
	}
	tuple := left.Tuple()
	first, err := left.TakeHTTPApplication(target)
	if err != nil {
		t.Fatal(err)
	}
	second, err := right.TakeHTTPApplication(target)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(first.Close)
	t.Cleanup(second.Close)
	return applicationFixture010{first, second, left, right, c, tuple, reads}
}
func TestHTTPApplicationRoundTrip010(t *testing.T) {
	for _, httpHandshake := range []bool{false, true} {
		for _, kind := range []string{"absent", "empty", "a2a", "unicode", "transport-error"} {
			t.Run(fmt.Sprintf("http=%t/%s", httpHandshake, kind), func(t *testing.T) {
				f := applicationPair010(t, httpHandshake)
				ctx := context.Background()
				var metadata map[string]string
				switch kind {
				case "empty":
					metadata = map[string]string{}
				case "a2a":
					metadata = map[string]string{"a2a_version": "1.0", "a2a_extensions": "https://sage-x-project.github.io/sage-spec/a2a/0.10.0/extension"}
				case "unicode":
					metadata = map[string]string{"note": "\uD55C\uAE00 \U0001f600 \x00 / \\\""}
				}
				want := []byte("inert JSON-RPC application bytes")
				wire, err := f.left.SealRequest(ctx, want, metadata, 300)
				if err != nil {
					t.Fatal(err)
				}
				request, err := f.right.OpenRequest(ctx, wire)
				if err != nil || request == nil {
					t.Fatalf("request: %v", err)
				}
				if !bytes.Equal(request.Payload(), want) || !equalApplicationMetadata010(request.Metadata(), metadata) {
					t.Fatal("request content changed")
				}
				if request.Sender() != completionAlice || request.Recipient() != completionBob || request.KeyID() != completionAlice+"#signing-1" || request.ContextID() != f.tuple["ctx"] || request.SessionID() != f.tuple["sid"] || !uuid010.MatchString(request.ID()) {
					t.Fatal("request identity changed")
				}
				if _, err = f.right.OpenRequest(ctx, wire); err == nil {
					t.Fatal("request replay")
				}
				returned := request.Payload()
				returned[0] ^= 1
				if !bytes.Equal(request.Payload(), want) {
					t.Fatal("payload storage alias")
				}
				copyMetadata := request.Metadata()
				if copyMetadata != nil {
					copyMetadata["extra"] = "changed"
					if !equalApplicationMetadata010(request.Metadata(), metadata) {
						t.Fatal("metadata alias")
					}
				}
				replyMetadata := map[string]string{"result_note": "independently authenticated response metadata"}
				success, code, status := true, "", 200
				if kind == "transport-error" {
					success, code, status = false, "policy_denied", 403
				}
				response, err := f.right.SealResponse(ctx, request.ID(), []byte("inert response"), replyMetadata, success, code, 300, status)
				if err != nil {
					t.Fatal(err)
				}
				result, err := f.left.OpenResponse(ctx, response)
				if err != nil || result == nil {
					t.Fatalf("response: %v", err)
				}
				if !bytes.Equal(result.Data(), []byte("inert response")) || !equalApplicationMetadata010(result.Metadata(), replyMetadata) || result.Success() != success || result.Error() != code || result.InReplyTo() != request.ID() {
					t.Fatal("response content/correlation changed")
				}
				if result.Sender() != completionBob || result.Recipient() != completionAlice || result.KeyID() != completionBob+"#signing-1" || result.ContextID() != f.tuple["ctx"] || result.SessionID() != f.tuple["sid"] || result.ID() == request.ID() || !uuid010.MatchString(result.ID()) {
					t.Fatal("response identity changed")
				}
				if _, err = f.left.OpenResponse(ctx, response); err == nil {
					t.Fatal("terminal response replay")
				}
				if _, err = f.right.SealResponse(ctx, request.ID(), nil, nil, true, "", 300, 200); err == nil {
					t.Fatal("second response emission")
				}
				if err = f.left.Check(ctx); err != nil {
					t.Fatal(err)
				}
				if _, err = f.left.Observe(ctx); err != nil {
					t.Fatal(err)
				}
				if _, err = f.left.LocalNow(); err != nil {
					t.Fatal(err)
				}
				local, peer, err := f.left.Participants()
				if err != nil || local != completionAlice || peer != completionBob {
					t.Fatal("participant diagnostics")
				}
				local, peer, err = f.right.Participants()
				if err != nil || local != completionBob || peer != completionAlice {
					t.Fatal("responder participant diagnostics")
				}
				// Verify signed bodies retain absence versus a present empty object.
				var body map[string]json.RawMessage
				if json.Unmarshal(wire.Body, &body) != nil {
					t.Fatal("body")
				}
				_, present := body["metadata"]
				if present != (metadata != nil) {
					t.Fatal("metadata presence changed")
				}
			})
		}
	}
}
func equalApplicationMetadata010(a, b map[string]string) bool {
	return (a == nil) == (b == nil) && bytes.Equal(canon010(a), canon010(b))
}
func TestHTTPApplicationMetadataAdmission010(t *testing.T) {
	cases := map[string]bool{
		`{}`: true, `{"a":""}`: true, `{"a0_":"value"}`: true, `{"note":"\uD83D\uDE00"}`: true, `{"note":"\\ud800"}`: true, `{"note":"\ufffd"}`: true,
		`null`: false, `[]`: false, `true`: false, `0`: false, `{"a":null}`: false, `{"a":1}`: false, `{"a":{}}`: false, `{"a":[]}`: false, `{"a":false}`: false,
		`{"a":"x","a":"y"}`: false, `{"a":"x","\u0061":"y"}`: false, `{"A":"x"}`: false, `{"_a":"x"}`: false, `{"1a":"x"}`: false, `{"a-":"x"}`: false, `{"a ":"x"}`: false, `{"":"x"}`: false,
		`{"a":"\ud800"}`: false, `{"a":"\udc00"}`: false, `{"a":"\ud800\ud800"}`: false, `{"a":"\uqqqq"}`: false, `{"a":"\u123"}`: false, `{"a":"x"} {}`: false, `{"a":"x"`: false, `{"a":`: false,
	}
	for raw, want := range cases {
		t.Run(raw, func(t *testing.T) {
			_, err := readApplicationMetadata010([]byte(raw))
			if (err == nil) != want {
				t.Fatalf("admission %t: %v", want, err)
			}
		})
	}
	for _, size := range []int{32, 33} {
		metadata := map[string]string{}
		for i := 0; i < size; i++ {
			metadata[fmt.Sprintf("k%d", i)] = "x"
		}
		if _, err := copyApplicationMetadata010(metadata); (err == nil) != (size == 32) {
			t.Fatal("entry count", err)
		}
		raw, _ := json.Marshal(metadata)
		if _, err := readApplicationMetadata010(raw); (err == nil) != (size == 32) {
			t.Fatal("decoded entry count", err)
		}
	}
	for _, size := range []int{1024, 1025} {
		if _, err := copyApplicationMetadata010(map[string]string{"a": strings.Repeat("x", size)}); (err == nil) != (size == 1024) {
			t.Fatal("value byte limit", err)
		}
	}
	for _, size := range []int{32, 33} {
		if _, err := copyApplicationMetadata010(map[string]string{strings.Repeat("a", size): "x"}); (err == nil) != (size == 32) {
			t.Fatal("key byte limit", err)
		}
	}
	if _, err := copyApplicationMetadata010(map[string]string{"a": string([]byte{255})}); err == nil {
		t.Fatal("invalid UTF8")
	}
	if _, err := readApplicationMetadata010([]byte("{\"a\":\"" + string([]byte{255}) + "\"}")); err == nil {
		t.Fatal("raw invalid UTF8")
	}
	if _, err := readApplicationMetadata010(bytes.Repeat([]byte(" "), 32769)); err == nil {
		t.Fatal("raw admission bound")
	}
	for _, size := range []int{995, 996} {
		value := map[string]string{"a": strings.Repeat("x", 1024), "b": strings.Repeat("x", 1024), "c": strings.Repeat("x", 1024), "d": strings.Repeat("x", size)}
		if len(canon010(value)) != 4096+size-995 {
			t.Fatal("authored JCS bound")
		}
		if _, err := copyApplicationMetadata010(value); (err == nil) != (size == 995) {
			t.Fatal("JCS metadata limit", err)
		}
	}
}

// Informational values below are data in unit scenarios, never alternate authority.
func TestHTTPApplicationExclusiveOwnership010(t *testing.T) {
	originalLeft, originalRight, _ := recordPair010(t, 0)
	oldCopy := *originalLeft
	left, err := originalLeft.TakeHTTPApplication("https://agent.example/messages")
	if err != nil {
		t.Fatal(err)
	}
	right, err := originalRight.TakeHTTPApplication("https://agent.example/messages")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(left.Close)
	t.Cleanup(right.Close)
	ctx := context.Background()
	for _, old := range []*AuthenticatedCompletion010{originalLeft, &oldCopy, originalRight} {
		if _, err = old.TakeHTTPApplication("https://agent.example/messages"); err == nil {
			t.Fatal("old transfer")
		}
		if _, err = old.TakeNonHTTP(); err == nil {
			t.Fatal("old alternate owner")
		}
		if err = old.BindHTTP("https://different.example/messages"); err == nil {
			t.Fatal("old rebind")
		}
		if _, err = old.SealRequest(ctx, nil, 300); err == nil {
			t.Fatal("old bare send")
		}
		if _, err = old.OpenRequest(ctx, nil); err == nil {
			t.Fatal("old bare receive")
		}
		if _, err = old.SealHTTPRequest(ctx, nil, 300); err == nil {
			t.Fatal("old HTTP send")
		}
		if _, err = old.OpenHTTPRequest(ctx, HTTPMessage010{}); err == nil {
			t.Fatal("old HTTP receive")
		}
		if _, err = old.SealResponse(ctx, "", nil, true, "", 300); err == nil {
			t.Fatal("old bare response send")
		}
		if _, err = old.OpenResponse(ctx, nil); err == nil {
			t.Fatal("old bare response receive")
		}
		if _, err = old.SealHTTPResponse(ctx, "", nil, true, "", 300, 200); err == nil {
			t.Fatal("old HTTP response send")
		}
		if _, err = old.OpenHTTPResponse(ctx, HTTPMessage010{}); err == nil {
			t.Fatal("old HTTP response receive")
		}
		if err = old.Check(ctx); err == nil {
			t.Fatal("old current check")
		}
		old.Close()
	}
	// Even accidental internal use of the moved session's raw APIs must fail.
	for _, moved := range []*AuthenticatedCompletion010{left.session, right.session} {
		if _, err = moved.TakeHTTPApplication("https://agent.example/messages"); err == nil {
			t.Fatal("second transfer")
		}
		if _, err = moved.SealHTTPRequest(ctx, nil, 300); err == nil {
			t.Fatal("raw application send")
		}
		if _, err = moved.OpenHTTPRequest(ctx, HTTPMessage010{}); err == nil {
			t.Fatal("raw application receive")
		}
		if _, err = moved.SealHTTPResponse(ctx, "", nil, true, "", 300, 200); err == nil {
			t.Fatal("raw application response send")
		}
		if _, err = moved.OpenHTTPResponse(ctx, HTTPMessage010{}); err == nil {
			t.Fatal("raw application response receive")
		}
	}
	wire, err := left.SealRequest(ctx, []byte("owned"), map[string]string{"note": "inert"}, 300)
	if err != nil {
		t.Fatal(err)
	}
	copyRight := *right
	view, err := copyRight.OpenRequest(ctx, wire)
	if err != nil || view == nil || !bytes.Equal(view.Payload(), []byte("owned")) {
		t.Fatal("old close erased new owner", err)
	}
	if _, err = right.OpenRequest(ctx, wire); err == nil {
		t.Fatal("copied owner reset replay")
	}
	copyRight.Close()
	if err = right.Check(ctx); err == nil {
		t.Fatal("copied owner reopened close")
	}
}
func TestHTTPApplicationTransferRefusals010(t *testing.T) {
	var absent *AuthenticatedCompletion010
	if _, err := absent.TakeHTTPApplication("https://agent.example/messages"); err == nil {
		t.Fatal("nil transfer")
	}
	if _, err := (&AuthenticatedCompletion010{}).TakeHTTPApplication("https://agent.example/messages"); err == nil {
		t.Fatal("zero transfer")
	}
	for _, kind := range []string{"invalid-target", "target-mismatch", "closed", "endpoint-retired", "record-used", "shadow-binding"} {
		t.Run(kind, func(t *testing.T) {
			a, _, _ := recordPair010(t, 0)
			target := "https://agent.example/messages"
			switch kind {
			case "invalid-target":
				target = "http://agent.example/messages"
			case "target-mismatch":
				if a.BindHTTP("https://different.example/messages") != nil {
					t.Fatal("bind control")
				}
			case "closed":
				a.Close()
			case "endpoint-retired":
				a.endpoint.Close()
			case "record-used":
				if _, err := a.SealRequest(context.Background(), nil, 300); err != nil {
					t.Fatal(err)
				}
			case "shadow-binding":
				shadow := *a
				if shadow.BindHTTP("https://different.example/messages") != nil {
					t.Fatal("shadow bind control")
				}
			}
			if owner, err := a.TakeHTTPApplication(target); err == nil || owner != nil {
				t.Fatal("invalid transfer")
			}
			if kind == "invalid-target" {
				owner, err := a.TakeHTTPApplication("https://agent.example/messages")
				if err != nil {
					t.Fatal("failed transfer consumed ownership", err)
				}
				owner.Close()
			}
		})
	}
}
func TestHTTPApplicationLegacyMetadataRefusal010(t *testing.T) {
	for _, metadata := range []map[string]string{{}, {"a2a_version": "1.0"}} {
		a, b, _ := recordPair010(t, 0)
		owner, err := a.TakeHTTPApplication("https://agent.example/messages")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(owner.Close)
		if err = b.BindHTTP("https://agent.example/messages"); err != nil {
			t.Fatal(err)
		}
		wire, err := owner.SealRequest(context.Background(), []byte("inert"), metadata, 300)
		if err != nil {
			t.Fatal(err)
		}
		before := len(b.endpoint.replay.(*completionReplay).seen)
		if plain, err := b.OpenHTTPRequest(context.Background(), wire); err == nil || plain != nil {
			t.Fatal("legacy silently discarded metadata")
		}
		if len(b.endpoint.replay.(*completionReplay).seen) != before || b.State() != "RESPONSE_SENT" {
			t.Fatal("legacy refusal published acceptance")
		}
		// The same cryptographically valid message is accepted only after explicit transfer.
		receiver, err := b.TakeHTTPApplication("https://agent.example/messages")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(receiver.Close)
		view, err := receiver.OpenRequest(context.Background(), wire)
		if err != nil || view == nil || !equalApplicationMetadata010(view.Metadata(), metadata) {
			t.Fatal("explicit mode positive control", err)
		}
	}
}
func TestHTTPApplicationEmptyAndCanceledOwners010(t *testing.T) {
	f := applicationPair010(t, true)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, owner := range []*HTTPApplicationOwner010{nil, {}, {session: &AuthenticatedCompletion010{}}, f.left} {
		contexts := []context.Context{nil, canceled}
		if owner != f.left {
			contexts = append(contexts, context.Background())
		}
		for _, ctx := range contexts {
			if err := owner.Check(ctx); err == nil {
				t.Fatal("invalid check")
			}
			if _, err := owner.Observe(ctx); err == nil {
				t.Fatal("invalid observation")
			}
			if _, err := owner.SealRequest(ctx, nil, nil, 300); err == nil {
				t.Fatal("invalid request send")
			}
			if _, err := owner.OpenRequest(ctx, HTTPMessage010{}); err == nil {
				t.Fatal("invalid request receive")
			}
			if _, err := owner.SealResponse(ctx, "", nil, nil, true, "", 300, 200); err == nil {
				t.Fatal("invalid response send")
			}
			if _, err := owner.OpenResponse(ctx, HTTPMessage010{}); err == nil {
				t.Fatal("invalid response receive")
			}
		}
		if owner != f.left {
			if _, _, err := owner.Participants(); err == nil {
				t.Fatal("invalid participants")
			}
			if _, err := owner.LocalNow(); err == nil {
				t.Fatal("invalid clock")
			}
			owner.Close()
		}
	}
	if _, err := f.left.SealRequest(context.Background(), make([]byte, 16349), nil, 300); err == nil {
		t.Fatal("oversize request")
	}
	if _, err := f.right.SealResponse(context.Background(), "", make([]byte, 16349), nil, true, "", 300, 200); err == nil {
		t.Fatal("oversize response")
	}
	for _, message := range []HTTPMessage010{{Body: make([]byte, 32769)}, {Headers: make([][2]string, 257)}} {
		if _, err := f.right.OpenRequest(context.Background(), message); err == nil {
			t.Fatal("oversize request admission")
		}
		if _, err := f.left.OpenResponse(context.Background(), message); err == nil {
			t.Fatal("oversize response admission")
		}
	}
	if _, err := f.right.OpenRequest(context.Background(), HTTPMessage010{}); err == nil {
		t.Fatal("empty request")
	}
	if _, err := f.left.OpenResponse(context.Background(), HTTPMessage010{}); err == nil {
		t.Fatal("empty response")
	}
	f.left.Close()
	if _, err := f.left.SealRequest(context.Background(), nil, nil, 300); err == nil {
		t.Fatal("closed request")
	}
	if _, err := f.left.LocalNow(); err == nil {
		t.Fatal("closed clock")
	}
}

func TestHTTPApplicationOwnedInputs010(t *testing.T) {
	f := applicationPair010(t, true)
	metadata := map[string]string{"note": "original"}
	payload := []byte("original bytes")
	f.reads.setEdit(func() { metadata["note"] = "changed"; payload[0] = 'X' })
	wire, err := f.left.SealRequest(context.Background(), payload, metadata, 300)
	if err != nil {
		t.Fatal(err)
	}
	originalWire := append([]byte(nil), wire.Body...)
	f.reads.setEdit(func() { wire.Body[0] ^= 1; wire.Headers[0][1] = "changed" })
	view, err := f.right.OpenRequest(context.Background(), wire)
	if err != nil || view == nil || !bytes.Equal(view.Payload(), []byte("original bytes")) || view.Metadata()["note"] != "original" {
		t.Fatal("provider changed owned request", err)
	}
	if bytes.Equal(wire.Body, originalWire) {
		t.Fatal("input mutation control did not execute")
	}
	responseData := []byte("original result")
	responseMetadata := map[string]string{"note": "original response"}
	f.reads.setEdit(func() { responseData[0] = 'X'; responseMetadata["note"] = "changed" })
	reply, err := f.right.SealResponse(context.Background(), view.ID(), responseData, responseMetadata, true, "", 300, 200)
	if err != nil {
		t.Fatal(err)
	}
	beforeReply := append([]byte(nil), reply.Body...)
	f.reads.setEdit(func() { reply.Body[0] ^= 1; reply.Headers[0][1] = "changed" })
	result, err := f.left.OpenResponse(context.Background(), reply)
	if err != nil || result == nil || !bytes.Equal(result.Data(), []byte("original result")) || result.Metadata()["note"] != "original response" {
		t.Fatal("provider changed owned response or retained request correlation", err)
	}
	if bytes.Equal(reply.Body, beforeReply) {
		t.Fatal("response mutation control did not execute")
	}
	returned := result.Metadata()
	returned["note"] = "caller changed"
	data := result.Data()
	data[0] = 'X'
	if result.Metadata()["note"] != "original response" || !bytes.Equal(result.Data(), []byte("original result")) {
		t.Fatal("response aliases internal view")
	}
}
func TestHTTPApplicationInvalidOutgoingMetadataHasNoSourceRead010(t *testing.T) {
	for _, metadata := range []map[string]string{{"Bad": "value"}, {"a": strings.Repeat("x", 1025)}, {"a": string([]byte{255})}} {
		f := applicationPair010(t, true)
		before := f.reads.calls.Load()
		if _, err := f.left.SealRequest(context.Background(), nil, metadata, 300); err == nil {
			t.Fatal("invalid request metadata")
		}
		if _, err := f.right.SealResponse(context.Background(), "", nil, metadata, true, "", 300, 200); err == nil {
			t.Fatal("invalid response metadata")
		}
		if f.reads.calls.Load() != before {
			t.Fatal("malformed metadata queried Source")
		}
	}
}
func TestHTTPApplicationRecordAuthorityRefusals010(t *testing.T) {
	for _, mode := range []string{"revoke-init", "revoke-resp", "source-error"} {
		for _, response := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/response=%t", mode, response), func(t *testing.T) {
				f := applicationPair010(t, true)
				metadata := map[string]string{"did_document": "display only", "canonical_id": "did:example:informational"}
				request, err := f.left.SealRequest(context.Background(), []byte("inert"), metadata, 300)
				if err != nil {
					t.Fatal(err)
				}
				receiver := f.right
				message := request
				if response {
					view, err := f.right.OpenRequest(context.Background(), request)
					if err != nil {
						t.Fatal(err)
					}
					message, err = f.right.SealResponse(context.Background(), view.ID(), []byte("inert response"), metadata, true, "", 300, 200)
					if err != nil {
						t.Fatal(err)
					}
					receiver = f.left
				}
				before := len(receiver.session.endpoint.replay.(*completionReplay).seen)
				reads := f.reads.calls.Load()
				f.control.mode = mode
				if response {
					if result, err := receiver.OpenResponse(context.Background(), message); err == nil || result != nil {
						t.Fatal("unavailable authority released response")
					}
				} else {
					if result, err := receiver.OpenRequest(context.Background(), message); err == nil || result != nil {
						t.Fatal("unavailable authority released request")
					}
				}
				if receiver.session.State() != "CLOSED" || len(receiver.session.endpoint.replay.(*completionReplay).seen) != before || f.reads.calls.Load() <= reads {
					t.Fatal("missing fresh refusal or premature replay acceptance")
				}
			})
		}
	}
}
func TestHTTPApplicationAggregateAADRefusesBeforeSequence010(t *testing.T) {
	f := applicationPair010(t, true)
	large := map[string]string{"a": strings.Repeat("x", 1024), "b": strings.Repeat("x", 1024), "c": strings.Repeat("x", 1024), "d": strings.Repeat("x", 995)}
	if len(canon010(large)) != 4096 {
		t.Fatal("metadata boundary control")
	}
	if _, err := f.left.SealRequest(context.Background(), nil, large, 300); err == nil {
		t.Fatal("aggregate request AAD accepted")
	}
	message, err := f.left.SealRequest(context.Background(), nil, map[string]string{}, 300)
	if err != nil {
		t.Fatal(err)
	}
	if applicationSequence010(t, message, "payload") != 0 {
		t.Fatal("oversize request consumed sequence")
	}
	request, err := f.right.OpenRequest(context.Background(), message)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.right.SealResponse(context.Background(), request.ID(), nil, large, true, "", 300, 200); err == nil {
		t.Fatal("aggregate response AAD accepted")
	}
	reply, err := f.right.SealResponse(context.Background(), request.ID(), nil, map[string]string{}, true, "", 300, 200)
	if err != nil {
		t.Fatal(err)
	}
	if applicationSequence010(t, reply, "data") != 0 {
		t.Fatal("oversize response consumed sequence")
	}
	if _, err := f.left.OpenResponse(context.Background(), reply); err != nil {
		t.Fatal(err)
	}
}
func applicationSequence010(t *testing.T, message HTTPMessage010, field string) uint64 {
	t.Helper()
	var body map[string]json.RawMessage
	if json.Unmarshal(message.Body, &body) != nil {
		t.Fatal("wire JSON")
	}
	record, err := base64.RawURLEncoding.DecodeString(str010(body, field))
	if err != nil || len(record) < 36 {
		t.Fatal("wire record")
	}
	return binary.BigEndian.Uint64(record[:8])
}

// All mutation controls below are confined to owned in-process unit fixtures.
// They are not transport tools and never contact external services.
func TestHTTPApplicationMetadataBindings010(t *testing.T) {
	for _, response := range []bool{false, true} {
		for _, kind := range []string{"null", "duplicate", "nonstring", "surrogate", "inner-signature", "outer-signature", "aead", "recipient", "session", "request-hash"} {
			if !response && kind == "request-hash" {
				continue
			}
			t.Run(fmt.Sprintf("response=%t/%s", response, kind), func(t *testing.T) {
				f := applicationPair010(t, true)
				ctx := context.Background()
				original, err := f.left.SealRequest(ctx, []byte("inert request"), map[string]string{"note": "original"}, 300)
				if err != nil {
					t.Fatal(err)
				}
				receiver, sender, seed := f.right, f.left, byte(1)
				var requestContext *httpContext010
				if response {
					request, err := f.right.OpenRequest(ctx, original)
					if err != nil {
						t.Fatal(err)
					}
					requestContext = f.left.session.sent[request.ID()].http
					original, err = f.right.SealResponse(ctx, request.ID(), []byte("inert response"), map[string]string{"note": "original"}, true, "", 300, 200)
					if err != nil {
						t.Fatal(err)
					}
					receiver, sender, seed = f.left, f.right, 2
				}
				var fields map[string]json.RawMessage
				if json.Unmarshal(original.Body, &fields) != nil {
					t.Fatal("control JSON")
				}
				switch kind {
				case "null":
					fields["metadata"] = json.RawMessage(`null`)
				case "duplicate":
					fields["metadata"] = json.RawMessage(`{"note":"one","\u006eote":"two"}`)
				case "nonstring":
					fields["metadata"] = json.RawMessage(`{"note":1}`)
				case "surrogate":
					fields["metadata"] = json.RawMessage(`{"note":"\ud800"}`)
				default:
					fields["metadata"] = json.RawMessage(`{"note":"changed"}`)
				}
				if kind == "recipient" {
					peer := completionAlice
					if response {
						peer = completionBob
					}
					fields["recipient"], _ = json.Marshal(peer)
				}
				if kind == "session" {
					fields["session_id"], _ = json.Marshal(base64.RawURLEncoding.EncodeToString(make([]byte, 16)))
				}
				if kind == "request-hash" {
					fields["request_hash"], _ = json.Marshal(base64.RawURLEncoding.EncodeToString(make([]byte, 32)))
				}
				mutated := original
				mutated.Headers = append([][2]string(nil), original.Headers...)
				mutated.Body, err = json.Marshal(fields)
				if err != nil {
					t.Fatal(err)
				}
				// Keep malformed nested bytes intact: this stage intentionally has no valid proof.
				if kind == "null" || kind == "duplicate" || kind == "nonstring" || kind == "surrogate" {
					for i := range mutated.Headers {
						if mutated.Headers[i][0] == "content-digest" {
							mutated.Headers[i][1] = httpDigest010(mutated.Body)
						}
					}
				} else {
					var envelope map[string]any
					if json.Unmarshal(mutated.Body, &envelope) != nil {
						t.Fatal("unit envelope")
					}
					if kind != "inner-signature" {
						delete(envelope, "signature")
					}
					domain := "sage-wire-request|0.10.0\n"
					if response {
						domain = "sage-wire-response|0.10.0\n"
					}
					key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32))
					if kind != "inner-signature" {
						mutated.Body = signKey010(envelope, domain, key)
					}
					var signed map[string]json.RawMessage
					_ = json.Unmarshal(mutated.Body, &signed)
					pinned := sender.session.a.Signing()
					if response {
						pinned = sender.session.b.Signing()
					}
					if kind != "inner-signature" && verifyWire010(signed, response, pinned) != nil {
						t.Fatal("valid inner proof control")
					}
					// The outer-only control keeps its original proof. The AEAD control has
					// valid inner and HTTP signatures but the unchanged ciphertext's old AAD.
					if kind == "outer-signature" {
						for i := range mutated.Headers {
							if mutated.Headers[i][0] == "content-digest" {
								mutated.Headers[i][1] = httpDigest010(mutated.Body)
							}
						}
					} else {
						mutated, err = signHTTP010(ctx, sender.session.endpoint, sender.session.httpTarget, sender.session.httpAuthority, mutated.Body, original.Status, requestContext)
						if err != nil {
							t.Fatal(err)
						}
						h, err := httpHeaders010(mutated)
						if err != nil {
							t.Fatal(err)
						}
						input := strings.TrimPrefix(h["signature-input"], "sig1=")
						signature, err := base64.StdEncoding.DecodeString(strings.TrimSuffix(strings.TrimPrefix(h["signature"], "sig1=:"), ":"))
						if err != nil || !ed25519.Verify(key.Public().(ed25519.PublicKey), httpBase010(mutated, h, input, requestContext), signature) {
							t.Fatal("valid outer proof control")
						}
					}
				}
				beforeReads := f.reads.calls.Load()
				replay := receiver.session.endpoint.replay.(*completionReplay)
				beforeReplay := len(replay.seen)
				if response {
					if view, err := receiver.OpenResponse(ctx, mutated); err == nil || view != nil {
						t.Fatal("invalid response released")
					}
				} else {
					if view, err := receiver.OpenRequest(ctx, mutated); err == nil || view != nil {
						t.Fatal("invalid request released")
					}
				}
				if len(replay.seen) != beforeReplay {
					t.Fatal("invalid message published acceptance")
				}
				if kind != "aead" && kind != "inner-signature" && f.reads.calls.Load() != beforeReads {
					t.Fatal("admission/tuple/proof failure consulted Source")
				}
				if kind == "aead" && f.reads.calls.Load() <= beforeReads {
					t.Fatal("AEAD control did not reach authority validation")
				}
				// A rejected proof/schema/AAD does not release the valid sequence or terminal.
				if response {
					view, err := receiver.OpenResponse(ctx, original)
					if err != nil || view == nil || string(view.Data()) != "inert response" {
						t.Fatal("valid response after refusal", err)
					}
				} else {
					view, err := receiver.OpenRequest(ctx, original)
					if err != nil || view == nil || string(view.Payload()) != "inert request" {
						t.Fatal("valid request after refusal", err)
					}
				}
			})
		}
	}
}

func TestHTTPApplicationExactAADBoundary010(t *testing.T) {
	for _, response := range []bool{false, true} {
		t.Run(fmt.Sprintf("response=%t", response), func(t *testing.T) {
			f := applicationPair010(t, true)
			ctx := context.Background()
			first, err := f.left.SealRequest(ctx, nil, map[string]string{}, 300)
			if err != nil {
				t.Fatal(err)
			}
			request, err := f.right.OpenRequest(ctx, first)
			if err != nil {
				t.Fatal(err)
			}
			control := first
			if response {
				control, err = f.right.SealResponse(ctx, request.ID(), nil, map[string]string{}, true, "", 300, 200)
				if err != nil {
					t.Fatal(err)
				}
			}
			var fields map[string]json.RawMessage
			_ = json.Unmarshal(control.Body, &fields)
			// Envelope IDs/nonces and correlation hashes have fixed lengths. Fill the
			// metadata budget from an actual generated request/response AAD, not estimates.
			base := len(requestAAD010(fields))
			budget := 4033 - base
			metadata := map[string]string{"a": "", "b": "", "c": "", "d": ""}
			remaining := budget - (len(canon010(metadata)) - 2)
			if remaining < 0 || remaining > 4096 {
				t.Fatal("boundary budget")
			}
			for _, key := range []string{"a", "b", "c", "d"} {
				n := remaining
				if n > 1024 {
					n = 1024
				}
				metadata[key] = strings.Repeat("x", n)
				remaining -= n
			}
			if remaining != 0 {
				t.Fatal("metadata budget overflow")
			}
			oversized := map[string]string{}
			for k, v := range metadata {
				oversized[k] = v
			}
			oversized["d"] += "x"
			var accepted HTTPMessage010
			if response {
				next, err := f.left.SealRequest(ctx, nil, nil, 300)
				if err != nil {
					t.Fatal(err)
				}
				view, err := f.right.OpenRequest(ctx, next)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = f.right.SealResponse(ctx, view.ID(), nil, oversized, true, "", 300, 200); err == nil {
					t.Fatal("4034 response AAD")
				}
				accepted, err = f.right.SealResponse(ctx, view.ID(), nil, metadata, true, "", 300, 200)
				if err != nil {
					t.Fatal("4033 response AAD", err)
				}
				if applicationSequence010(t, accepted, "data") != 1 {
					t.Fatal("refused response consumed sequence")
				}
			} else {
				if _, err = f.left.SealRequest(ctx, nil, oversized, 300); err == nil {
					t.Fatal("4034 request AAD")
				}
				accepted, err = f.left.SealRequest(ctx, nil, metadata, 300)
				if err != nil {
					t.Fatal("4033 request AAD", err)
				}
				if applicationSequence010(t, accepted, "payload") != 1 {
					t.Fatal("refused request consumed sequence")
				}
			}
			var acceptedFields map[string]json.RawMessage
			_ = json.Unmarshal(accepted.Body, &acceptedFields)
			if n := len(requestAAD010(acceptedFields)); n != 4033 {
				t.Fatalf("actual AAD %d", n)
			}
			if response {
				if _, err = f.left.OpenResponse(ctx, accepted); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err = f.right.OpenRequest(ctx, accepted); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	for _, value := range []string{strings.Repeat("é", 512), strings.Repeat("😀", 256)} {
		if _, err := copyApplicationMetadata010(map[string]string{"note": value}); err != nil {
			t.Fatal("1024 UTF8 byte boundary", err)
		}
		if _, err := copyApplicationMetadata010(map[string]string{"note": value + "x"}); err == nil {
			t.Fatal("1025 UTF8 byte boundary")
		}
	}
}

// This local TLS fixture exercises only successful authenticated exchanges and
// unavailable configured authority. No malformed-message runtime is provided.
func TestHTTPApplicationOwnedTLS010(t *testing.T) {
	for _, unavailable := range []bool{false, true} {
		t.Run(fmt.Sprintf("unavailable=%t", unavailable), func(t *testing.T) {
			var f applicationFixture010
			var target string
			handled := atomic.Int64{}
			problems := make(chan error, 4)
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.TLS == nil || r.Method != "POST" {
					problems <- fmt.Errorf("actual TLS or method")
					http.Error(w, "unavailable", http.StatusServiceUnavailable)
					return
				}
				body, err := io.ReadAll(io.LimitReader(r.Body, 32769))
				if err != nil {
					problems <- err
					http.Error(w, "unavailable", http.StatusServiceUnavailable)
					return
				}
				message := HTTPMessage010{Method: r.Method, Target: serverOrigin010(target) + r.URL.RequestURI(), Authority: r.Host, Body: body}
				for k, values := range r.Header {
					for _, v := range values {
						message.Headers = append(message.Headers, [2]string{strings.ToLower(k), v})
					}
				}
				view, err := f.right.OpenRequest(r.Context(), message)
				if err != nil {
					if !unavailable {
						problems <- err
					}
					http.Error(w, "unavailable", http.StatusServiceUnavailable)
					return
				}
				if unavailable {
					problems <- fmt.Errorf("unavailable authority released request")
					http.Error(w, "unavailable", http.StatusServiceUnavailable)
					return
				}
				if string(view.Payload()) != "inert TLS request" || view.Metadata()["a2a_version"] != "1.0" {
					problems <- fmt.Errorf("accepted request changed")
					http.Error(w, "unavailable", http.StatusServiceUnavailable)
					return
				}
				// Inert fixture handler, not the future ADK production dispatch barrier.
				handled.Add(1)
				reply, err := f.right.SealResponse(r.Context(), view.ID(), []byte("inert TLS response"), map[string]string{"note": "response"}, true, "", 300, 200)
				if err != nil {
					problems <- err
					http.Error(w, "unavailable", http.StatusServiceUnavailable)
					return
				}
				for _, h := range reply.Headers {
					w.Header().Add(h[0], h[1])
				}
				w.WriteHeader(reply.Status)
				if _, err = w.Write(reply.Body); err != nil {
					problems <- err
				}
			}))
			t.Cleanup(server.Close)
			target = server.URL + "/messages"
			f = applicationPairTarget010(t, true, target)
			ctx := context.Background()
			wire, err := f.left.SealRequest(ctx, []byte("inert TLS request"), map[string]string{"a2a_version": "1.0"}, 300)
			if err != nil {
				t.Fatal(err)
			}
			if unavailable {
				f.control.mode = "source-error"
			}
			request, err := http.NewRequestWithContext(ctx, "POST", target, bytes.NewReader(wire.Body))
			if err != nil {
				t.Fatal(err)
			}
			for _, h := range wire.Headers {
				request.Header.Add(h[0], h[1])
			}
			client := server.Client()
			client.Timeout = 5 * time.Second
			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := response.Body.Close(); err != nil {
					t.Error(err)
				}
			}()
			body, err := io.ReadAll(io.LimitReader(response.Body, 32769))
			if err != nil {
				t.Fatal(err)
			}
			message := HTTPMessage010{Status: response.StatusCode, Body: body}
			for k, values := range response.Header {
				for _, v := range values {
					message.Headers = append(message.Headers, [2]string{strings.ToLower(k), v})
				}
			}
			if unavailable {
				if response.StatusCode != http.StatusServiceUnavailable || handled.Load() != 0 {
					t.Fatal("authority refusal dispatched")
				}
				if v, err := f.left.OpenResponse(ctx, message); err == nil || v != nil {
					t.Fatal("unsigned refusal released response")
				}
			} else {
				view, err := f.left.OpenResponse(ctx, message)
				if err != nil || view == nil || string(view.Data()) != "inert TLS response" || view.Metadata()["note"] != "response" || handled.Load() != 1 {
					t.Fatal("actual TLS round trip", err)
				}
			}
			select {
			case err := <-problems:
				t.Fatal(err)
			default:
			}
		})
	}
}
func serverOrigin010(target string) string { return strings.TrimSuffix(target, "/messages") }

func TestHTTPApplicationConcurrentOwnerCopies010(t *testing.T) {
	f := applicationPair010(t, true)
	ctx := context.Background()
	copyOwner := *f.left
	wires := make(chan HTTPMessage010, 8)
	errors := make(chan error, 8)
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			owner := f.left
			if i%2 == 1 {
				owner = &copyOwner
			}
			wire, err := owner.SealRequest(ctx, []byte("inert"), map[string]string{"note": "concurrent"}, 300)
			if err != nil {
				errors <- err
				return
			}
			wires <- wire
		}(i)
	}
	workers.Wait()
	close(wires)
	close(errors)
	for err := range errors {
		t.Fatal(err)
	}
	sequences := map[uint64]bool{}
	var last HTTPMessage010
	for wire := range wires {
		n := applicationSequence010(t, wire, "payload")
		if sequences[n] {
			t.Fatal("owner copy reset sequence")
		}
		sequences[n] = true
		last = wire
	}
	if len(sequences) != 8 {
		t.Fatal("missing records")
	}
	// Race identical incoming records: acceptance and dispatch data occur once.
	copies := []HTTPApplicationOwner010{*f.right, *f.right}
	var accepted atomic.Int64
	for i := range copies {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			view, err := copies[i].OpenRequest(ctx, last)
			if err == nil && view != nil {
				accepted.Add(1)
			}
		}(i)
	}
	workers.Wait()
	if accepted.Load() != 1 {
		t.Fatal("concurrent duplicate acceptance", accepted.Load())
	}
	f.left.session.endpoint.Close()
	if _, err := copyOwner.SealRequest(ctx, nil, nil, 300); err == nil {
		t.Fatal("retired endpoint still usable")
	}
}

func TestHTTPApplicationProviderAndFinalGateRefusals010(t *testing.T) {
	for _, mode := range []string{"store-error", "store-delay", "clock-error", "rollback", "cancel-during-source", "expired-during-source"} {
		for _, response := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/response=%t", mode, response), func(t *testing.T) {
				f := applicationPair010(t, true)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				wire, err := f.left.SealRequest(ctx, []byte("inert"), nil, 300)
				if err != nil {
					t.Fatal(err)
				}
				receiver := f.right
				if response {
					view, err := f.right.OpenRequest(ctx, wire)
					if err != nil {
						t.Fatal(err)
					}
					wire, err = f.right.SealResponse(ctx, view.ID(), []byte("inert response"), nil, true, "", 300, 200)
					if err != nil {
						t.Fatal(err)
					}
					receiver = f.left
				}
				replay := receiver.session.endpoint.replay.(*completionReplay)
				before := len(replay.seen)
				switch mode {
				case "rollback":
					f.control.mono = -1
				case "cancel-during-source":
					f.reads.setEdit(cancel)
				case "expired-during-source":
					f.reads.setEdit(func() { f.control.utc = 400; f.control.mono = 300000 })
				default:
					f.control.mode = mode
				}
				if response {
					if view, err := receiver.OpenResponse(ctx, wire); err == nil || view != nil {
						t.Fatal("provider failure response")
					}
				} else {
					if view, err := receiver.OpenRequest(ctx, wire); err == nil || view != nil {
						t.Fatal("provider failure request")
					}
				}
				// Durable denial staging on final gate failure is not undone or asserted
				// absent; early failures have no published replay acceptance.
				if mode != "store-delay" && mode != "expired-during-source" && len(replay.seen) != before {
					t.Fatal("early failure accepted replay")
				}
			})
		}
	}
}

func TestHTTPApplicationLegacyResponseMetadataRefusal010(t *testing.T) {
	for _, metadata := range []map[string]string{{}, {"a2a_version": "1.0"}} {
		t.Run(fmt.Sprintf("entries=%d", len(metadata)), func(t *testing.T) {
			client, server, _ := recordPair010(t, 0)
			ctx := context.Background()
			if err := client.BindHTTP("https://agent.example/messages"); err != nil {
				t.Fatal(err)
			}
			owner, err := server.TakeHTTPApplication("https://agent.example/messages")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(owner.Close)
			wire, err := client.SealHTTPRequest(ctx, []byte("inert legacy request"), 300)
			if err != nil {
				t.Fatal(err)
			}
			request, err := owner.OpenRequest(ctx, wire)
			if err != nil {
				t.Fatal(err)
			}
			response, err := owner.SealResponse(ctx, request.ID(), []byte("inert metadata response"), metadata, true, "", 300, 200)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			_ = json.Unmarshal(response.Body, &fields)
			if err = verifyWire010(fields, true, owner.session.b.Signing()); err != nil {
				t.Fatal("valid signed response control", err)
			}
			if _, _, err = sessionRecord010(response.Body, 100, true); err == nil {
				t.Fatal("default response codec accepted metadata")
			}
			before := len(client.endpoint.replay.(*completionReplay).seen)
			if view, err := client.OpenHTTPResponse(ctx, response); err == nil || view != nil {
				t.Fatal("legacy response discarded metadata")
			}
			if client.sent[request.ID()].terminal || len(client.endpoint.replay.(*completionReplay).seen) != before || client.State() != "ESTABLISHED" {
				t.Fatal("legacy response refusal changed acceptance")
			}
			// A later metadata-free response remains usable by the ordinary API.
			next, err := client.SealHTTPRequest(ctx, []byte("inert positive control"), 300)
			if err != nil {
				t.Fatal(err)
			}
			nextView, err := owner.OpenRequest(ctx, next)
			if err != nil {
				t.Fatal(err)
			}
			reply, err := owner.SealResponse(ctx, nextView.ID(), []byte("inert legacy response"), nil, true, "", 300, 200)
			if err != nil {
				t.Fatal(err)
			}
			view, err := client.OpenHTTPResponse(ctx, reply)
			if err != nil || view == nil || string(view.Data) != "inert legacy response" {
				t.Fatal("metadata-free positive control", err)
			}
		})
	}
}

// The transaction store is trusted fault injection, never an external service.
// Its final callback is reached after current-authority validation.
type applicationCancelStore010 struct {
	*completionReplay
	cancel             context.CancelFunc
	before             func()
	calls, validations int
}

func (s *applicationCancelStore010) ReserveRecord(entry Replay010, validate func() error) error {
	s.calls++
	if s.before != nil {
		s.before()
	}
	if s.cancel != nil {
		s.cancel()
	}
	return s.completionReplay.ReserveRecord(entry, func() error { s.validations++; return validate() })
}
func TestHTTPApplicationCancellationAtPublication010(t *testing.T) {
	for _, response := range []bool{false, true} {
		for _, cancelAtStore := range []bool{false, true} {
			t.Run(fmt.Sprintf("response=%t/cancel=%t", response, cancelAtStore), func(t *testing.T) {
				f := applicationPair010(t, true)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				wire, err := f.left.SealRequest(ctx, []byte("inert request"), map[string]string{"note": "request"}, 300)
				if err != nil {
					t.Fatal(err)
				}
				receiver := f.right
				if response {
					view, err := f.right.OpenRequest(ctx, wire)
					if err != nil {
						t.Fatal(err)
					}
					wire, err = f.right.SealResponse(ctx, view.ID(), []byte("inert response"), nil, true, "", 300, 200)
					if err != nil {
						t.Fatal(err)
					}
					receiver = f.left
				}
				replay := receiver.session.endpoint.replay.(*completionReplay)
				beforeReplay := len(replay.seen)
				beforeReads := f.reads.calls.Load()
				store := &applicationCancelStore010{completionReplay: replay, before: func() {
					if f.reads.calls.Load() <= beforeReads {
						t.Error("store reached without current Source")
					}
				}}
				if cancelAtStore {
					store.cancel = cancel
				}
				receiver.session.endpoint.replay = store
				var accepted bool
				if response {
					view, err := receiver.OpenResponse(ctx, wire)
					accepted = err == nil && view != nil
					if (cancelAtStore && (err == nil || view != nil)) || (!cancelAtStore && (err != nil || view == nil)) {
						t.Fatal("partial response")
					}
				} else {
					view, err := receiver.OpenRequest(ctx, wire)
					accepted = err == nil && view != nil
					if (cancelAtStore && (err == nil || view != nil)) || (!cancelAtStore && (err != nil || view == nil)) {
						t.Fatal("partial request")
					}
				}
				if store.calls != 1 || store.validations != 1 {
					t.Fatal("final publication callback not reached once")
				}
				if cancelAtStore {
					if accepted || receiver.session.State() != "CLOSED" || len(replay.seen) != beforeReplay {
						t.Fatal("cancellation at final gate published acceptance")
					}
				} else {
					if !accepted || len(replay.seen) != beforeReplay+2 {
						t.Fatal("uncanceled publication control")
					}
				}
			})
		}
	}
}
