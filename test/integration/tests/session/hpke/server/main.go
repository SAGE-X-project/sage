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

//go:build integration && a2a
// +build integration,a2a

package main

import (
	"context"
	"crypto/ecdh"
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

	a2apb "github.com/a2aproject/a2a/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	sagedid "github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/hpke"
	"github.com/sage-x-project/sage/pkg/agent/session"
	a2atransport "github.com/sage-x-project/sage/pkg/agent/transport/a2a"
)

const (
	httpAddr = "127.0.0.1:8080"
	grpcAddr = "127.0.0.1:18080"
)

var seq uint64

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Server signing key plus HPKE recipient KEM (X25519)
	serverSignKP, _ := keys.GenerateEd25519KeyPair()
	serverKEMKP, _ := keys.GenerateX25519KeyPair()
	serverDID := string(sagedid.GenerateDID(sagedid.ChainEthereum, "server"))

	// Session manager with a short idle timeout (2s) to exercise expiry logic
	sessMgr := session.NewManager()
	sessMgr.SetDefaultConfig(session.Config{
		MaxAge:      time.Hour,
		IdleTimeout: 2 * time.Second,
		MaxMessages: 1000,
	})

	// In-memory resolver publishes the server KEM public key and tracks registered clients
	resolver := newMemResolver()
	resolver.add(&sagedid.AgentMetadata{
		DID:          sagedid.AgentDID(serverDID),
		Name:         "HPKE Server",
		IsActive:     true,
		PublicKey:    serverSignKP.PublicKey(),
		PublicKEMKey: serverKEMKP.PublicKey(),
	})

	// gRPC: HPKE Server with A2A adapter
	grpcLis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}
	grpcSrv := grpc.NewServer()
	hpkeSrv := hpke.NewServer(serverSignKP, sessMgr, serverDID, resolver, &hpke.ServerOpts{
		MaxSkew: 2 * time.Minute,
		Info:    hpke.DefaultInfoBuilder{},
		KEM:     serverKEMKP,
	})
	// Wrap the HPKE server with A2A server adapter
	a2aAdapter := a2atransport.NewA2AServerAdapter(hpkeSrv)
	a2apb.RegisterA2AServiceServer(grpcSrv, a2aAdapter)
	go func() { _ = grpcSrv.Serve(grpcLis) }()

	// gRPC client used by the HTTP → gRPC proxy
	grpcConn, _ := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	a2aCli := a2apb.NewA2AServiceClient(grpcConn)

	// HTTP server
	mux := http.NewServeMux()

	// A2A JSON → gRPC proxy (debug helper)
	mux.HandleFunc("/v1/a2a:sendMessage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		var req a2apb.SendMessageRequest
		if err := protojson.Unmarshal(mustReadAll(r.Body), &req); err != nil {
			http.Error(w, "bad json: "+err.Error(), 400)
			return
		}
		resp, err := a2aCli.SendMessage(r.Context(), &req)
		if err != nil {
			http.Error(w, "grpc error: "+err.Error(), 500)
			return
		}
		out, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(out)
	})

	// Protected endpoint: verify RFC9421 signature and decrypt/encrypt session payloads
	mux.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		id := atomic.AddUint64(&seq, 1)
		pfx := fmt.Sprintf("[req#%03d]", id)
		dump, _ := httputil.DumpRequest(r, false)
		log.Printf("%s HEADERS:\n%s", pfx, dump)

		cipher, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		// Parse Signature-Input
		inp, err := rfc9421.ParseSignatureInput(r.Header.Get("Signature-Input"))
		if err != nil {
			http.Error(w, "invalid Signature-Input", 400)
			return
		}
		p := inp["sig1"]
		if p == nil || p.KeyID == "" || p.Nonce == "" {
			http.Error(w, "missing sig params", 400)
			return
		}

		// Replay protection
		if sessMgr.ReplayGuardSeenOnce(p.KeyID, p.Nonce) {
			http.Error(w, "replay", 401)
			return
		}

		// Look up session by key ID
		sess, ok := sessMgr.GetByKeyID(p.KeyID)
		if !ok {
			http.Error(w, "no session", 401)
			return
		}

		// Verify HTTP signature with the client DID
		cliDID := r.Header.Get("X-Agent-DID")
		if cliDID == "" {
			http.Error(w, "missing X-Agent-DID", 400)
			return
		}
		meta, err := resolver.Resolve(r.Context(), sagedid.AgentDID(cliDID))
		if err != nil {
			http.Error(w, "unknown agent", 401)
			return
		}
		pub, ok2 := meta.PublicKey.(ed25519.PublicKey)
		if !ok2 {
			http.Error(w, "bad agent key", 500)
			return
		}
		ver := rfc9421.NewHTTPVerifier()
		if err := ver.VerifyRequest(r, pub, &rfc9421.HTTPVerificationOptions{
			SignatureName: "sig1", MaxAge: 2 * time.Minute,
		}); err != nil {
			http.Error(w, "sig verify failed: "+err.Error(), 401)
			return
		}

		// Decrypt ciphertext
		plain, err := sess.Decrypt(cipher)
		if err != nil {
			http.Error(w, "decrypt: "+err.Error(), 401)
			return
		}
		log.Printf("%s CIPHER(len=%d) hex-preview:\n%s", pfx, len(cipher), hex.Dump(cipher[:min(64, len(cipher))]))
		log.Printf("%s DECRYPTED: %s", pfx, string(plain))

		// Encrypt response (server → client)
		reply := []byte(fmt.Sprintf(`{"from":"server","ok":true,"echo":%s}`, plain))
		ct, _ := sess.Encrypt(reply)
		resp := map[string]string{"cipher_b64": base64.StdEncoding.EncodeToString(ct)}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Debug endpoints
	mux.HandleFunc("/debug/server-pub", func(w http.ResponseWriter, r *http.Request) {
		pub := serverSignKP.PublicKey().(ed25519.PublicKey)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"pub": base64.RawURLEncoding.EncodeToString(pub),
		})
	})
	mux.HandleFunc("/debug/kem-pub", func(w http.ResponseWriter, r *http.Request) {
		pub := serverKEMKP.PublicKey().(*ecdh.PublicKey).Bytes()
		_ = json.NewEncoder(w).Encode(map[string]string{"kem": base64.RawURLEncoding.EncodeToString(pub)})
	})
	mux.HandleFunc("/debug/server-did", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"did": serverDID})
	})
	mux.HandleFunc("/debug/register-agent", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			DID    string `json:"DID"`
			Name   string `json:"Name"`
			Active bool   `json:"Active"`
			PubB64 string `json:"PubB64"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json: "+err.Error(), 400)
			return
		}
		b, err := base64.RawURLEncoding.DecodeString(in.PubB64)
		if err != nil || len(b) != ed25519.PublicKeySize {
			http.Error(w, "bad pub", 400)
			return
		}
		resolver.add(&sagedid.AgentMetadata{
			DID:       sagedid.AgentDID(in.DID),
			Name:      in.Name,
			IsActive:  in.Active,
			PublicKey: ed25519.PublicKey(b),
		})
		w.WriteHeader(204)
	})
	mux.HandleFunc("/debug/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("gRPC : %s", grpcAddr)
	log.Printf("HTTP : %s", httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, mux))
}

/* -------------------- memory resolver -------------------- */

type memResolver struct {
	mu    sync.RWMutex
	store map[string]*sagedid.AgentMetadata
}

func newMemResolver() *memResolver { return &memResolver{store: map[string]*sagedid.AgentMetadata{}} }
func (m *memResolver) add(meta *sagedid.AgentMetadata) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[string(meta.DID)] = meta
}
func (m *memResolver) Resolve(ctx context.Context, did sagedid.AgentDID) (*sagedid.AgentMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	meta := m.store[string(did)]
	if meta == nil {
		return nil, fmt.Errorf("not found")
	}
	return meta, nil
}
func (m *memResolver) ResolvePublicKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	meta, err := m.Resolve(ctx, did)
	if err != nil {
		return nil, err
	}
	return meta.PublicKey, nil
}
func (m *memResolver) ResolveKEMKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	meta, err := m.Resolve(ctx, did)
	if err != nil {
		return nil, err
	}
	return meta.PublicKEMKey, nil
}
func (m *memResolver) VerifyMetadata(context.Context, sagedid.AgentDID, *sagedid.AgentMetadata) (*sagedid.VerificationResult, error) {
	return &sagedid.VerificationResult{Valid: true, VerifiedAt: time.Now()}, nil
}
func (m *memResolver) ListAgentsByOwner(context.Context, string) ([]*sagedid.AgentMetadata, error) {
	return nil, nil
}
func (m *memResolver) Search(context.Context, sagedid.SearchCriteria) ([]*sagedid.AgentMetadata, error) {
	return nil, nil
}

/* ------------------------- utils ------------------------- */
func mustReadAll(rc io.ReadCloser) []byte { b, _ := io.ReadAll(rc); _ = rc.Close(); return b }
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
