package main

import (
	"bytes"
	"context"
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
	"github.com/test-go/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/sage-x-project/sage/crypto/keys"
	sagedid "github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/handshake"
	"github.com/sage-x-project/sage/handshake/session"
	sessioninit "github.com/sage-x-project/sage/internal"
)
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
	serverKeyPair, err := keys.GenerateEd25519KeyPair()
	if err != nil { log.Fatalf("keygen: %v", err) }
	sessMgr := session.NewManager()
	sessMgr.SetDefaultConfig(session.Config{
		MaxAge:      time.Hour,
		IdleTimeout: 2 * time.Second,
		MaxMessages: 10,
	})
	events := sessioninit.NewCreator(sessMgr)

	ethResolver := new(mockResolver)
	multiResolver := sagedid.NewMultiChainResolver()
	multiResolver.AddResolver(sagedid.ChainEthereum, ethResolver)


	grpcLis, err := net.Listen("tcp", "127.0.0.1:50051")
	if err != nil { log.Fatal(err) }
	grpcSrv := grpc.NewServer(grpc.ChainUnaryInterceptor(unaryLog()))
	srv := handshake.NewServer(serverKeyPair, events, nil, multiResolver, nil)

	a2a.RegisterA2AServiceServer(grpcSrv, srv)
	go func() {
		log.Println("gRPC :50051")
		if err := grpcSrv.Serve(grpcLis); err != nil { log.Fatalf("grpc serve: %v", err) }
	}()

	mux := http.NewServeMux()


	// 2) 보호 API: kid/nonce → 세션 복호화(+만료/리플레이 차단)
	mux.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		dump2, _ := httputil.DumpRequest(r, true)
		log.Println("----------  received packet -----------")
		log.Printf("HTTP IN:\n%s", dump2)
		ts := time.Now()
		si := r.Header.Get("Signature-Input")
		kid := pick(si, `keyid="`)
		nonce := pick(si, `nonce="`)
		if kid == "" || nonce == "" { http.Error(w, "bad signature-input", 400); return }
		if sessMgr.ReplayGuardSeenOnce(kid, nonce) { http.Error(w, "replay", 401); return }
		
		sess, ok := sessMgr.GetByKeyID(kid)
		if !ok { http.Error(w, "no session", 401); return }
		log.Printf(`Get session ID %s`, sess.GetID())
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
		log.Println("decrypt body ✅")
		if err != nil { http.Error(w, "decrypt: "+err.Error(), 401); return }
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(plain)
		log.Printf("[/protected] 200 OK kid=%s nonce=%s took=%s body=%s\n\n", kid, nonce, time.Since(ts), string(plain))
	})

	mux.HandleFunc("/debug/server-pub", func(w http.ResponseWriter, r *http.Request) {
		pub := serverKeyPair.PublicKey().(ed25519.PublicKey)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"pub": base64.RawURLEncoding.EncodeToString(pub),
		})
	})

	mux.HandleFunc("/debug/register-agent", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			DID    string `json:"DID"`
			Name   string `json:"Name"`
			Active bool   `json:"Active"`
			PubB64 string `json:"PubB64"` // ed25519 raw pubkey (base64url)
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json: "+err.Error(), 400); return
		}
		b, err := base64.RawURLEncoding.DecodeString(in.PubB64)
		if err != nil || len(b) != ed25519.PublicKeySize {
			http.Error(w, "bad pub", 400); return
		}

		did := sagedid.AgentDID(in.DID)
		meta := &sagedid.AgentMetadata{
			DID:       did,
			Name:      in.Name,
			IsActive:  in.Active,
			PublicKey: ed25519.PublicKey(b),
		}

		// mock 기대치 주입 (ctx는 뭐가 올지 몰라서 Any)
		ethResolver.
			On("Resolve", mock.Anything, did).
			Return(meta, nil) // 호출 횟수 제한 X (포크라 Maybe 없으니 이대로)
		w.WriteHeader(204)
	})


	// Allow wiring outbound to a peer's A2A gRPC inbox: POST {"Target":"127.0.0.1:6xxx"}
	mux.HandleFunc("/debug/set-outbound-grpc", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Target string }
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json: "+err.Error(), 400); return
		}
		if in.Target == "" {
			http.Error(w, "missing Target", 400); return
		}

		// Dial peer inbox (gRPC) and set as outbound client.
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		outConn, err := grpc.DialContext(ctx, in.Target,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			http.Error(w, "dial outbound: "+err.Error(), 500); return
		}

		// IMPORTANT: keep a reference somewhere to close later if needed.
		srv.SetOutbound(a2a.NewA2AServiceClient(outConn))
		log.Printf("[OUTBOUND gRPC] set to %s", in.Target)
		w.WriteHeader(204)
	})


	httpSrv := &http.Server{Addr: ":8080", Handler: mux}
	log.Println("HTTP :8080  (POST /v1/a2a:sendMessage, /protected)")
	log.Fatal(httpSrv.ListenAndServe())
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
    if !strings.HasPrefix(sigHdr, "sig1=:") || !strings.HasSuffix(sigHdr, ":") {
        return "", false
    }
    inner := strings.TrimSuffix(strings.TrimPrefix(sigHdr, "sig1=:"), ":")
    if inner == "" { return "", false }
    return inner, true
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

type mockResolver struct {
	mock.Mock
}

func (m *mockResolver) Resolve(ctx context.Context, did sagedid.AgentDID) (*sagedid.AgentMetadata, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sagedid.AgentMetadata), args.Error(1)
}

func (m *mockResolver) ResolvePublicKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
}

func (m *mockResolver) VerifyMetadata(ctx context.Context, did sagedid.AgentDID, metadata *sagedid.AgentMetadata) (*sagedid.VerificationResult, error) {
	args := m.Called(ctx, did, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sagedid.VerificationResult), args.Error(1)
}

func (m *mockResolver) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*sagedid.AgentMetadata, error) {
	args := m.Called(ctx, ownerAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sagedid.AgentMetadata), args.Error(1)
}

func (m *mockResolver) Search(ctx context.Context, criteria sagedid.SearchCriteria) ([]*sagedid.AgentMetadata, error) {
	args := m.Called(ctx, criteria)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sagedid.AgentMetadata), args.Error(1)
}
