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

	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	sagedid "github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/hpke"
	"github.com/sage-x-project/sage/pkg/agent/session"
)

const (
	baseURL        = "http://127.0.0.1:8080"
	grpcAddr       = "127.0.0.1:18080"
	scenarioHeader = "X-Test-Scenario"
)

type packetScenario struct {
	label string
	desc  string
}

func logPacket(direction string, sc packetScenario, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[packet %s] %s %s", sc.label, direction, msg)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	ctx := context.Background()

	// 1) Generate client Ed25519 keypair and DID
	clientKP, _ := keys.GenerateEd25519KeyPair()
	clientPriv := clientKP.PrivateKey().(ed25519.PrivateKey)
	clientDID := string(sagedid.GenerateDID(sagedid.ChainEthereum, "cli-"+uuid.NewString()))

	// 2) Fetch server DID and KEM public key
	serverDID := fetchServerDID()
	_ = mustFetchServerKEMPub() // quick health check

	// 3) Register the client public key so the server can verify HTTP signatures
	registerClientOnServer(clientDID, clientKP.PublicKey().(ed25519.PublicKey))

	// 4) Create hpke.Client over gRPC
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	mustNoErr(err, "grpc dial")
	defer conn.Close()

	resolver := &httpResolver{serverDID: serverDID}
	cliMgr := session.NewManager()
	hClient := hpke.NewClient(conn, resolver, clientKP, clientDID, hpke.DefaultInfoBuilder{}, cliMgr)

	// 5) Run HPKE init to obtain the key ID and session
	ctxID := "ctx-" + uuid.NewString()
	kid, err := hClient.Initialize(ctx, ctxID, clientDID, serverDID)
	mustNoErr(err, "hpke Initialize")
	log.Printf("[client] hpke initialized, kid=%s", kid)
	sess, ok := cliMgr.GetByKeyID(kid)
	if !ok {
		log.Fatal("session not found after hpke init")
	}

	// ===== Scenario definitions =====
	scValid := packetScenario{label: "01-signed", desc: "signed packet"}
	scEmpty := packetScenario{label: "02-empty-body", desc: "empty body"}
	scBad := packetScenario{label: "03-bad-signature", desc: "bad Signature-Input"}
	scReplay := packetScenario{label: "04-replay", desc: "reused nonce"}
	scExpired := packetScenario{label: "05-expired", desc: "after idle timeout"}

	// ===== 1) Valid packet =====
	body := []byte(`{"op":"ping","ts":1}`)
	cipher := must(sess.Encrypt(body))
	req := newProtectedRequest(cipher, clientDID, kid, clientPriv)
	tagScenario(req, scValid)
	logPacket("client ->", scValid, "sending %s", scValid.desc)
	resp := must(http.DefaultClient.Do(req))
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("expected 200 OK, got %d body=%s", resp.StatusCode, string(b))
	}
	logPacket("client <-", scValid, "status %d", resp.StatusCode)
	plain := mustDecryptWrapper(sess, b)
	log.Printf("[client] decrypted server echo: %s", string(plain))

	// ===== 2) Empty body =====
	reqEmpty, _ := http.NewRequest("POST", baseURL+"/protected", bytes.NewReader([]byte("")))
	reqEmpty.Header = req.Header.Clone()
	tagScenario(reqEmpty, scEmpty)
	d := sha256.Sum256([]byte(""))
	reqEmpty.Header.Set("Content-Digest", fmt.Sprintf("sha-256=:%s:", base64.StdEncoding.EncodeToString(d[:])))
	logPacket("client ->", scEmpty, "sending %s", scEmpty.desc)
	resEmpty := must(http.DefaultClient.Do(reqEmpty))
	defer resEmpty.Body.Close()
	logPacket("client <-", scEmpty, "status %d (expected 401)", resEmpty.StatusCode)

	// ===== 3) Invalid Signature-Input =====
	reqBad, _ := http.NewRequest("POST", baseURL+"/protected", bytes.NewReader([]byte(`{"op":"bad"}`)))
	reqBad.Header = req.Header.Clone()
	tagScenario(reqBad, scBad)
	reqBad.Header.Set("Signature-Input", "sig1=invalid")
	logPacket("client ->", scBad, "sending %s", scBad.desc)
	resBad := must(http.DefaultClient.Do(reqBad))
	defer resBad.Body.Close()
	logPacket("client <-", scBad, "status %d (expected 400/401)", resBad.StatusCode)

	// ===== 4) Replay (nonce reuse) =====
	reqReplay, _ := http.NewRequest("POST", baseURL+"/protected", bytes.NewReader(cipher))
	reqReplay.Header = req.Header.Clone()
	tagScenario(reqReplay, scReplay)
	logPacket("client ->", scReplay, "sending %s", scReplay.desc)
	resReplay := must(http.DefaultClient.Do(reqReplay))
	defer resReplay.Body.Close()
	logPacket("client <-", scReplay, "status %d (expected 401)", resReplay.StatusCode)

	// ===== 5) Send after session idle expiry =====
	time.Sleep(3 * time.Second) // server IdleTimeout is 2s
	body2 := []byte(`{"op":"after-idle","ts":2}`)
	cipher2 := must(sess.Encrypt(body2))
	reqIdle := newProtectedRequest(cipher2, clientDID, kid, clientPriv)
	tagScenario(reqIdle, scExpired)
	logPacket("client ->", scExpired, "sending %s", scExpired.desc)
	resIdle := must(http.DefaultClient.Do(reqIdle))
	defer resIdle.Body.Close()
	logPacket("client <-", scExpired, "status %d (expected 401)", resIdle.StatusCode)

	log.Println("[client] DONE")
}

/* ---------------- HTTP resolver for client ---------------- */

type httpResolver struct {
	serverDID string
}

func (r *httpResolver) Resolve(ctx context.Context, did sagedid.AgentDID) (*sagedid.AgentMetadata, error) {
	if string(did) != r.serverDID {
		return nil, fmt.Errorf("unknown DID: %s", did)
	}
	kem := mustFetchServerKEMPub()
	return &sagedid.AgentMetadata{
		DID:          did,
		IsActive:     true,
		PublicKey:    kem, // return the same value for convenience
		PublicKEMKey: kem,
	}, nil
}
func (r *httpResolver) ResolvePublicKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	return mustFetchServerKEMPub(), nil
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

/* ---------------- protected helpers ---------------- */

func newProtectedRequest(cipher []byte, did, kid string, priv ed25519.PrivateKey) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/protected", bytes.NewReader(cipher))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-DID", did)
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	sum := sha256.Sum256(cipher)
	req.Header.Set("Content-Digest", fmt.Sprintf("sha-256=:%s:", base64.StdEncoding.EncodeToString(sum[:])))

	params := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{`"@method"`, `"@path"`, `"content-digest"`, `"date"`, `"@authority"`},
		KeyID:             kid,
		Nonce:             "n-" + uuid.NewString(),
		Created:           time.Now().Unix(),
	}
	ver := rfc9421.NewHTTPVerifier()
	if err := ver.SignRequest(req, "sig1", params, priv); err != nil {
		log.Fatalf("sign request: %v", err)
	}
	return req
}
func tagScenario(req *http.Request, sc packetScenario) { req.Header.Set(scenarioHeader, sc.label) }

func mustDecryptWrapper(sess session.Session, body []byte) []byte {
	var resp struct {
		CipherB64 string `json:"cipher_b64"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		log.Fatalf("decode server response json: %v", err)
	}
	ct, err := base64.StdEncoding.DecodeString(resp.CipherB64)
	if err != nil {
		log.Fatalf("decode server response cipher: %v", err)
	}
	plain, err := sess.Decrypt(ct)
	if err != nil {
		log.Fatalf("decrypt server response: %v", err)
	}
	return plain
}

/* ---------------- debug endpoints ---------------- */

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
func fetchServerDID() string {
	resp := must(http.Get(baseURL + "/debug/server-did"))
	defer resp.Body.Close()
	var m map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&m)
	return m["did"]
}

/* ---------------- tiny must helpers ---------------- */

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
