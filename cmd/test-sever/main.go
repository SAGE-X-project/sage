package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/handshake"
	sessioninit "github.com/sage-x-project/sage/internal"
	"github.com/sage-x-project/sage/session"
	"google.golang.org/protobuf/encoding/protojson"
)

// ----- 서버가 peer(클라) 쪽으로 푸시할 때 쓰는 JSON-RPC A2A 클라 -----
type jsonrpcA2AClient struct{ url string }

func (c *jsonrpcA2AClient) SendMessage(ctx context.Context, in *a2a.SendMessageRequest, _ ...grpc.CallOption) (*a2a.SendMessageResponse, error) {
	params, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(in)
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "SendMessage",
		"params":  json.RawMessage(params),
		"id":      1,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil { return nil, err }
	defer res.Body.Close()

	var out struct {
		Result json.RawMessage `json:"result"`
		Error  *struct{ Code int; Message string } `json:"error"`
		ID     any `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("jsonrpc decode: %w", err)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("jsonrpc error: %d %s", out.Error.Code, out.Error.Message)
	}
	var resp a2a.SendMessageResponse
	if err := json.Unmarshal(out.Result, &resp); err != nil {
		return nil, fmt.Errorf("result decode: %w", err)
	}
	return &resp, nil
}

// 인터페이스 채우기(서버 outbound에서는 안 씀)
func (c *jsonrpcA2AClient) SendStreamingMessage(context.Context, *a2a.SendMessageRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[a2a.StreamResponse], error) {
	return nil, status.Errorf(codes.Unimplemented, "SendStreamingMessage not implemented")
}
func (c *jsonrpcA2AClient) GetTask(context.Context, *a2a.GetTaskRequest, ...grpc.CallOption) (*a2a.Task, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetTask not implemented")
}
func (c *jsonrpcA2AClient) CancelTask(context.Context, *a2a.CancelTaskRequest, ...grpc.CallOption) (*a2a.Task, error) {
	return nil, status.Errorf(codes.Unimplemented, "CancelTask not implemented")
}
func (c *jsonrpcA2AClient) TaskSubscription(context.Context, *a2a.TaskSubscriptionRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[a2a.StreamResponse], error) {
	return nil, status.Errorf(codes.Unimplemented, "TaskSubscription not implemented")
}
func (c *jsonrpcA2AClient) CreateTaskPushNotificationConfig(context.Context, *a2a.CreateTaskPushNotificationConfigRequest, ...grpc.CallOption) (*a2a.TaskPushNotificationConfig, error) {
	return nil, status.Errorf(codes.Unimplemented, "CreateTaskPushNotificationConfig not implemented")
}
func (c *jsonrpcA2AClient) GetTaskPushNotificationConfig(context.Context, *a2a.GetTaskPushNotificationConfigRequest, ...grpc.CallOption) (*a2a.TaskPushNotificationConfig, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetTaskPushNotificationConfig not implemented")
}
func (c *jsonrpcA2AClient) ListTaskPushNotificationConfig(context.Context, *a2a.ListTaskPushNotificationConfigRequest, ...grpc.CallOption) (*a2a.ListTaskPushNotificationConfigResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "ListTaskPushNotificationConfig not implemented")
}
func (c *jsonrpcA2AClient) GetAgentCard(context.Context, *a2a.GetAgentCardRequest, ...grpc.CallOption) (*a2a.AgentCard, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetAgentCard not implemented")
}
func (c *jsonrpcA2AClient) DeleteTaskPushNotificationConfig(context.Context, *a2a.DeleteTaskPushNotificationConfigRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "DeleteTaskPushNotificationConfig not implemented")
}

func main() {
	serverKeyPair, err := keys.GenerateEd25519KeyPair()
	if err != nil { log.Fatalf("keygen: %v", err) }
	sessMgr := session.NewManager()
	sessMgr.SetDefaultConfig(session.Config{
		MaxAge:      time.Hour,
		IdleTimeout: 10 * time.Minute,
		MaxMessages: 1,
	})
	events := sessioninit.NewCreator(sessMgr)

	reg := map[string]ed25519.PublicKey{}
	resolve := func(ctx context.Context, msg *a2a.Message, meta *structpb.Struct) (crypto.PublicKey, error) {
		if meta == nil {
			return nil, errors.New("missing metadata")
		}
		f := meta.GetFields()["client_pub_b64"]
		if f == nil {
			return nil, errors.New("missing client_pub_b64")
		}
		raw, err := base64.RawURLEncoding.DecodeString(f.GetStringValue())
		if err != nil {
			return nil, fmt.Errorf("bad client_pub_b64: %w", err)
		}
		if len(raw) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("bad ed25519 pubkey length: %d", len(raw))
		}
		return ed25519.PublicKey(raw), nil
	}
	// 3) gRPC 서버 + handshake.Server
	grpcLis, err := net.Listen("tcp", "127.0.0.1:50051")
	if err != nil { log.Fatal(err) }
	grpcSrv := grpc.NewServer()
	srv := handshake.NewServer(serverKeyPair, events, nil, resolve, nil)
	a2a.RegisterA2AServiceServer(grpcSrv, srv)
	go func() {
		log.Println("gRPC :50051")
		if err := grpcSrv.Serve(grpcLis); err != nil { log.Fatalf("grpc serve: %v", err) }
	}()

	conn, err := grpc.NewClient(grpcLis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { log.Fatalf("grpc self-dial: %v", err) }
	grpcCli := a2a.NewA2AServiceClient(conn)

	mux := http.NewServeMux()

	// 수동 브리지: /v1/a2a:sendMessage → gRPC SendMessage
	mux.HandleFunc("/v1/a2a:sendMessage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", 405); return }
		defer r.Body.Close()
    	raw, _ := io.ReadAll(r.Body)

		var req a2a.SendMessageRequest
		if err := protojson.Unmarshal(raw, &req); err != nil {
			http.Error(w, "bad json: "+err.Error(), 400); return
		}
	
		resp, err := grpcCli.SendMessage(r.Context(), &req)
		if err != nil { http.Error(w, "grpc error: "+err.Error(), 500); return }
		out, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	})

	// 2) 보호 API: kid/nonce → 세션 복호화(+만료/리플레이 차단)
	mux.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		dump2, _ := httputil.DumpRequest(r, true)
		log.Printf("HTTP IN:\n%s", dump2)
		ts := time.Now()
		si := r.Header.Get("Signature-Input")
		kid := pick(si, `keyid="`)
		nonce := pick(si, `nonce="`)
		if kid == "" || nonce == "" { http.Error(w, "bad signature-input", 400); return }
		if sessMgr.ReplayGuardSeenOnce(kid, nonce) { http.Error(w, "replay", 401); return }
		
		sess, ok := sessMgr.GetByKeyID(kid)
		if !ok { http.Error(w, "no session", 401); return }
		
		sigB64, ok := parseSig1(r.Header.Get("Signature"))
   		if !ok { http.Error(w, "missing signature", 400); return }
		sig, err := base64.RawURLEncoding.DecodeString(sigB64)
    	if err != nil { http.Error(w, "bad signature", 400); return }

		covered := buildCovered(r, "sig1", kid, nonce)
		if err := sess.VerifyCovered(covered, sig); err != nil {
			http.Error(w, "signature verify failed", 401); return
		}

		cipher, _ := io.ReadAll(r.Body)
    	plain, err := sess.Decrypt(cipher)
		if err != nil { http.Error(w, "decrypt: "+err.Error(), 401); return }
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(plain)
		log.Printf("Decrypt: [/protected] 200 OK kid=%s nonce=%s took=%s body=%s\n\n", kid, nonce, time.Since(ts), string(plain))
	})

	mux.HandleFunc("/debug/server-pub", func(w http.ResponseWriter, r *http.Request) {
		pub := serverKeyPair.PublicKey().(ed25519.PublicKey)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"pub": base64.RawURLEncoding.EncodeToString(pub),
		})
	})

	mux.HandleFunc("/debug/register-did", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ DID, PubB64 string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		b, err := base64.RawURLEncoding.DecodeString(in.PubB64)
		if err != nil || len(b) != ed25519.PublicKeySize {
			http.Error(w, "bad pub", 400); return
		}
		reg[in.DID] = ed25519.PublicKey(b)
		w.WriteHeader(204)
	})

	mux.HandleFunc("/debug/set-outbound", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ URL string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		srv.SetOutbound(&jsonrpcA2AClient{url: in.URL})
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