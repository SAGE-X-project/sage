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
	"bytes"
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

	scenarioHeader = "X-Test-Scenario"
)

var seq uint64

/* ---------------- logging helpers ---------------- */

func packetPrefix(id uint64, scenario string) string {
	if scenario == "" {
		return fmt.Sprintf("[req#%03d]", id)
	}
	return fmt.Sprintf("[packet %s|req#%03d]", scenario, id)
}
func logPacket(prefix, direction, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("%s %s %s", prefix, direction, msg)
}
func respondError(w http.ResponseWriter, prefix string, status int, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	logPacket(prefix, "server ->", "error %d: %s", status, msg)
	http.Error(w, msg, status)
}
func banner(prefix, title string) {
	log.Printf("\n==================== %s %s ====================\n", prefix, title)
}
func dumpHeaders(prefix string, r *http.Request) {
	dump, _ := httputil.DumpRequest(r, false)
	log.Printf("%s HEADERS:\n%s", prefix, dump)
}
func dumpCipherRaw(prefix string, b []byte) {
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
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

/* ---------------- memory resolver ---------------- */

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
	return meta.PublicKey, nil // Ed25519 key used to verify client metadata signatures
}
func (m *memResolver) ResolveKEMKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	meta, err := m.Resolve(ctx, did)
	if err != nil {
		return nil, err
	}
	return meta.PublicKEMKey, nil // Provide the server KEM public key
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

/* ---------------- main ---------------- */

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Server signing key (Ed25519) and recipient KEM (X25519)
	serverSignKP, _ := keys.GenerateEd25519KeyPair()
	serverKEMKP, _ := keys.GenerateX25519KeyPair()
	serverDID := string(sagedid.GenerateDID(sagedid.ChainEthereum, "server"))

	// Session manager with idle timeout 2s to exercise expiration
	sessMgr := session.NewManager()
	sessMgr.SetDefaultConfig(session.Config{
		MaxAge:      time.Hour,
		IdleTimeout: 2 * time.Second,
		MaxMessages: 1000,
	})

	// DID resolver containing the server itself plus registered clients
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

	grpcConn, _ := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	a2aCli := a2apb.NewA2AServiceClient(grpcConn)

	// HTTP server
	mux := http.NewServeMux()

	// A2A JSON → gRPC proxy
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
		scLabel := r.Header.Get(scenarioHeader)
		pfx := packetPrefix(id, scLabel)
		banner(pfx, "INCOMING /protected")

		// Capture ciphertext body
		cipherBody, err := io.ReadAll(r.Body)
		if err != nil {
			respondError(w, pfx, 400, "read body: %v", err)
			return
		}
		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(cipherBody)) // keep signature library compatibility

		logPacket(pfx, "server <-", "received %d bytes", len(cipherBody))
		dumpHeaders(pfx, r)
		dumpCipherRaw(pfx, cipherBody)
		dumpCipherB64(pfx, cipherBody)
		dumpCipherHexPreview(pfx, cipherBody, 64)

		// Signature parameters
		inp, err := rfc9421.ParseSignatureInput(r.Header.Get("Signature-Input"))
		if err != nil {
			respondError(w, pfx, 400, "invalid Signature-Input")
			return
		}
		p := inp["sig1"]
		if p == nil || p.KeyID == "" {
			respondError(w, pfx, 400, "missing key id")
			return
		}
		if p.Nonce == "" {
			respondError(w, pfx, 400, "missing nonce")
			return
		}

		// Replay protection
		if sessMgr.ReplayGuardSeenOnce(p.KeyID, p.Nonce) {
			respondError(w, pfx, 401, "replay")
			return
		}

		// Look up session by key ID
		sess, ok := sessMgr.GetByKeyID(p.KeyID)
		if !ok {
			respondError(w, pfx, 401, "no session")
			return
		}

		// Resolve client Ed25519 public key and verify signature
		cliDID := r.Header.Get("X-Agent-DID")
		if cliDID == "" {
			respondError(w, pfx, 400, "missing X-Agent-DID")
			return
		}
		meta, err := resolver.Resolve(r.Context(), sagedid.AgentDID(cliDID))
		if err != nil {
			respondError(w, pfx, 401, "unknown agent")
			return
		}
		pub, ok := meta.PublicKey.(ed25519.PublicKey)
		if !ok {
			respondError(w, pfx, 500, "invalid agent key")
			return
		}

		ver := rfc9421.NewHTTPVerifier()
		if err := ver.VerifyRequest(r, pub, &rfc9421.HTTPVerificationOptions{
			SignatureName: "sig1", MaxAge: 2 * time.Minute,
		}); err != nil {
			respondError(w, pfx, 401, "signature verify failed: %v", err)
			return
		}

		// Decrypt payload
		plain, err := sess.Decrypt(cipherBody)
		if err != nil {
			respondError(w, pfx, 401, "decrypt: %v", err)
			return
		}
		dumpPlainPretty(pfx, plain)

		// Encrypt response from server to client
		reply := []byte(fmt.Sprintf(`{"from":"server","ok":true,"echo":%s}`, plain))
		ct, _ := sess.Encrypt(reply)
		resp := map[string]string{"cipher_b64": base64.StdEncoding.EncodeToString(ct)}

		banner(pfx, "OUTGOING RESPONSE")
		logPacket(pfx, "server ->", "status %d", http.StatusOK)
		log.Printf("%s BODY(JSON): %v", pfx, resp)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Debug endpoints
	mux.HandleFunc("/debug/kem-pub", func(w http.ResponseWriter, r *http.Request) {
		pub := serverKEMKP.PublicKey().(*ecdh.PublicKey).Bytes()
		_ = json.NewEncoder(w).Encode(map[string]string{"kem": base64.RawURLEncoding.EncodeToString(pub)})
	})
	mux.HandleFunc("/debug/sign-pub", func(w http.ResponseWriter, r *http.Request) {
		pub := serverSignKP.PublicKey().(ed25519.PublicKey)
		_ = json.NewEncoder(w).Encode(map[string]string{"sign": base64.RawURLEncoding.EncodeToString(pub)})
	})
	mux.HandleFunc("/debug/server-did", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"did": serverDID})
	})
	mux.HandleFunc("/debug/register-agent", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			DID    string `json:"DID"`
			Name   string `json:"Name"`
			Active bool   `json:"Active"`
			PubB64 string `json:"PubB64"` // Ed25519
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

/* ---------------- utils ---------------- */
func mustReadAll(rc io.ReadCloser) []byte { b, _ := io.ReadAll(rc); _ = rc.Close(); return b }
