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
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sage-x-project/sage/crypto/keys"
	sagedid "github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/hpke"
	"github.com/sage-x-project/sage/session"
)

const base = "http://127.0.0.1:8080"

func main() {
	ctx := context.Background()

	// 0) 내 DID/서명키(ed25519)
	signKP, _ := keys.GenerateEd25519KeyPair()
	myDID := sagedid.AgentDID("did:sage:ethereum:client001")

	// 1) 서버 DID + KEM(X25519) 공개키 조회
	var srv struct {
		DID   string `json:"DID"`
		PubB64 string `json:"PubB64"`
	}
	getJSON(base+"/debug/server-kem", &srv)
	serverDID := sagedid.AgentDID(srv.DID)
	srvKemRaw := mustB64(srv.PubB64)
	srvKemPub := must(ecdh.X25519().NewPublicKey(srvKemRaw))

	// 2) 서버에 내 ed25519 검증키 등록 (서버가 gRPC 수신 서명 검증)
	postJSON(base+"/debug/register-agent", map[string]any{
		"DID":    string(myDID),
		"Name":   "Client",
		"Active": true,
		"PubB64": base64.RawURLEncoding.EncodeToString(signKP.PublicKey().(ed25519.PublicKey)),
	})

	// 3) gRPC 연결 + hpke.Client
	conn := must(grpc.Dial("127.0.0.1:50051", grpc.WithTransportCredentials(insecure.NewCredentials())))
	defer conn.Close()

	// 클라 측 resolver: peer(server) DID → HPKE KEM pub 반환
	res := newStaticResolver()
	res.Put(serverDID, srvKemPub)

	sessMgr := session.NewManager()
	cli := hpke.NewClient(conn, res, signKP, string(myDID), hpke.DefaultInfoBuilder{}, sessMgr)

	// 4) HPKE Initialize → exporter 합의, 세션 생성, kid 수신/바인딩
	ctxID := "ctx-" + uuid.NewString()
	kid, err := cli.Initialize(ctx, ctxID, string(myDID), string(serverDID))
	if err != nil { log.Fatalf("Initialize failed: %v", err) }
	log.Printf("HPKE ready: kid=%s", kid)

	// 5) 세션으로 /protected 호출(HMAC+AEAD)
	sess, ok := sessMgr.GetByKeyID(kid)
	if !ok { log.Fatal("session not found by kid") }

	body := []byte(`{"op":"ping","ts":1} TEST MESSAGE`)
	cipher := mustBytes(sess.Encrypt(body))

	nonce := "n-" + uuid.NewString()
	date := time.Now().UTC().Format(time.RFC1123)
	digest := "sha-256=:" + b64(sha256Sum(cipher)) + ":"

	url := base + "/protected"
	req, _ := http.NewRequest("POST", url, bytes.NewReader(cipher))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Digest", digest)
	req.Header.Set("Date", date)
	req.Header.Set("Signature-Input",
		fmt.Sprintf(`sig1=("@method" "@path" "host" "date" "content-digest");alg="hmac-sha256";keyid="%s";nonce="%s"`, kid, nonce))
	covered := buildCovered(req, "sig1", kid, nonce)
	sig := sess.SignCovered(covered)
	req.Header.Set("Signature", "sig1=:"+base64.RawURLEncoding.EncodeToString(sig)+":")

	dump, _ := httputil.DumpRequestOut(req, true)
	log.Println("----- sending protected -----")
	log.Printf("HTTP OUT:\n%s\n", dump)

	resHTTP := must(http.DefaultClient.Do(req))
	defer resHTTP.Body.Close()
	b1, _ := io.ReadAll(resHTTP.Body)
	log.Printf("protected status=%d body=%s\n", resHTTP.StatusCode, string(b1))

    reqEmpty, _ := http.NewRequest("POST", url, bytes.NewReader([]byte("")))
    reqEmpty.Header = req.Header.Clone()
    reqEmpty.Header.Set("Signature-Input",
        fmt.Sprintf(`sig1=("@method" "@path" "host" "date" "content-digest");alg="hmac-sha256";keyid="%s";nonce="%s"`, kid, "n-"+uuid.NewString()))
    dump, _ = httputil.DumpRequestOut(reqEmpty, true) 
    log.Println("---------- invalid packet(empty body) -----------")
    log.Printf("HTTP OUT:\n%s\n", dump)
    res2 := must(http.DefaultClient.Do(reqEmpty))
    defer res2.Body.Close()
    log.Printf("protected#empty body status=%d (expect 401)\n\n", res2.StatusCode)


    reqBad, _ := http.NewRequest("POST", url, bytes.NewReader([]byte(`{"op":"bad"}`)))
    dump, _ = httputil.DumpRequestOut(reqBad, true) 
    log.Println("---------- invalid packet(bad Signature-Input) -----------")
    log.Printf("HTTP OUT:\n%s\n", dump)
    resBad := must(http.DefaultClient.Do(reqBad))
    defer resBad.Body.Close()
    bBad, _ := io.ReadAll(resBad.Body)
    log.Printf("protected#bad-signature status=%d (expect 401), body=%s", resBad.StatusCode, string(bBad))

    reqReplay, _ := http.NewRequest("POST", url, bytes.NewReader(cipher))
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

    reqIdle, _ := http.NewRequest("POST", url, bytes.NewReader(cipher2))
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

/* ---------- helpers ---------- */

type staticResolver struct{ m map[sagedid.AgentDID]any }
func newStaticResolver() *staticResolver { return &staticResolver{m: map[sagedid.AgentDID]any{}} }
func (s *staticResolver) Put(id sagedid.AgentDID, pub any) { s.m[id] = pub }
func (s *staticResolver) ResolvePublicKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	v, ok := s.m[did]
	if !ok { return nil, fmt.Errorf("not found: %s", did) }
	return v, nil
}
// 나머지 인터페이스 메서드는 사용 안 함
func (s *staticResolver) Resolve(context.Context, sagedid.AgentDID) (*sagedid.AgentMetadata, error) { return nil, fmt.Errorf("not implemented") }
func (s *staticResolver) VerifyMetadata(context.Context, sagedid.AgentDID, *sagedid.AgentMetadata) (*sagedid.VerificationResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *staticResolver) ListAgentsByOwner(context.Context, string) ([]*sagedid.AgentMetadata, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *staticResolver) Search(context.Context, sagedid.SearchCriteria) ([]*sagedid.AgentMetadata, error) {
	return nil, fmt.Errorf("not implemented")
}

func getJSON(url string, out any){ res := must(http.Get(url)); defer res.Body.Close(); _=json.NewDecoder(res.Body).Decode(out) }
func postJSON(url string, v any){ b,_ := json.Marshal(v); req,_ := http.NewRequest("POST", url, bytes.NewReader(b)); req.Header.Set("Content-Type","application/json"); res := must(http.DefaultClient.Do(req)); res.Body.Close() }
func must[T any](v T, err error) T { if err!=nil { log.Fatal(err) }; return v }
func mustB64(s string) []byte { b,err:=base64.RawURLEncoding.DecodeString(s); if err!=nil { log.Fatal(err) }; return b }
func sha256Sum(b []byte) []byte { h:=sha256.New(); h.Write(b); return h.Sum(nil) }
func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
func mustBytes(b []byte, err error) []byte { if err!=nil { log.Fatal(err) }; return b }

func buildCovered(r *http.Request, label, kid, nonce string) []byte {
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
