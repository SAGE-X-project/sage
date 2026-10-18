package hpke

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

// These are inert application bytes, never an authority input or a DID parser test.
func TestHTTPInformationalIdentityPayload010(t *testing.T) {
	payloads := map[string]string{
		"document": `{"didDocument":{"id":"did:sage:web:agent.example:alice","verificationMethod":[],"service":[]}}`,
		"aliases":  `{"alsoKnownAs":["did:example:display"],"equivalentId":["did:sage:web:agent.example:bob"],"canonicalId":"did:example:display"}`,
		"combined": `{"didDocument":{"id":"did:sage:web:agent.example:bob","verificationMethod":[]},"canonicalId":"did:example:display","note":"display-only"}`,
	}
	for name, payload := range payloads {
		for _, mode := range []string{"revoke-init", "revoke-resp", "source-error"} {
			t.Run(name+"/"+mode, func(t *testing.T) {
				client, server, control := recordPair010(t, 0)
				ctx := context.Background()
				for _, session := range []*AuthenticatedCompletion010{client, server} {
					if err := session.BindHTTP("https://agent.example/messages"); err != nil {
						t.Fatal(err)
					}
				}
				first, err := client.SealHTTPRequest(ctx, []byte(payload), 300)
				if err != nil {
					t.Fatal(err)
				}
				var request map[string]json.RawMessage
				if err = json.Unmarshal(first.Body, &request); err != nil {
					t.Fatal(err)
				}
				if str010(request, "did") != completionAlice || str010(request, "recipient") != completionBob || str010(request, "kid") != completionAlice+"#signing-1" {
					t.Fatal("configured tuple changed")
				}
				got, err := server.OpenHTTPRequest(ctx, first)
				if err != nil || !bytes.Equal(got, []byte(payload)) {
					t.Fatalf("positive control: %q %v", got, err)
				}
				reply, err := server.SealHTTPResponse(ctx, str010(request, "id"), []byte(payload), true, "", 300, 200)
				if err != nil {
					t.Fatal(err)
				}
				result, err := client.OpenHTTPResponse(ctx, reply)
				if err != nil || result == nil || !result.Success || result.MessageID != str010(request, "id") || !bytes.Equal(result.Data, []byte(payload)) {
					t.Fatalf("bound response: %v %v", result, err)
				}
				if _, err = server.OpenHTTPRequest(ctx, first); err == nil {
					t.Fatal("request replay accepted")
				}
				if _, err = client.OpenHTTPResponse(ctx, reply); err == nil {
					t.Fatal("terminal response reused")
				}
				second, err := client.SealHTTPRequest(ctx, []byte(payload), 300)
				if err != nil {
					t.Fatal(err)
				}
				replay := server.endpoint.replay.(*completionReplay)
				retained := len(replay.seen)
				control.mode = mode
				got, err = server.OpenHTTPRequest(ctx, second)
				if err == nil || got != nil || server.State() != "CLOSED" {
					t.Fatalf("unavailable authority released bytes: %q %v %s", got, err, server.State())
				}
				if len(replay.seen) != retained {
					t.Fatal("early registry refusal published replay acceptance")
				}
			})
		}
	}
}
