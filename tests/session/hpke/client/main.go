// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.


package main

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sage-x-project/sage/core/rfc9421"
	"github.com/sage-x-project/sage/crypto/keys"
	sagedid "github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/hpke"
	"github.com/sage-x-project/sage/session"
)

const (
	baseURL  = "http://127.0.0.1:8080" // Hosts both the A2A proxy and the protected endpoint
	grpcAddr = "127.0.0.1:18080"       // HPKE gRPC service
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	ctx := context.Background()

	// 1) Generate the client signing key and DID
	clientSignKP, _ := keys.GenerateEd25519KeyPair()
	clientPriv := clientSignKP.PrivateKey().(ed25519.PrivateKey)
	clientDID := string(sagedid.GenerateDID(sagedid.ChainEthereum, "cli-"+uuid.NewString()))

	// 2) Fetch server debug information
	serverDID := fetchServerDID()
	_ = mustFetchServerKEMPub() // quick health check

	// 3) Register the client's DID/public key so the server can verify HTTP signatures
	registerClientOnServer(clientDID, clientSignKP.PublicKey().(ed25519.PublicKey))

	// 4) Instantiate hpke.Client over gRPC; resolver fetches the KEM public key over HTTP
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	mustNoErr(err, "grpc dial")
	defer conn.Close()

	resolver := &httpResolver{serverDID: serverDID}
	cliMgr := session.NewManager()
	cli := hpke.NewClient(conn, resolver, clientSignKP, clientDID, hpke.DefaultInfoBuilder{}, cliMgr)

	// 5) HPKE init
	ctxID := "ctx-" + uuid.NewString()
	kid, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	mustNoErr(err, "hpke Initialize")
	log.Printf("[client] hpke initialized, kid=%s", kid)

	sess, ok := cliMgr.GetByKeyID(kid)
	if !ok {
		log.Fatal("session not found after hpke init")
	}

	// ===== Test #1: Server → client round-trip (decrypt server response) =====
	callProtectedExpectOK(sess, clientPriv, clientDID, kid, `{"op":"server-echo","n":1}`)

	// ===== Test #2: Nonce replay — expect the second request to return 401 =====
	{
		nonce := "n-replay-" + uuid.NewString()

		// First request with this nonce should succeed (200 OK)
		body1 := []byte(`{"op":"replay-1","n":2}`)
		cipher1, _ := sess.Encrypt(body1)
		req1 := newSignedProtectedRequestWithNonce(cipher1, clientDID, kid, clientPriv, nonce)
		res1 := must(http.DefaultClient.Do(req1))
		func() {
			defer res1.Body.Close()
			if res1.StatusCode != http.StatusOK {
				b, _ := io.ReadAll(res1.Body)
				log.Fatalf("[client] replay#1 unexpected status=%d body=%s", res1.StatusCode, string(b))
			}
			var w struct {
				CipherB64 string `json:"cipher_b64"`
			}
			_ = json.NewDecoder(res1.Body).Decode(&w)
			ct, _ := base64.StdEncoding.DecodeString(w.CipherB64)
			plain, _ := sess.Decrypt(ct)
			log.Printf("[client] replay#1 OK -> %s", string(plain))
		}()

		// Second request reuses the same nonce; expect 401
		body2 := []byte(`{"op":"replay-2","n":3}`)
		cipher2, _ := sess.Encrypt(body2)
		req2 := newSignedProtectedRequestWithNonce(cipher2, clientDID, kid, clientPriv, nonce) // reuse the same nonce
		res2 := must(http.DefaultClient.Do(req2))
		defer res2.Body.Close()
		log.Printf("[client] replay#2 status=%d (expect 401)", res2.StatusCode)
		if res2.StatusCode == http.StatusOK {
			log.Fatal("expected server to reject replayed nonce")
		}
	}

	// ===== Test #3: Invalid signature should be rejected =====
	// Sign correctly, then corrupt only the Signature-Input header
	{
		body := []byte(`{"op":"bad-signature","n":4}`)
		cipher, _ := sess.Encrypt(body)
		req := mustNewSignedProtectedRequest(cipher, clientDID, kid, clientPriv)
		req.Header.Set("Signature-Input", "sig1=invalid") // deliberate tampering
		resp := must(http.DefaultClient.Do(req))
		defer resp.Body.Close()
		log.Printf("[client] bad-signature status=%d (expect 400/401)", resp.StatusCode)
		if resp.StatusCode == http.StatusOK {
			log.Fatal("expected server to reject invalid signature")
		}
	}

	// ===== Test #4: Idle-expired session packet should be rejected =====
	// Server idle timeout is 2s; wait 3s before retrying.
	time.Sleep(3 * time.Second)
	{
		body := []byte(`{"op":"after-idle","n":5}`)
		cipher, _ := sess.Encrypt(body)
		req := mustNewSignedProtectedRequest(cipher, clientDID, kid, clientPriv)
		resp := must(http.DefaultClient.Do(req))
		defer resp.Body.Close()
		log.Printf("[client] after-idle status=%d (expect 401)", resp.StatusCode)
		if resp.StatusCode == http.StatusOK {
			log.Fatal("expected server to reject expired/idle session packet")
		}
	}

	log.Println("[client] DONE")
}

/* --------------------- HTTP resolver --------------------- */

type httpResolver struct {
	serverDID string
}

func (r *httpResolver) Resolve(ctx context.Context, did sagedid.AgentDID) (*sagedid.AgentMetadata, error) {
	if string(did) != r.serverDID {
		return nil, fmt.Errorf("unknown DID: %s", did)
	}
	return &sagedid.AgentMetadata{
		DID:      did,
		IsActive: true,
		// hpke.Client may call both ResolvePublicKey and ResolveKEMKey, so return the same KEM key via each path.
		PublicKey:    mustFetchServerSignPub(),
		PublicKEMKey: mustFetchServerKEMPub(),
	}, nil
}
func (r *httpResolver) ResolvePublicKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	return mustFetchServerSignPub(), nil
}
func (r *httpResolver) ResolveKEMKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	return mustFetchServerKEMPub(), nil
}
func (r *httpResolver) VerifyMetadata(context.Context, sagedid.AgentDID, *sagedid.AgentMetadata) (*sagedid.VerificationResult, error) {
	return &sagedid.VerificationResult{Valid: true, VerifiedAt: time.Now()}, nil
}
func (r *httpResolver) ListAgentsByOwner(context.Context, string) ([]*sagedid.AgentMetadata, error) {
	return nil, nil
}
func (r *httpResolver) Search(context.Context, sagedid.SearchCriteria) ([]*sagedid.AgentMetadata, error) {
	return nil, nil
}

/* --------------------- protected calls --------------------- */

func callProtectedExpectOK(sess session.Session, clientPriv ed25519.PrivateKey, clientDID, kid string, jsonBody string) {
	body := []byte(jsonBody)
	cipher, _ := sess.Encrypt(body)

	req := mustNewSignedProtectedRequest(cipher, clientDID, kid, clientPriv)
	resp := must(http.DefaultClient.Do(req))
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("protected response %d: %s", resp.StatusCode, string(out))
	}
	var wrapper struct {
		CipherB64 string `json:"cipher_b64"`
	}
	_ = json.Unmarshal(out, &wrapper)
	ct, _ := base64.StdEncoding.DecodeString(wrapper.CipherB64)
	plain, _ := sess.Decrypt(ct)
	log.Printf("[client] protected OK -> %s", string(plain))
}

func mustNewSignedProtectedRequest(cipher []byte, did, kid string, priv ed25519.PrivateKey) *http.Request {
	nonce := "n-" + uuid.NewString()
	return newSignedProtectedRequestWithNonce(cipher, did, kid, priv, nonce)
}

// Helper for signing with a caller-specified nonce
func newSignedProtectedRequestWithNonce(cipher []byte, did, kid string, priv ed25519.PrivateKey, nonce string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/protected", bytes.NewReader(cipher))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-DID", did)
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	sum := sha256.Sum256(cipher)
	req.Header.Set("Content-Digest", fmt.Sprintf("sha-256=:%s:", base64.StdEncoding.EncodeToString(sum[:])))

	params := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{`"@method"`, `"@path"`, `"content-digest"`, `"date"`, `"@authority"`},
		KeyID:             kid,
		Nonce:             nonce, // inject fixed nonce
		Created:           time.Now().Unix(),
	}
	verifier := rfc9421.NewHTTPVerifier()
	if err := verifier.SignRequest(req, "sig1", params, priv); err != nil {
		log.Fatalf("SignRequest: %v", err)
	}
	return req
}

/* --------------------- debug helpers --------------------- */

func registerClientOnServer(did string, pub ed25519.PublicKey) {
	obj := map[string]any{
		"DID":    did,
		"Name":   "Client",
		"Active": true,
		"PubB64": base64.RawURLEncoding.EncodeToString(pub),
	}
	b, _ := json.Marshal(obj)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/debug/register-agent", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp := must(http.DefaultClient.Do(req))
	resp.Body.Close()
}

func mustFetchServerKEMPub() *ecdh.PublicKey {
	resp := must(http.Get(baseURL + "/debug/kem-pub"))
	defer resp.Body.Close()
	var m map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&m)
	b, _ := base64.RawURLEncoding.DecodeString(m["kem"])
	pub, _ := ecdh.X25519().NewPublicKey(b)
	return pub
}
func mustFetchServerSignPub() ed25519.PublicKey {
    resp := must(http.Get(baseURL + "/debug/server-pub"))
    defer resp.Body.Close()
    var m map[string]string
    if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
        log.Fatalf("decode server-pub: %v", err)
    }
    b, err := base64.RawURLEncoding.DecodeString(m["pub"])
    if err != nil || len(b) != ed25519.PublicKeySize {
        log.Fatalf("bad server ed25519 pub: %v len=%d", err, len(b))
    }
    return ed25519.PublicKey(b)
}
func fetchServerDID() string {
	resp := must(http.Get(baseURL + "/debug/server-did"))
	defer resp.Body.Close()
	var m map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&m)
	return m["did"]
}

func must[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}
func mustNoErr(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %v", msg, err)
	}
}
