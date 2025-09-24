package handshake

import (
	"context"
	"crypto"
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	a2a "github.com/a2aproject/a2a/grpc"
	"github.com/google/uuid"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/handshake/session"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// pendingState holds only public transcript material with an expiry.
// No shared secret is stored here (Core derives it later at OnComplete).
type pendingState struct {
	peerEph   []byte    // raw client ephemeral public (32B)
	serverEph []byte    // raw server ephemeral public (32B) provided by Core
	expires   time.Time // TTL for cleanup if Complete never arrives
}


// Server is A2AServiceServer implementation
//  - Does not create/store sessions.
//  - Emits Events so the application layer can manage sessions separately.
//  - Can send Response to the peer via outbound client if configured.
type Server struct {
	a2a.UnimplementedA2AServiceServer // MUST be embedded by value

	key      sagecrypto.KeyPair
	events   Events
	// Outbound lets the server proactively send messages to peer (e.g., Response after Request).
	outbound a2a.A2AServiceClient

	// Resolve the sender pubkey from ctx/message/metadata for signature check or decrypt.
	// Pasrse JWT and get DID field
	resolver did.Resolver
	mu      sync.Mutex
	// pending holds per-context ephemeral handshake state created at Request phase.
	pending map[string] pendingState

	// sessionCfg defines default session policies for SecureSession instances.
	sessionCfg session.Config

	// TTL and cleaner
	pendingTTL    time.Duration
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}

	exporter sagecrypto.KeyExporter
	importer sagecrypto.KeyImporter
}

// NewServer creates a server with required dependencies.
// - events: application-level hooks (can be NoopEvents{})
// - outbound: optional, used when this server should call the peer proactively (B->A Response).
func NewServer(
	key sagecrypto.KeyPair,
	events Events,
	outbound a2a.A2AServiceClient,
	resolver did.Resolver,
	sessionCfg *session.Config,
) *Server {
	if events == nil {
		events = NoopEvents{}
	}

	// Apply defaults if sessionCfg is nil
	var cfg session.Config
	if sessionCfg == nil {
		cfg = session.Config{
			MaxAge:      time.Hour,        // default absolute expiration
			IdleTimeout: 10 * time.Minute, // default idle timeout
			MaxMessages: 10000,            // default max message count
		}
	} else {
		cfg = *sessionCfg
	}
	s :=  &Server{
		key:                 key,
		events:              events,
		outbound:            outbound,
		resolver: 			 resolver,
		pending:			 make(map[string]pendingState),
		sessionCfg:          cfg,
		exporter:            formats.NewJWKExporter(),
		importer:            formats.NewJWKImporter(),
	}
	s.cleanupTicker = time.NewTicker(10 * time.Minute)
	go s.cleanupLoop()
	return s
}

// SetOutbound allows wiring the peer client after server startup.
func (s *Server) SetOutbound(c a2a.A2AServiceClient) { s.outbound = c }

// SendMessage is the single entry point for all phases.
// It validates input, decodes payload, and triggers event callbacks.
// If outbound is configured, the server may proactively send a Response after Request.
func (s *Server) SendMessage(ctx context.Context, in *a2a.SendMessageRequest) (*a2a.SendMessageResponse, error) {
	if in == nil || in.Request == nil {
		return nil, errors.New("empty request")
	}
	msg := in.Request

	phase, err := parsePhase(msg.TaskId)
	if err != nil {
		return nil, err
	}
	payload, err := firstDataPart(msg)
	if err != nil {
		return nil, err
	}

	// Resolve sender public key when needed
	var senderPub crypto.PublicKey
	var senderDID string
	if s.resolver != nil {
		v := in.Metadata.GetFields()["did"]
		if v == nil || v.GetStringValue() == "" {
			return nil, errors.New("missing did")
		}
		senderDID = v.GetStringValue()
		senderPub, _ = s.resolver.ResolvePublicKey(ctx, did.AgentDID(senderDID))
		if senderPub == nil {
			return nil, errors.New("cannot resolve sender pubkey")
		}
	}

	// Verify sender signature if metadata is present.
	if in.Metadata != nil {
		if err := s.verifySenderSignature(msg, in.Metadata, senderPub); err != nil {
			return nil, fmt.Errorf("signature verification failed: %w", err)
		}
	}

	switch phase {

	case Invitation:
		var inv InvitationMessage
		if err := fromStructPB(payload, &inv); err != nil {
			return nil, fmt.Errorf("invitation decode: %w", err)
		}
		_ = s.events.OnInvitation(ctx, msg.ContextId, inv)
		return s.ack(msg, "invitation_received")

	case Request:
		plain, err := s.decryptPacket(payload)
		if err != nil {
			return nil, fmt.Errorf("request decrypt: %w", err)
		}
		var req RequestMessage
		if err := json.Unmarshal(plain, &req); err != nil {
			return nil, fmt.Errorf("request json: %w", err)
		}
		
		if len(req.EphemeralPubKey) == 0 {
			return nil, fmt.Errorf("empty peer ephemeral: %w", err)
		}

		exported, err := s.importer.ImportPublic([]byte(req.EphemeralPubKey), sagecrypto.KeyFormatJWK)
		peerPub, ok := exported.(*ecdh.PublicKey)
		if !ok { 
			return nil, fmt.Errorf("unexpected peer eph key type: %T", peerPub) 
		}
		peerEphRaw := peerPub.Bytes()
		if len(peerEphRaw) != 32 {
			return nil, fmt.Errorf("invalid peer eph length: %d", len(peerEphRaw))
		}

		serverEphRaw, serverEphJWK, err := s.events.AskEphemeral(ctx, msg.ContextId)
		if err != nil { 
			return nil, fmt.Errorf("ask ephemeral: %w", err) 
		}
		if len(serverEphRaw) != 32 {
			return nil, fmt.Errorf("invalid server eph length: %d", len(serverEphRaw))
		}

		s.savePending(msg.ContextId, pendingState{
			peerEph:   append([]byte(nil), peerEphRaw...),
			serverEph: append([]byte(nil), serverEphRaw...),
			// expires: time.Now().Add(time.Hour), 
		})
		_ = s.events.OnRequest(ctx, msg.ContextId, req, senderPub)

		// Optionally respond immediately to the peer.
		if s.outbound != nil {
			res := ResponseMessage{
				EphemeralPubKey: json.RawMessage(serverEphJWK),  
				Ack:       true,
			}
			
			if _, err := s.sendResponseToPeer(ctx, res, msg.ContextId, senderPub, senderDID); err != nil {
				return nil, fmt.Errorf("send response to peer: %w", err)
			}
		}
		return s.ack(msg, "request_processed")

	case Response:
		var res ResponseMessage
		// Prefer encrypted envelope; fall back to clear JSON if not present.
		if b64, err := structPBToB64(payload); err == nil {
			plain, err := s.decryptPacket(payload)
			if err != nil {
				return nil, fmt.Errorf("response decrypt: %w", err)
			}
			if err := json.Unmarshal(plain, &res); err != nil {
				return nil, fmt.Errorf("response json: %w", err)
			}
			_ = b64 // silence if unused in your linter
		} else {
			if err := fromStructPB(payload, &res); err != nil {
				return nil, fmt.Errorf("response decode: %w", err)
			}
		}
		_ = s.events.OnResponse(ctx, msg.ContextId, res, senderPub)
		return s.ack(msg, "response_received")

	case Complete:
		var comp CompleteMessage
		_ = fromStructPB(payload, &comp) // best-effort

		st, ok := s.takePending(msg.ContextId)
		if !ok {
			_ = s.events.OnComplete(ctx, msg.ContextId, comp, session.Params{})
			return s.ack(msg, "complete_received_no_pending")
		}

		sessParams := session.Params {
				ContextID: msg.ContextId,
				SelfEph: st.serverEph,
				PeerEph: st.peerEph,
				Label:  "a2a/handshake v1",
		}

		_ = s.events.OnComplete(ctx, msg.ContextId, comp, sessParams)

		if binder, ok := any(s.events).(KeyIDBinder); ok && s.outbound != nil && senderPub != nil {
			if kid, ok2 := binder.IssueKeyID(msg.ContextId); ok2 && kid != "" {
				res := ResponseMessage{
					Ack:   true,
					KeyID: kid,
				}
				if _, err := s.sendResponseToPeer(ctx, res, msg.ContextId, senderPub, senderDID); err != nil {
					return nil, fmt.Errorf("send response to peer: %w", err)
				}
			}
		}
		return s.ack(msg, "complete_received_session_ready")

	default:
		return nil, errors.New("unknown phase")
	}
}

// firstDataPart extracts the first DataPart struct payload.
func firstDataPart(m *a2a.Message) (*structpb.Struct, error) {
	if m == nil || len(m.Content) == 0 {
		return nil, errors.New("empty content")
	}
	dpart, ok := m.Content[0].GetPart().(*a2a.Part_Data)
	if !ok || dpart.Data == nil || dpart.Data.Data == nil {
		return nil, errors.New("missing data part")
	}
	return dpart.Data.Data, nil
}

// decryptPacket decodes base64url and performs bootstrap decrypt with peer key.
func (s *Server) decryptPacket(st *structpb.Struct) ([]byte, error) {
	b64, err := structPBToB64(st)
	if err != nil {
		return nil, err
	}
	packet, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("b64 decode: %w", err)
	}
	plain, err := keys.DecryptWithEd25519Peer(s.key.PrivateKey(), packet)
	if err != nil {
		return nil, err
	}
	return plain, nil
}

// sendResponseToPeer builds and sends a Response to the peer over gRPC using outbound client.
// It encrypts the Response with the peer's public key (bootstrap envelope).
func (s *Server) sendResponseToPeer(ctx context.Context, res ResponseMessage, ctxID string, peerPub crypto.PublicKey, senderDID string) (*a2a.SendMessageResponse, error) {
	if s.outbound == nil {
		return nil, errors.New("no outbound client configured")
	}
	plain, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}
	packet, err := keys.EncryptWithEd25519Peer(peerPub, plain)
	if err != nil {
		return nil, fmt.Errorf("encrypt response: %w", err)
	}
	payload, _ := b64ToStructPB(base64.RawURLEncoding.EncodeToString(packet))
	msg := &a2a.Message{
		MessageId: uuid.NewString(),
		ContextId: ctxID,
		TaskId:    GenerateTaskID(Response),
		Role:      a2a.Role_ROLE_AGENT,
		Content:   []*a2a.Part{{Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: payload}}}},
	}
	return s.sendSigned(ctx, msg, senderDID)
}

// sendSigned marshals deterministically, signs with server key, and invokes outbound.SendMessage.
func (s *Server) sendSigned(ctx context.Context, msg *a2a.Message, did string) (*a2a.SendMessageResponse, error) {
	bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal for signing: %w", err)
	}
	meta, err := signStruct(s.key, bytes, did)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	return s.outbound.SendMessage(ctx, &a2a.SendMessageRequest{Request: msg, Metadata: meta})
}

// verifySenderSignature checks metadata.signature against deterministic-marshaled message bytes.
func (s *Server) verifySenderSignature(m *a2a.Message, meta *structpb.Struct, senderPub crypto.PublicKey) error {
	field := meta.GetFields()["signature"]
	if field == nil {
		return errors.New("missing signature")
	}
	sig, err := base64.RawURLEncoding.DecodeString(field.GetStringValue())
	if err != nil {
		return fmt.Errorf("bad signature b64: %w", err)
	}
	bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
	if err != nil {
		return err
	}
	// Support either a custom Verify interface or raw ed25519.PublicKey
	type verifyKey interface {
		Verify(msg, sig []byte) error
	}

	switch pk := senderPub.(type) {
	case verifyKey:
		// Your key type implements Verify([]byte, []byte) error
		if err := pk.Verify(bytes, sig); err != nil {
			return fmt.Errorf("signature verify failed: %w", err)
		}
		return nil
	case ed25519.PublicKey:
		// Standard ed25519
		if !ed25519.Verify(pk, bytes, sig) {
			return errors.New("signature verify failed: invalid ed25519 signature")
		}
		return nil
	default:
		return fmt.Errorf("unsupported public key type: %T", senderPub)
	}
}

// ack builds a SendMessageResponse carrying a Task payload.
// It sets Task.Id, Task.ContextId, optional Status and Metadata.
func (s *Server) ack(in *a2a.Message, note string) (*a2a.SendMessageResponse, error) {
	// Metadata payload
	ack := map[string]any{
		"note": note,
		"ts":   time.Now().UTC().Format(time.RFC3339Nano),
		"ctx":  in.ContextId,
		"task": in.TaskId,
	}
	ackPB, _ := toStructPB(ack)

	// Build Task with status
	task := &a2a.Task{
		Id:        in.TaskId,
		ContextId: in.ContextId,
		Metadata:  ackPB,
		Status: &a2a.TaskStatus{
			State:     a2a.TaskState_TASK_STATE_SUBMITTED,
			Update:    &a2a.Message{
				Content: []*a2a.Part{
					{Part: &a2a.Part_Text{Text: note}},
				},
			},
			Timestamp: timestamppb.Now(),
		},
	}

	// Wrap in oneof Payload
	return &a2a.SendMessageResponse{
		Payload: &a2a.SendMessageResponse_Task{
			Task: task,
		},
	}, nil
}


func (s *Server) cleanupLoop() {
	for {
		select {
		case <-s.cleanupTicker.C:
			now := time.Now()
			s.mu.Lock()
			for ctxID, st := range s.pending {
				if now.After(st.expires) {
					delete(s.pending, ctxID)
				}
			}
			s.mu.Unlock()
		case <-s.stopCleanup:
			s.cleanupTicker.Stop()
			return
		}
	}
}

// save/take helpers
func (s *Server) savePending(id string, st pendingState) {
	s.mu.Lock()
	s.pending[id] = st
	s.mu.Unlock()
}
func (s *Server) takePending(id string) (pendingState, bool) {
	s.mu.Lock()
	st, ok := s.pending[id]
	if ok {
		delete(s.pending, id) // delete on normal Complete path
	}
	s.mu.Unlock()
	return st, ok
}