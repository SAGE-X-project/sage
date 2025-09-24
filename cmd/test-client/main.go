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
	"github.com/sage-x-project/sage/core/message"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/sage-x-project/sage/crypto/keys"
	sagedid "github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/handshake"
	"github.com/sage-x-project/sage/handshake/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const base = "http://127.0.0.1:8080"

var clientPriv ed25519.PrivateKey
var clientKeypair sagecrypto.KeyPair
var (
	gotEphCh = make(chan []byte, 1)
	gotKidCh = make(chan string, 1)
)

// dialGRPC dials the A2A gRPC server.
func dialGRPC(addr string) *grpc.ClientConn {
    conn, err := grpc.NewClient(
        addr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        log.Fatal(err)
    }
    return conn
}

// ---- gRPC inbox (client-side) ----
// This server receives server's outbound A2A SendMessage calls.
type peerInboxGRPC struct {
    a2a.UnimplementedA2AServiceServer
    clientPriv ed25519.PrivateKey
    gotEphCh   chan []byte
    gotKidCh   chan string
}

func (p *peerInboxGRPC) SendMessage(ctx context.Context, in *a2a.SendMessageRequest) (*a2a.SendMessageResponse, error) {
    // 1) Extract first DataPart as base64 bootstrap packet
    m := in.GetRequest()
    if m != nil && len(m.GetContent()) > 0 {
        if dp := m.GetContent()[0].GetData(); dp != nil && dp.GetData() != nil {
            if f := dp.Data.Fields["b64"]; f != nil {
                b64s := f.GetStringValue()
                pkt, _ := base64.RawURLEncoding.DecodeString(b64s)
                plain, _ := keys.DecryptWithEd25519Peer(p.clientPriv, pkt)

                // 2) Decode ResponseMessage (may include server eph JWK and/or KID)
                var rm handshake.ResponseMessage
                _ = json.Unmarshal(plain, &rm)

                // server ephemeral key (optional)
                if len(rm.EphemeralPubKey) > 0 && string(rm.EphemeralPubKey) != "null" {
                    pubAny, err := formats.NewJWKImporter().ImportPublic([]byte(rm.EphemeralPubKey), sagecrypto.KeyFormatJWK)
                    if err == nil {
                        if pk, ok := pubAny.(*ecdh.PublicKey); ok && pk != nil {
                            select { case p.gotEphCh <- pk.Bytes(): default: }
                        }
                    }
                }
                // KID (optional)
                if rm.KeyID != "" {
                    select { case p.gotKidCh <- rm.KeyID: default: }
                }
            }
        }
    }
    // Simple ACK
    return &a2a.SendMessageResponse{}, nil
}

// Start an A2A gRPC inbox server on a random loopback port.
func startPeerInboxGRPC(priv ed25519.PrivateKey, gotEphCh chan []byte, gotKidCh chan string) (addr string, stop func()) {
    lis, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil { log.Fatalf("inbox listen: %v", err) }
    gs := grpc.NewServer()
    a2a.RegisterA2AServiceServer(gs, &peerInboxGRPC{
        clientPriv: priv, gotEphCh: gotEphCh, gotKidCh: gotKidCh,
    })
    go gs.Serve(lis)
    return lis.Addr().String(), func(){ gs.GracefulStop() }
}


func main() {
	ctx := context.Background()

	// 0) 내 DID/서명키 (테스트와 동일하게 ed25519)
	clientKeypair, _ = keys.GenerateEd25519KeyPair()
	myDID := sagedid.AgentDID("did:sage:ethereum:agent001")
	clientPriv = clientKeypair.PrivateKey().(ed25519.PrivateKey) 

	// 1) 서버 ed25519 공개키(부트스트랩 암호화용)
	var sp struct{ Pub string }
	getJSON(base+"/debug/server-pub", &sp)
	serverPub := mustB64(sp.Pub)
	
	conn := dialGRPC("127.0.0.1:50051")
	defer conn.Close()
	hcli := handshake.NewClient(conn, clientKeypair)

	inboxAddr, stopInbox := startPeerInboxGRPC(clientPriv, gotEphCh, gotKidCh)
	defer stopInbox()
	postJSON(base+"/debug/set-outbound-grpc", map[string]string{"Target": inboxAddr})

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
	// handshake
	ctxID := "ctx-" + uuid.NewString()
	log.Printf("----- handshake -----")
	// 4-1) Invitation (clear JSON) — ROLE_USER
	log.Printf("Invitation ...")
	inv := handshake.InvitationMessage{
		BaseMessage: message.BaseMessage{
			ContextID: ctxID,
		},
	}
	if _, err := hcli.Invitation(ctx, inv, string(myDID)); err != nil {
		log.Fatalf("Invitation failed: %v", err)
	}

	// 4-2) Request (암호화 b64) — ROLE_USER
	// X25519 eph JWK 만들고 → 서버 ed25519로 부트스트랩 암호화 → {"b64": ...}
	log.Printf("Request ...")
	eph := mustX25519()
	jwk := must(formats.NewJWKExporter().ExportPublic(eph, sagecrypto.KeyFormatJWK))
	
	reqMsg := handshake.RequestMessage{
		BaseMessage: message.BaseMessage{
			ContextID: 	ctxID,    
		},
		EphemeralPubKey: json.RawMessage(jwk), 
	}
	if _, err := hcli.Request(ctx, reqMsg, serverPub, string(myDID)); err != nil {
		log.Fatalf("Request failed: %v", err)
	}	

	// 4-3) Complete (clear JSON) — ROLE_USER
	comMsg := handshake.CompleteMessage{
		BaseMessage: message.BaseMessage{
			ContextID: ctxID,
		},
	}
	if _, err := hcli.Complete(ctx, comMsg, string(myDID)); err != nil {
		log.Fatalf("Complete failed: %v", err)
	}
	log.Printf("Complete ... ✅")


	log.Printf("----- handshake -----")
	// 5) 서버 outbound: server eph / kid 수신 → 세션 생성
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
	// 6) /protected 호출: 1회 성공 → 2회(다른 nonce) 401 (서버 MaxMessages=1 가정)
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

	// IdleTimeout 경과 대기 (서버 IdleTimeout 2s 가정)
	time.Sleep(2500 * time.Millisecond) // IdleTimeout + epsilon
	body2 := []byte(`{"op":"after-idle","ts":2}`)
	cipher2 := mustBytes(sess.Encrypt(body2)) // 클라 세션은 안만료여도 OK(서버가 만료 판단)

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


func waitServerEph() []byte {
	select { case b := <-gotEphCh: return b; case <-time.After(5*time.Second): log.Fatal("server eph timeout"); return nil }
}
func waitKID() string {
	select { case k := <-gotKidCh: return k; case <-time.After(5*time.Second): log.Fatal("kid timeout"); return "" }
}


func getJSON(url string, out any){ res := must(http.Get(url)); defer res.Body.Close(); _=json.NewDecoder(res.Body).Decode(out) }
func postJSON(url string, v any){ b,_ := json.Marshal(v); req,_ := http.NewRequest("POST", url, bytes.NewReader(b)); req.Header.Set("Content-Type","application/json"); res := must(http.DefaultClient.Do(req)); res.Body.Close() }
func must[T any](v T, err error) T { if err!=nil { log.Fatal(err) }; return v }
func mustB64(s string) ed25519.PublicKey { b,err:=base64.RawURLEncoding.DecodeString(s); if err!=nil { log.Fatal(err) }; return ed25519.PublicKey(b) }
func mustX25519() *keys.X25519KeyPair { kp,err := keys.GenerateX25519KeyPair(); if err!=nil{log.Fatal(err)}; return kp.(*keys.X25519KeyPair) }
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