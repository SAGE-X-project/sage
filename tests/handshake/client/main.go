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
	"net/http/httputil"
	"time"

	a2a "github.com/a2aproject/a2a/grpc"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/sage-x-project/sage/core/rfc9421"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/sage-x-project/sage/crypto/keys"
	sagedid "github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/handshake"
	"github.com/sage-x-project/sage/session"
)

const base = "http://127.0.0.1:8080"
const verbose = false

func vprintf(format string, args ...any) {
	if verbose {
		log.Printf(format, args...)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	ctx := context.Background()

	clientKeyPair, err := keys.GenerateEd25519KeyPair()
	if err != nil {
		log.Fatalf("keygen: %v", err)
	}
	clientPriv := clientKeyPair.PrivateKey().(ed25519.PrivateKey)
	clientDID := sagedid.GenerateDID(sagedid.ChainEthereum, uuid.NewString())

	// ✅ 서버 공개키 얻기 (방어적 파서)
	serverPub := fetchServerPub()

	// 에이전트 등록
	registerPayload := map[string]any{
		"DID":    string(clientDID),
		"Name":   "Handshake Test Agent",
		"Active": true,
		"PubB64": base64.RawURLEncoding.EncodeToString(clientKeyPair.PublicKey().(ed25519.PublicKey)),
	}
	postJSON(base+"/debug/register-agent", registerPayload)

	ctxID := "ctx-" + uuid.NewString()

	sendInvitationHTTP(ctx, clientKeyPair, clientPriv, string(clientDID), ctxID, map[string]any{"note": "hi"})

	eph := mustX25519()
	jwk := must(formats.NewJWKExporter().ExportPublic(eph, sagecrypto.KeyFormatJWK))
	reqMsg := handshake.RequestMessage{EphemeralPubKey: json.RawMessage(jwk)}
	reqPlain := must(json.Marshal(reqMsg))
	packet := must(keys.EncryptWithEd25519Peer(serverPub, reqPlain))
	b64Packet := base64.RawURLEncoding.EncodeToString(packet)

	serverEph := sendRequestHTTP(ctx, clientKeyPair, clientPriv, string(clientDID), ctxID, b64Packet)

	sessionKey := sendCompleteHTTP(ctx, clientKeyPair, clientPriv, string(clientDID), ctxID, map[string]any{})

	shared := must(eph.DeriveSharedSecret(serverEph))
	params := session.Params{
		ContextID:    ctxID,
		SelfEph:      eph.PublicBytesKey(),
		PeerEph:      serverEph,
		Label:        "a2a/handshake v1",
		SharedSecret: shared,
	}
	clientSess, _, _, err := session.NewManager().EnsureSessionWithParams(params, nil)
	if err != nil {
		log.Fatalf("EnsureSession: %v", err)
	}

	// 정상 패킷
	body := []byte(`{"op":"ping","ts":1}`)
	cipher := mustBytes(clientSess.Encrypt(body))
	req := newProtectedRequest(cipher, string(clientDID), sessionKey, clientPriv)

	dump, _ := httputil.DumpRequestOut(req, true)
	log.Println("1) signed packet")
	vprintf("HTTP OUT:\n%s\n", dump)

	resp := must(http.DefaultClient.Do(req))
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("expected 200 OK, got %d body=%s", resp.StatusCode, string(respBody))
	}

	serverPlain := decryptServerResponse(clientSess, respBody)
	vprintf("protected#1 status=%d decrypted=%s\n", resp.StatusCode, string(serverPlain))

	// Invalid: empty body
	reqEmpty, _ := http.NewRequest("POST", base+"/protected", bytes.NewReader([]byte("again")))
	reqEmpty.Header = req.Header.Clone()
	digestEmpty := sha256.Sum256([]byte("again"))
	reqEmpty.Header.Set("Content-Digest", fmt.Sprintf("sha-256=:%s:", base64.StdEncoding.EncodeToString(digestEmpty[:])))
	dump, _ = httputil.DumpRequestOut(reqEmpty, true)
	log.Println("2) invalid packet(empty body)")
	vprintf("HTTP OUT:\n%s\n", dump)
	resEmpty := must(http.DefaultClient.Do(reqEmpty))
	defer resEmpty.Body.Close()
	vprintf("protected#empty body status=%d (expect 401)\n", resEmpty.StatusCode)

	// Invalid: bad Signature-Input
	reqBad, _ := http.NewRequest("POST", base+"/protected", bytes.NewReader([]byte(`{"op":"bad"}`)))
	reqBad.Header = req.Header.Clone()
	reqBad.Header.Set("Signature-Input", "sig1=invalid")
	dump, _ = httputil.DumpRequestOut(reqBad, true)
	log.Println("3) invalid packet(bad Signature-Input)")
	vprintf("HTTP OUT:\n%s\n", dump)
	resBad := must(http.DefaultClient.Do(reqBad))
	defer resBad.Body.Close()
	bBad, _ := io.ReadAll(resBad.Body)
	vprintf("protected#bad-signature status=%d (expect 400), body=%s\n", resBad.StatusCode, string(bBad))

	// Invalid: replay same nonce
	reqReplay, _ := http.NewRequest("POST", base+"/protected", bytes.NewReader(cipher))
	reqReplay.Header = req.Header.Clone()
	dump, _ = httputil.DumpRequestOut(reqReplay, true)
	log.Println("4) invalid packet(reuse nonce)")
	vprintf("HTTP OUT:\n%s\n", dump)
	resReplay := must(http.DefaultClient.Do(reqReplay))
	defer resReplay.Body.Close()
	bReplay, _ := io.ReadAll(resReplay.Body)
	vprintf("protected#replay status=%d (expect 401), body=%s\n", resReplay.StatusCode, string(bReplay))

	// Invalid: session expired
	time.Sleep(2500 * time.Millisecond)
	body2 := []byte(`{"op":"after-idle","ts":2}`)
	cipher2 := mustBytes(clientSess.Encrypt(body2))
	reqIdle := newProtectedRequest(cipher2, string(clientDID), sessionKey, clientPriv)
	dump, _ = httputil.DumpRequestOut(reqIdle, true)
	log.Println("5) invalid packet(expired session)")
	vprintf("HTTP OUT:\n%s\n", dump)
	resIdle := must(http.DefaultClient.Do(reqIdle))
	defer resIdle.Body.Close()
	bIdle, _ := io.ReadAll(resIdle.Body)
	vprintf("protected#idle status=%d (expect 401) body=%s\n", resIdle.StatusCode, string(bIdle))
}

// ========== 서버 공개키 파서(방어적) ==========
func fetchServerPub() ed25519.PublicKey {
	resp := must(http.Get(base + "/debug/server-pub"))
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// 1) {"pub":"..."} 형태
	var obj map[string]any
	if json.Unmarshal(body, &obj) == nil {
		if v, ok := obj["pub"]; ok {
			switch t := v.(type) {
			case string:
				return mustDecodeKey(t) // base64url 문자열
			case float64:
				log.Fatalf("server-pub 'pub' is number (%v), want base64 string", t)
			case []any:
				buf := make([]byte, 0, len(t))
				for _, e := range t {
					f, ok := e.(float64)
					if !ok || f < 0 || f > 255 {
						log.Fatalf("invalid byte in pub array: %v", e)
					}
					buf = append(buf, byte(f))
				}
				if len(buf) != ed25519.PublicKeySize {
					log.Fatalf("unexpected pubkey length: got %d", len(buf))
				}
				return ed25519.PublicKey(buf)
			default:
				log.Fatalf("unsupported 'pub' type: %T", t)
			}
		}
	}

	// 2) "...." 단일 JSON 문자열
	var s string
	if json.Unmarshal(body, &s) == nil {
		return mustDecodeKey(s)
	}

	// 3) 생바이너리(드묾)
	if len(body) == ed25519.PublicKeySize {
		return ed25519.PublicKey(append([]byte(nil), body...))
	}

	log.Fatalf("cannot parse /debug/server-pub: %q", string(body))
	return nil
}

// ===== handshake helpers =====

func sendInvitationHTTP(ctx context.Context, kp sagecrypto.KeyPair, priv ed25519.PrivateKey, did, ctxID string, payload map[string]any) {
	msg := buildMessage(ctxID, handshake.Invitation, payload)
	meta := mustMeta(kp, did, msg)
	_ = sendToHTTP(ctx, msg, meta, priv)
}

func sendRequestHTTP(ctx context.Context, kp sagecrypto.KeyPair, priv ed25519.PrivateKey, did, ctxID, b64Packet string) []byte {
	msg := buildMessage(ctxID, handshake.Request, map[string]any{"b64": b64Packet})
	meta := mustMeta(kp, did, msg)
	resp := sendToHTTP(ctx, msg, meta, priv)
	if resp == nil || len(resp.EphemeralPubKey) == 0 {
		log.Fatal("server response missing ephemeral key")
	}
	importer := formats.NewJWKImporter()
	pubAny, err := importer.ImportPublic(resp.EphemeralPubKey, sagecrypto.KeyFormatJWK)
	if err != nil {
		log.Fatalf("import server eph: %v", err)
	}
	pub, ok := pubAny.(*ecdh.PublicKey)
	if !ok {
		log.Fatalf("unexpected eph key type: %T", pubAny)
	}
	return pub.Bytes()
}

func sendCompleteHTTP(ctx context.Context, kp sagecrypto.KeyPair, priv ed25519.PrivateKey, did, ctxID string, payload map[string]any) string {
	msg := buildMessage(ctxID, handshake.Complete, payload)
	meta := mustMeta(kp, did, msg)
	resp := sendToHTTP(ctx, msg, meta, priv)
	if resp == nil || resp.KeyID == "" {
		log.Fatal("server response missing key id")
	}
	return resp.KeyID
}

func sendToHTTP(ctx context.Context, msg *a2a.Message, meta *structpb.Struct, priv ed25519.PrivateKey) *handshake.ResponseMessage {
	payload := &a2a.SendMessageRequest{Request: msg, Metadata: meta}
	b, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", base+"/v1/a2a:sendMessage", bytes.NewReader(b))
	if err != nil {
		log.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res := must(http.DefaultClient.Do(req))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		log.Fatalf("SendMessage failed: %s", string(body))
	}
	body, _ := io.ReadAll(res.Body)
	if len(body) == 0 {
		return nil
	}
	var resp a2a.SendMessageResponse
	if err := protojson.Unmarshal(body, &resp); err != nil {
		log.Fatalf("decode response: %v", err)
	}
	msgResp := resp.GetMsg()
	if msgResp == nil || len(msgResp.GetContent()) == 0 {
		return nil
	}
	data := msgResp.GetContent()[0].GetData()
	if data == nil || data.Data == nil {
		return nil
	}
	b64Field := data.Data.Fields["b64"].GetStringValue()
	packet, err := base64.RawURLEncoding.DecodeString(b64Field)
	if err != nil {
		log.Fatalf("decode packet: %v", err)
	}
	plain, err := keys.DecryptWithEd25519Peer(priv, packet)
	if err != nil {
		log.Fatalf("decrypt packet: %v", err)
	}
	var response handshake.ResponseMessage
	if err := json.Unmarshal(plain, &response); err != nil {
		log.Fatalf("unmarshal response: %v", err)
	}
	return &response
}

func buildMessage(ctxID string, phase handshake.Phase, payload map[string]any) *a2a.Message {
	data, _ := structpb.NewStruct(payload)
	return &a2a.Message{
		MessageId: uuid.NewString(),
		ContextId: ctxID,
		TaskId:    handshake.GenerateTaskID(phase),
		Role:      a2a.Role_ROLE_USER,
		Content:   []*a2a.Part{{Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: data}}}},
	}
}

func mustMeta(kp sagecrypto.KeyPair, did string, m *a2a.Message) *structpb.Struct {
	bin, _ := proto.MarshalOptions{Deterministic: true}.Marshal(m)
	sig, _ := kp.Sign(bin)
	st, _ := structpb.NewStruct(map[string]any{
		"signature":      base64.RawURLEncoding.EncodeToString(sig),
		"did":            did,
		"client_pub_b64": base64.RawURLEncoding.EncodeToString(kp.PublicKey().(ed25519.PublicKey)),
	})
	return st
}

func must[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func mustBytes(b []byte, err error) []byte {
	if err != nil {
		log.Fatal(err)
	}
	return b
}

func mustDecodeKey(b64 string) ed25519.PublicKey {
	b, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		log.Fatalf("decode pub: %v", err)
	}
	return ed25519.PublicKey(b)
}

func mustX25519() *keys.X25519KeyPair {
	kp, err := keys.GenerateX25519KeyPair()
	if err != nil {
		log.Fatalf("x25519: %v", err)
	}
	return kp.(*keys.X25519KeyPair)
}

func newProtectedRequest(cipher []byte, did string, keyID string, priv ed25519.PrivateKey) *http.Request {
	req, err := http.NewRequest("POST", base+"/protected", bytes.NewReader(cipher))
	if err != nil {
		log.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-DID", did)
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	digest := sha256.Sum256(cipher)
	req.Header.Set("Content-Digest", fmt.Sprintf("sha-256=:%s:", base64.StdEncoding.EncodeToString(digest[:])))

	params := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{`"@method"`, `"@path"`, `"content-digest"`, `"date"`, `"@authority"`},
		KeyID:             keyID,
		Nonce:             "n-" + uuid.NewString(),
		Created:           time.Now().Unix(),
	}
	verifier := rfc9421.NewHTTPVerifier()
	if err := verifier.SignRequest(req, "sig1", params, priv); err != nil {
		log.Fatalf("sign request: %v", err)
	}
	return req
}

func decryptServerResponse(sess session.Session, body []byte) []byte {
	var resp struct {
		CipherB64 string `json:"cipher_b64"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		log.Fatalf("decode server response json: %v", err)
	}
	cipher, err := base64.StdEncoding.DecodeString(resp.CipherB64)
	if err != nil {
		log.Fatalf("decode server response cipher: %v", err)
	}
	plain, err := sess.Decrypt(cipher)
	if err != nil {
		log.Fatalf("decrypt server response: %v", err)
	}
	return plain
}

func getJSON(url string, out any) {
	res := must(http.Get(url))
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		log.Fatalf("decode json: %v", err)
	}
}

func postJSON(url string, payload any) {
	b, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		log.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res := must(http.DefaultClient.Do(req))
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		log.Fatalf("post failed: %s", string(body))
	}
}
