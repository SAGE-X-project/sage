package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"sync/atomic"
	"time"

	a2a "github.com/a2aproject/a2a/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/sage-x-project/sage/core/rfc9421"
	"github.com/sage-x-project/sage/crypto/keys"
	sagedid "github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/handshake"
	sessioninit "github.com/sage-x-project/sage/internal"
	"github.com/sage-x-project/sage/session"
)

var seq uint64

// ---- logging helpers ----
func banner(prefix, title string) {
	log.Printf("\n==================== %s %s ====================\n", prefix, title)
}

func dumpHeaders(prefix string, r *http.Request) {
	// Dump only headers/request line (body printed separately)
	dump, _ := httputil.DumpRequest(r, false)
	log.Printf("%s HEADERS:\n%s", prefix, dump)
}

func dumpCipherRaw(prefix string, b []byte) {
	// Print raw request body as UTF-8 (may look garbled)
	log.Printf("%s CIPHER (raw/utf8) len=%d:\n%s", prefix, len(b), string(b))
}

func dumpCipherB64(prefix string, b []byte) {
	log.Printf("%s CIPHER (base64) len=%d:\n%s", prefix, len(b), base64.StdEncoding.EncodeToString(b))
}

func dumpCipherHexPreview(prefix string, b []byte, max int) {
	if len(b) > max {
		log.Printf("%s CIPHER (hex preview %d/%d):\n%s", prefix, max, len(b), hex.Dump(b[:max]))
	} else {
		log.Printf("%s CIPHER (hex):\n%s", prefix, hex.Dump(b))
	}
}

func dumpPlainPretty(prefix string, plain []byte) {
	var out bytes.Buffer
	if json.Indent(&out, plain, "", "  ") == nil {
		log.Printf("%s DECRYPTED (pretty JSON):\n%s", prefix, out.String())
	} else {
		log.Printf("%s DECRYPTED (raw):\n%s", prefix, string(plain))
	}
}

type memoryResolver struct {
	mu    sync.RWMutex
	store map[string]*sagedid.AgentMetadata
}

func newMemoryResolver() *memoryResolver {
	return &memoryResolver{store: make(map[string]*sagedid.AgentMetadata)}
}

func (m *memoryResolver) add(meta *sagedid.AgentMetadata) {
	m.mu.Lock()
	m.store[string(meta.DID)] = meta
	m.mu.Unlock()
}

func (m *memoryResolver) Resolve(ctx context.Context, did sagedid.AgentDID) (*sagedid.AgentMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	meta, ok := m.store[string(did)]
	if !ok {
		return nil, fmt.Errorf("DID not found: %s", did)
	}
	return meta, nil
}

func (m *memoryResolver) ResolvePublicKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	meta, err := m.Resolve(ctx, did)
	if err != nil {
		return nil, err
	}
	if !meta.IsActive {
		return nil, sagedid.ErrInactiveAgent
	}
	return meta.PublicKey, nil
}

func (m *memoryResolver) VerifyMetadata(ctx context.Context, did sagedid.AgentDID, metadata *sagedid.AgentMetadata) (*sagedid.VerificationResult, error) {
	meta, err := m.Resolve(ctx, did)
	if err != nil {
		return nil, err
	}
	valid := meta.Name == metadata.Name && meta.IsActive == metadata.IsActive
	return &sagedid.VerificationResult{Valid: valid, Agent: meta, VerifiedAt: time.Now()}, nil
}

func (m *memoryResolver) ListAgentsByOwner(context.Context, string) ([]*sagedid.AgentMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	agents := make([]*sagedid.AgentMetadata, 0, len(m.store))
	for _, meta := range m.store {
		agents = append(agents, meta)
	}
	return agents, nil
}

func (m *memoryResolver) Search(context.Context, sagedid.SearchCriteria) ([]*sagedid.AgentMetadata, error) {
	return nil, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	serverKeyPair, err := keys.GenerateEd25519KeyPair()
	if err != nil {
		log.Fatalf("keygen: %v", err)
	}

	sessMgr := session.NewManager()
	sessMgr.SetDefaultConfig(session.Config{
		MaxAge:      time.Hour,
		IdleTimeout: 2 * time.Second,
		MaxMessages: 10,
	})
	events := sessioninit.NewCreator(sessMgr)

	resolver := newMemoryResolver()
	multiResolver := sagedid.NewMultiChainResolver()
	multiResolver.AddResolver(sagedid.ChainEthereum, resolver)

	hs := handshake.NewServer(serverKeyPair, events, multiResolver, nil, 0)

	grpcLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	grpcSrv := grpc.NewServer()
	a2a.RegisterA2AServiceServer(grpcSrv, hs)
	go func() {
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	conn, err := grpc.NewClient(grpcLis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("grpc self-dial: %v", err)
	}
	grpcCli := a2a.NewA2AServiceClient(conn)

	mux := http.NewServeMux()
	var srv *http.Server
	// shutdownOnce := &sync.Once{}
	// scheduleShutdown := func(delay time.Duration) {
	// 	shutdownOnce.Do(func() {
	// 		go func() {
	// 			time.Sleep(delay)
	// 			log.Println("auto shutdown triggered")
	// 			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	// 			defer cancel()
	// 			if err := srv.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
	// 				log.Printf("http shutdown error: %v", err)
	// 			}
	// 			grpcSrv.Stop()
	// 			_ = conn.Close()
	// 		}()
	// 	})
	// }

	// gRPC proxy
	mux.HandleFunc("/v1/a2a:sendMessage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var req a2a.SendMessageRequest
		if err := protojson.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}

		resp, err := grpcCli.SendMessage(r.Context(), &req)
		if err != nil {
			http.Error(w, "grpc error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		out, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(out)
	})

	// protected endpoint: packet logging + verification + decryption
	mux.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		id := atomic.AddUint64(&seq, 1)
		prefix := fmt.Sprintf("[req#%03d]", id)
		banner(prefix, "INCOMING /protected")

		// 1) capture ciphertext first
		cipherBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		_ = r.Body.Close()
		// Safely restore body even if verification library ignores it
		r.Body = io.NopCloser(bytes.NewReader(cipherBody))

		// 2) request line and headers
		dumpHeaders(prefix, r)

		// 3) ciphertext: raw -> base64 -> hex
		dumpCipherRaw(prefix, cipherBody)
		dumpCipherB64(prefix, cipherBody)
		dumpCipherHexPreview(prefix, cipherBody, 64)

		// ===== verification/decryption =====
		sigInputs, err := rfc9421.ParseSignatureInput(r.Header.Get("Signature-Input"))
		if err != nil {
			http.Error(w, "invalid Signature-Input", http.StatusBadRequest)
			return
		}
		params := sigInputs["sig1"]
		if params == nil || params.KeyID == "" {
			http.Error(w, "missing key id", http.StatusBadRequest)
			return
		}
		if params.Nonce == "" {
			http.Error(w, "missing nonce", http.StatusBadRequest)
			return
		}
		if sessMgr.ReplayGuardSeenOnce(params.KeyID, params.Nonce) {
			http.Error(w, "replay", http.StatusUnauthorized)
			return
		}
		sess, ok := sessMgr.GetByKeyID(params.KeyID)
		if !ok {
			http.Error(w, "no session", http.StatusUnauthorized)
			return
		}

		agentDID := r.Header.Get("X-Agent-DID")
		if agentDID == "" {
			http.Error(w, "missing agent DID", http.StatusBadRequest)
			return
		}
		meta, err := resolver.Resolve(r.Context(), sagedid.AgentDID(agentDID))
		if err != nil {
			http.Error(w, "unknown agent", http.StatusUnauthorized)
			return
		}
		pubKey, ok := meta.PublicKey.(ed25519.PublicKey)
		if !ok {
			http.Error(w, "invalid agent key", http.StatusInternalServerError)
			return
		}

		verifier := rfc9421.NewHTTPVerifier()
		if err := verifier.VerifyRequest(r, pubKey, &rfc9421.HTTPVerificationOptions{
			SignatureName: "sig1", MaxAge: 2 * time.Minute,
		}); err != nil {
			http.Error(w, "signature verify failed: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// 4) decrypt and print
		plain, err := sess.Decrypt(cipherBody)
		if err != nil {
			http.Error(w, "decrypt: "+err.Error(), http.StatusUnauthorized)
			return
		}
		dumpPlainPretty(prefix, plain)

		// 5) response
		responsePayload := []byte(fmt.Sprintf("{\"ok\":true,\"echo\":%s}", plain))
		cipherOut, err := sess.Encrypt(responsePayload)
		if err != nil {
			http.Error(w, "encrypt response: "+err.Error(), http.StatusInternalServerError)
			return
		}
		resp := map[string]string{"cipher_b64": base64.StdEncoding.EncodeToString(cipherOut)}

		banner(prefix, "OUTGOING RESPONSE")
		log.Printf("%s STATUS: %d", prefix, http.StatusOK)
		log.Printf("%s BODY(JSON): %v", prefix, resp)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)

		// Automatic test shutdown (disable if desired)
		// scheduleShutdown(3 * time.Second)
	})

	// Always respond with {"pub":"<base64url>"} so the client can parse it as a string
	mux.HandleFunc("/debug/server-pub", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
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
			PubB64 string `json:"PubB64"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}
		b, err := base64.RawURLEncoding.DecodeString(in.PubB64)
		if err != nil || len(b) != ed25519.PublicKeySize {
			http.Error(w, "bad pub", http.StatusBadRequest)
			return
		}
		meta := &sagedid.AgentMetadata{
			DID:       sagedid.AgentDID(in.DID),
			Name:      in.Name,
			IsActive:  in.Active,
			PublicKey: ed25519.PublicKey(b),
		}
		resolver.add(meta)
		w.WriteHeader(http.StatusNoContent)
	})

	httpLis, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatalf("http listen: %v", err)
	}

	srv = &http.Server{Handler: mux}

	log.Println("gRPC :", grpcLis.Addr())
	log.Println("HTTP :8080  (POST /v1/a2a:sendMessage, /protected)")

	if err := srv.Serve(httpLis); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http serve: %v", err)
	}
	log.Println("server exiting")
}
