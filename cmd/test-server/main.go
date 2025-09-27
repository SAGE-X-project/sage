package main

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/sage-x-project/sage/crypto/keys"
	sagedid "github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/hpke"
	"github.com/sage-x-project/sage/session"
)

const serverDID = "did:sage:ethereum:server001"

func unaryLog() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
		start := time.Now()
		var ctxID, taskID string
		if r, ok := req.(*a2a.SendMessageRequest); ok && r.GetRequest() != nil {
			ctxID = r.GetRequest().GetContextId()
			taskID = r.GetRequest().GetTaskId()
		}
		resp, err := h(ctx, req)
		log.Printf("[gRPC IN] %s ctx=%s task=%s code=%s dur=%s",
			info.FullMethod, ctxID, taskID, status.Code(err), time.Since(start))
		return resp, err
	}
}

func main() {
	// 1) KEM 키(X25519) — HPKE 수신측
	kemKP, err := keys.GenerateX25519KeyPair()
	if err != nil { log.Fatalf("kem keygen: %v", err) }

	// 2) 세션 매니저
	sessMgr := session.NewManager()
	sessMgr.SetDefaultConfig(session.Config{
		MaxAge:      time.Hour,
		IdleTimeout: 2 * time.Second,
		MaxMessages: 10,
	})

	// 3) DID resolver (간단 구현: 클라 ed25519 검증키 등록/조회)
	res := newMemResolver()

	// 4) gRPC 서버 시작
	lis, err := net.Listen("tcp", "127.0.0.1:50051")
	if err != nil { log.Fatal(err) }
	gs := grpc.NewServer(grpc.ChainUnaryInterceptor(unaryLog()))
	a2a.RegisterA2AServiceServer(gs, hpke.NewServer(kemKP, sessMgr, serverDID, res, nil))
	go func() {
		log.Println("gRPC :50051 (A2A)")
		if err := gs.Serve(lis); err != nil { log.Fatalf("grpc serve: %v", err) }
	}()

	// 5) HTTP 컨트롤 + 보호 API
	mux := http.NewServeMux()

	// 서버 DID + HPKE KEM 공개키(raw 32B b64url) 제공
	mux.HandleFunc("/debug/server-kem", func(w http.ResponseWriter, r *http.Request) {
		raw := kemKP.PublicKey().(*ecdh.PublicKey).Bytes()
		_ = json.NewEncoder(w).Encode(map[string]string{
			"DID":   serverDID,
			"PubB64": base64.RawURLEncoding.EncodeToString(raw),
		})
	})

	// 클라이언트 DID의 ed25519 검증키 등록 (서명 검증용)
	mux.HandleFunc("/debug/register-agent", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			DID    string `json:"DID"`
			Name   string `json:"Name"`
			Active bool   `json:"Active"`
			PubB64 string `json:"PubB64"` // ed25519 raw pubkey (b64url)
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json: "+err.Error(), 400); return
		}
		b, err := base64.RawURLEncoding.DecodeString(in.PubB64)
		if err != nil || len(b) != ed25519.PublicKeySize {
			http.Error(w, "bad pub", 400); return
		}
		res.PutEd25519(sagedid.AgentDID(in.DID), ed25519.PublicKey(b))
		w.WriteHeader(204)
	})

	// 보호 API: RFC-9421 스타일 HMAC 검증 + 세션 복호화
	mux.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		dump2, _ := httputil.DumpRequest(r, true)
		log.Println("----- received packet -----")
		log.Printf("HTTP IN:\n%s", dump2)

		ts := time.Now()
		si := r.Header.Get("Signature-Input")
		kid := pick(si, `keyid="`)
		nonce := pick(si, `nonce="`)
		if kid == "" || nonce == "" {
			http.Error(w, "bad signature-input", 401); return
		}
		if sessMgr.ReplayGuardSeenOnce(kid, nonce) {
			http.Error(w, "replay", 401); return
		}
		sess, ok := sessMgr.GetByKeyID(kid)
		if !ok {
			http.Error(w, "no session", 401); return
		}

		sigB64, ok := parseSig1(r.Header.Get("Signature"))
		if !ok { http.Error(w, "missing signature", 400); return }
		sig, err := base64.RawURLEncoding.DecodeString(sigB64)
		if err != nil { http.Error(w, "bad signature", 400); return }

		covered := buildCovered(r, "sig1", kid, nonce)
		if err := sess.VerifyCovered(covered, sig); err != nil {
			http.Error(w, "signature verify failed", 401); return
		}
		log.Println("verify signature ✅")

		cipher, _ := io.ReadAll(r.Body)
		plain, err := sess.Decrypt(cipher)
		if err != nil { http.Error(w, "decrypt: "+err.Error(), 401); return }
		log.Println("decrypt body ✅")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(plain)
		log.Printf("[/protected] 200 OK kid=%s nonce=%s took=%s body=%s\n\n", kid, nonce, time.Since(ts), string(plain))
	})

	httpSrv := &http.Server{Addr: ":8080", Handler: mux}
	log.Println("HTTP :8080  (/debug/server-kem, /debug/register-agent, /protected)")
	log.Fatal(httpSrv.ListenAndServe())
}

/* ---------- helpers ---------- */

type memResolver struct {
	ed map[sagedid.AgentDID]ed25519.PublicKey
}
func newMemResolver() *memResolver { return &memResolver{ed: map[sagedid.AgentDID]ed25519.PublicKey{}} }
func (m *memResolver) PutEd25519(id sagedid.AgentDID, pk ed25519.PublicKey) {
	m.ed[id] = pk
}
func (m *memResolver) ResolvePublicKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	pk, ok := m.ed[did]
	if !ok { return nil, fmt.Errorf("no ed25519 pub for %s", did) }
	return pk, nil
}
// 나머지는 사용 안 함
func (m *memResolver) Resolve(context.Context, sagedid.AgentDID) (*sagedid.AgentMetadata, error) { return nil, fmt.Errorf("not implemented") }
func (m *memResolver) VerifyMetadata(context.Context, sagedid.AgentDID, *sagedid.AgentMetadata) (*sagedid.VerificationResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *memResolver) ListAgentsByOwner(context.Context, string) ([]*sagedid.AgentMetadata, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *memResolver) Search(context.Context, sagedid.SearchCriteria) ([]*sagedid.AgentMetadata, error) {
	return nil, fmt.Errorf("not implemented")
}

func pick(s, pre string) string {
	i := strings.Index(s, pre); if i < 0 { return "" }
	j := i + len(pre)
	k := strings.Index(s[j:], `"`)
	if k < 0 { return "" }
	return s[j : j+k]
}
func parseSig1(sigHdr string) (string, bool) {
	sigHdr = strings.TrimSpace(sigHdr)
	if !strings.HasPrefix(sigHdr, "sig1=:") || !strings.HasSuffix(sigHdr, ":") { return "", false }
	inner := strings.TrimSuffix(strings.TrimPrefix(sigHdr, "sig1=:"), ":")
	if inner == "" { return "", false }
	return inner, true
}
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
