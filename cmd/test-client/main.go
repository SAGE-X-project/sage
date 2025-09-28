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
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	a2a "github.com/a2aproject/a2a/grpc"
	"github.com/google/uuid"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/sage-x-project/sage/crypto/keys"
	sagedid "github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/handshake"
	"github.com/sage-x-project/sage/session"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

const base = "http://127.0.0.1:8080"

var clientPriv ed25519.PrivateKey
var clientKeypair sagecrypto.KeyPair
var (
	gotEphCh = make(chan []byte, 1)
	gotKidCh = make(chan string, 1)
)

func main() {
	ctx := context.Background()

	// 0) My DID/signing key (ed25519, same as test)
	clientKeypair, _ = keys.GenerateEd25519KeyPair()
	myDID := sagedid.AgentDID("did:sage:ethereum:agent001")
	clientPriv = clientKeypair.PrivateKey().(ed25519.PrivateKey) 

	// 1) Server ed25519 public key (for bootstrap encryption)
	var sp struct{ Pub string }
	getJSON(base+"/debug/server-pub", &sp)
	serverPub := mustB64(sp.Pub)
 
	// 3) Start callback server & register server outbound target
	rpcURL := startClientRPC()
	postJSON(base+"/debug/set-outbound", map[string]string{"URL": rpcURL})

	meta := &sagedid.AgentMetadata{
		DID:       myDID,
		Name:      "Active Agent",
		IsActive:  true,
		PublicKey: clientKeypair.PublicKey(),
	}

	postJSON(base+"/debug/register-agent", map[string]any{
		"DID":    string(meta.DID),
		"Name":   meta.Name,
		"Active": meta.IsActive,
		"PubB64": base64.RawURLEncoding.EncodeToString(meta.PublicKey.(ed25519.PublicKey)),
	})


	// 4) 3-step handshake — implementing handshake.Client.* logic from tests adapted for HTTP
	ctxID := "ctx-" + uuid.NewString()
	log.Printf("----- handshake -----")
	// 4-1) Invitation (clear JSON) — ROLE_USER
	log.Printf("Invitation ...")
	sendInvitationHTTP(ctx, clientPriv, string(myDID), ctxID, map[string]any{"note": "hi"})

	// 4-2) Request (encrypted b64) — ROLE_USER
	// Create X25519 eph JWK → bootstrap encryption with server ed25519 → {"b64": ...}
	eph := mustX25519()
	jwk := must(formats.NewJWKExporter().ExportPublic(eph, sagecrypto.KeyFormatJWK))
	reqMsg := handshake.RequestMessage{
		EphemeralPubKey: json.RawMessage(jwk), 
	}
	reqPlain := must(json.Marshal(reqMsg))
	packet := must(keys.EncryptWithEd25519Peer(serverPub, reqPlain))
	b64Packet := base64.RawURLEncoding.EncodeToString(packet)
	log.Printf("Request ...")
	sendRequestHTTP(ctx, clientPriv, string(myDID), ctxID, b64Packet)

	// 4-3) Complete (clear JSON) — ROLE_USER
	log.Printf("Complete ... ✅")
	sendCompleteHTTP(ctx, clientPriv, string(myDID), ctxID, map[string]any{})
	log.Printf("----- handshake -----")
	// 5) Server outbound: receive server eph / kid → create session
	serverEphRaw := waitServerEph()

	kid := waitKID()
	log.Printf(`Get KID %s`, kid)
	shared := must(eph.DeriveSharedSecret(serverEphRaw))
	params := session.Params{
		ContextID:    ctxID,
		SelfEph:      eph.PublicBytesKey(),
		PeerEph:      serverEphRaw,
		Label:        "a2a/handshake v1",
		SharedSecret: shared,
	}
	
	sess, sid, _, err := session.NewManager().EnsureSessionWithParams(params, nil)
	if err != nil { log.Fatal(err) }
	log.Printf(`Get session ID %s`, sid)
	// 6) Call /protected: 1st success → 2nd (different nonce) 401 (assuming server MaxMessages=1)
	body := []byte(`{"op":"ping","ts":1} TEST MESSAGE`)
	cipher := mustBytes(sess.Encrypt(body))

	nonce := "n-" + uuid.NewString()
	date := time.Now().UTC().Format(time.RFC1123)
	digest := "sha-256=:" + b64(sha256Sum(cipher)) + ":"

	protected := base + "/protected"
	req, _ := http.NewRequest("POST", protected, bytes.NewReader(cipher))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Digest", digest)
	req.Header.Set("Date", date)
	req.Header.Set("Signature-Input",
		fmt.Sprintf(`sig1=("@method" "@path" "host" "date" "content-digest");alg="hmac-sha256";keyid="%s";nonce="%s"`, kid, nonce))
	covered := buildCovered(req, "sig1", kid, nonce)
	sig := sess.SignCovered(covered)
	
	req.Header.Set("Signature", "sig1=:" + base64.RawURLEncoding.EncodeToString(sig) + ":")
	dump, _ := httputil.DumpRequestOut(req, true) // 클라에서 보낼 때
	log.Println("---------- valid packet -----------")
	log.Printf("HTTP OUT:\n%s\n", dump)

	res := must(http.DefaultClient.Do(req))
	defer res.Body.Close()
	b1, _ := io.ReadAll(res.Body)
	log.Printf("protected#1 status=%d body=%s\n\n", res.StatusCode, string(b1))

	reqEmpty, _ := http.NewRequest("POST", protected, bytes.NewReader([]byte("again")))
	reqEmpty.Header = req.Header.Clone()
	reqEmpty.Header.Set("Signature-Input",
		fmt.Sprintf(`sig1=("@method" "@path" "host" "date" "content-digest");alg="hmac-sha256";keyid="%s";nonce="%s"`, kid, "n-"+uuid.NewString()))
	dump, _ = httputil.DumpRequestOut(reqEmpty, true) 
	log.Println("---------- invalid packet(empty body) -----------")
	log.Printf("HTTP OUT:\n%s\n", dump)
	res2 := must(http.DefaultClient.Do(reqEmpty))
	defer res2.Body.Close()
	log.Printf("protected#empty body status=%d (expect 401)\n\n", res2.StatusCode)


	reqBad, _ := http.NewRequest("POST", protected, bytes.NewReader([]byte(`{"op":"bad"}`)))
	dump, _ = httputil.DumpRequestOut(reqBad, true) 
	log.Println("---------- invalid packet(bad Signature-Input) -----------")
	log.Printf("HTTP OUT:\n%s\n", dump)
	resBad := must(http.DefaultClient.Do(reqBad))
	defer resBad.Body.Close()
	bBad, _ := io.ReadAll(resBad.Body)
	log.Printf("protected#bad-signature status=%d (expect 400), body=%s", resBad.StatusCode, string(bBad))

	reqReplay, _ := http.NewRequest("POST", protected, bytes.NewReader(cipher))
	reqReplay.Header = req.Header.Clone() 
	dump, _ = httputil.DumpRequestOut(reqReplay, true) 
	log.Println("---------- invalid packet(reuse nonce) -----------")
	log.Printf("HTTP OUT:\n%s\n", dump)
	resReplay := must(http.DefaultClient.Do(reqReplay))
	defer resReplay.Body.Close()
	bReplay, _ := io.ReadAll(resReplay.Body)
	log.Printf("protected#replay status=%d (expect 401), body=%s", resReplay.StatusCode, string(bReplay))

	// Wait for IdleTimeout to pass (assuming server IdleTimeout 2s)
	time.Sleep(2500 * time.Millisecond) // IdleTimeout + epsilon
	body2 := []byte(`{"op":"after-idle","ts":2}`)
	cipher2 := mustBytes(sess.Encrypt(body2)) // Client session OK even if not expired (server judges expiration)

	nonce2 := "n-" + uuid.NewString()
	date2  := time.Now().UTC().Format(time.RFC1123)
	digest2 := "sha-256=:" + b64(sha256Sum(cipher2)) + ":"

	reqIdle, _ := http.NewRequest("POST", protected, bytes.NewReader(cipher2))
	reqIdle.Header.Set("Content-Type", "application/json")
	reqIdle.Header.Set("Content-Digest", digest2)
	reqIdle.Header.Set("Date", date2)
	reqIdle.Header.Set("Signature-Input",
		fmt.Sprintf(`sig1=("@method" "@path" "host" "date" "content-digest");alg="hmac-sha256";keyid="%s";nonce="%s"`, kid, nonce2))
	covered2 := buildCovered(reqIdle, "sig1", kid, nonce2)
	sig2 := sess.SignCovered(covered2)
	reqIdle.Header.Set("Signature", "sig1=:"+base64.RawURLEncoding.EncodeToString(sig2)+":")

	log.Println("---------- invalid packet(expired session) -----------")
	log.Printf("HTTP OUT:\n%s\n", dump)
	resIdle := must(http.DefaultClient.Do(reqIdle))
	defer resIdle.Body.Close()
	bIdle, _ := io.ReadAll(resIdle.Body)
	log.Printf("protected#idle status=%d (expect 401) body=%s\n", resIdle.StatusCode, string(bIdle))
}

/* =======================
   3 functions that directly translate handshake.Client behavior to HTTP
   ======================= */

func sendInvitationHTTP(ctx context.Context, priv ed25519.PrivateKey, did, ctxID string, payload map[string]any) {
	data := mapToStruct(payload)
	msg := &a2a.Message{
		MessageId: uuid.NewString(),
		ContextId: ctxID,
		TaskId:    handshake.GenerateTaskID(handshake.Invitation),
		Role:      a2a.Role_ROLE_USER,
		Content:   []*a2a.Part{{Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: data}}}},
	}
	meta := mustMeta(clientKeypair, did, msg) // deterministic marshal → ed25519 sign (테스트 동일)
	sendToHTTP(ctx, msg, meta)
}

func sendRequestHTTP(ctx context.Context, priv ed25519.PrivateKey, did, ctxID, b64Packet string) {
	data := structB64(b64Packet) 
	msg := &a2a.Message{
		MessageId: uuid.NewString(),
		ContextId: ctxID,
		TaskId:    handshake.GenerateTaskID(handshake.Request),
		Role:      a2a.Role_ROLE_USER, 
		Content:   []*a2a.Part{{Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: data}}}},
	}
	meta := mustMeta(clientKeypair, did, msg)
	sendToHTTP(ctx, msg, meta)
}

func sendCompleteHTTP(ctx context.Context, priv ed25519.PrivateKey, did, ctxID string, payload map[string]any) {
	data := mapToStruct(payload)
	msg := &a2a.Message{
		MessageId: uuid.NewString(),
		ContextId: ctxID,
		TaskId:    handshake.GenerateTaskID(handshake.Complete),
		Role:      a2a.Role_ROLE_USER, // Same as test
		Content:   []*a2a.Part{{Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: data}}}},
	}
	meta := mustMeta(clientKeypair, did, msg)
	sendToHTTP(ctx, msg, meta)
}

func sendToHTTP(ctx context.Context, msg *a2a.Message, meta *structpb.Struct) {
	payload := &a2a.SendMessageRequest{Request: msg, Metadata: meta}
	b, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", base+"/v1/a2a:sendMessage", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res := must(http.DefaultClient.Do(req))
	defer res.Body.Close()
	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		log.Fatalf("SendMessage failed: %s", string(body))
	}
}

/* ==============
   Server outbound (JSON-RPC) reception (/rpc)
   ============== */

func startClientRPC() string {
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			JsonRPC string          `json:"jsonrpc"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
			ID      any             `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		if in.Method != "SendMessage" {
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc":"2.0","error":map[string]any{"code":-32601,"message":"no method"},"id":in.ID})
			return
		}
		var req a2a.SendMessageRequest
		if err := protojson.Unmarshal(in.Params, &req); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc":"2.0","error":map[string]any{"code":-32602,"message":"bad params: "+err.Error()},"id":in.ID})
			return
		}

		// Decrypt Response (encrypted b64) sent by server → ResponseMessage
		sb := req.GetRequest().GetContent()[0].GetData().GetData()
		if sb != nil {
			b64s := sb.Fields["b64"].GetStringValue()
			packet, _ := base64.RawURLEncoding.DecodeString(b64s)
			plain, _ := keys.DecryptWithEd25519Peer(clientPriv, packet)

			var res handshake.ResponseMessage
			_ = json.Unmarshal(plain, &res)
			raw := bytes.TrimSpace(res.EphemeralPubKey)
			if len(raw) > 0 && !bytes.Equal(raw, []byte("null")) {
				imp := formats.NewJWKImporter()
				pubAny, err := imp.ImportPublic(raw, sagecrypto.KeyFormatJWK)
				if err != nil {
					log.Printf("bad server eph JWK: %v", err)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"jsonrpc": "2.0",
						"error": map[string]any{"code": -32001, "message": "bad server eph JWK"},
						"id": in.ID,
					})
					return
				}
				pk, ok := pubAny.(*ecdh.PublicKey)
				if !ok || pk == nil {
					log.Printf("unexpected eph key type: %T", pubAny)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"jsonrpc": "2.0",
						"error": map[string]any{"code": -32002, "message": "unexpected eph key type"},
						"id": in.ID,
					})
					return
				}
				select { case gotEphCh <- pk.Bytes(): default: }
			}
			if res.KeyID != "" {
				select { case gotKidCh <- res.KeyID: default: }
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc":"2.0","result":map[string]string{"ok":"1"},"id":in.ID})
	})

	ln := mustListenAny()
	go (&http.Server{Handler:mux}).Serve(ln)
	return "http://"+ln.Addr().String()+"/rpc"
}

func waitServerEph() []byte {
	select { case b := <-gotEphCh: return b; case <-time.After(5*time.Second): log.Fatal("server eph timeout"); return nil }
}
func waitKID() string {
	select { case k := <-gotKidCh: return k; case <-time.After(5*time.Second): log.Fatal("kid timeout"); return "" }
}


func getJSON(url string, out any){ res := must(http.Get(url)); defer res.Body.Close(); _=json.NewDecoder(res.Body).Decode(out) }
func postJSON(url string, v any){ b,_ := json.Marshal(v); req,_ := http.NewRequest("POST", url, bytes.NewReader(b)); req.Header.Set("Content-Type","application/json"); res := must(http.DefaultClient.Do(req)); res.Body.Close() }
func must[T any](v T, err error) T { if err!=nil { log.Fatal(err) }; return v }
func mustListenAny() net.Listener { ln,err := net.Listen("tcp","127.0.0.1:0"); if err!=nil{log.Fatal(err)}; return ln }
func mustB64(s string) ed25519.PublicKey { b,err:=base64.RawURLEncoding.DecodeString(s); if err!=nil { log.Fatal(err) }; return ed25519.PublicKey(b) }
func mustX25519() *keys.X25519KeyPair { kp,err := keys.GenerateX25519KeyPair(); if err!=nil{log.Fatal(err)}; return kp.(*keys.X25519KeyPair) }
func mapToStruct(m map[string]any) *structpb.Struct { st,_ := structpb.NewStruct(m); return st }
func structB64(s string) *structpb.Struct { st,_ := structpb.NewStruct(map[string]any{"b64": s}); return st }
func mustMeta(kp sagecrypto.KeyPair, did string, m *a2a.Message) *structpb.Struct {
	bin,_ := proto.MarshalOptions{Deterministic:true}.Marshal(m)
	sig, _ := kp.Sign(bin)
	st,_ := structpb.NewStruct(map[string]any{
		"signature": base64.RawURLEncoding.EncodeToString(sig),
		"did": did,
		"client_pub_b64": base64.RawURLEncoding.EncodeToString(kp.PublicKey().(ed25519.PublicKey) ),
	})
	return st
}
func sha256Sum(b []byte) []byte { h:=sha256.New(); h.Write(b); return h.Sum(nil) }
func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
func mustBytes(b []byte, err error) []byte {
    if err != nil { log.Fatal(err) }
    return b
}


func buildCovered(r *http.Request, label, kid, nonce string) []byte {
    //@method" "@path" "host" "date" "content-digest"
    var b bytes.Buffer
    fmt.Fprintf(&b, "\"@method\": %s\n", strings.ToUpper(r.Method))
    fmt.Fprintf(&b, "\"@path\": %s\n", r.URL.RequestURI())
    fmt.Fprintf(&b, "\"host\": %s\n", r.Host)
    fmt.Fprintf(&b, "\"date\": %s\n", r.Header.Get("Date"))
    fmt.Fprintf(&b, "\"content-digest\": %s\n", r.Header.Get("Content-Digest"))
    fmt.Fprintf(&b, "\"@signature-params\": (\"@method\" \"@path\" \"host\" \"date\" \"content-digest\");alg=\"hmac-sha256\";keyid=\"%s\";nonce=\"%s\"",
        kid, nonce)
    return b.Bytes()
}