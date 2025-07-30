package handshake

import (
	"context"
	"crypto"
	"encoding/base64"
	"encoding/json"
	"fmt"

	a2a "github.com/a2aproject/a2a/grpc"
	"github.com/google/uuid"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/keys"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Handshaker defines the methods for a full handshake flow.
type Handshaker interface {
    Invitation(ctx context.Context, jwt string, session Session) (*a2a.SendMessageResponse, error)
    Request(ctx context.Context, reqMsg RequestMessage, edPeerPub crypto.PublicKey)  (*a2a.SendMessageResponse, error)
    Response(ctx context.Context, resMsg ResponseMessage, edPeerPub crypto.PublicKey) (*a2a.SendMessageResponse, error)
    Complete(ctx context.Context, compMsg CompleteMessage)  (*a2a.SendMessageResponse, error)
}

// Client wraps the generated A2AServiceClient and holds agent metadata for handshake operations.
type Client struct {
	a2a.A2AServiceClient
    key sagecrypto.KeyPair
}

// Server wraps the generated A2AServiceServer and holds agent metadata for handshake operations.
type Server struct {
	a2a.A2AServiceServer
    key sagecrypto.KeyPair
}

// NewClient creates a new Client using the provided gRPC connection
// and associates it with the given key pair for signing
func NewClient(conn grpc.ClientConnInterface, key sagecrypto.KeyPair) *Client {
    return &Client{
        A2AServiceClient: a2a.NewA2AServiceClient(conn),
        key: key,
    }
}

// NewServer creates a new Server using the provided A2AServiceServer implementation
// and associates it with the given key pair for signing
func NewServer(serviceImpl a2a.A2AServiceServer, key sagecrypto.KeyPair) *Server {
    return &Server{
        A2AServiceServer: serviceImpl,
        key: key,
    }
}

// Invitation sends the initial handshake invitation message.
// InvitationMessage into a protobuf Struct, wraps it in Part_Data, and sends it.
func (c *Client) Invitation(ctx context.Context, invMsg InvitationMessage) (*a2a.SendMessageResponse, error) {
    payload, err := toStructPB(invMsg)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal session: %w", err)
    }

	msg := &a2a.Message{
        MessageId: uuid.NewString(),
        ContextId: invMsg.ContextID,
        TaskId:    generateTaskID(Invitation),
        Role:      a2a.Role_ROLE_USER,
        Content: []*a2a.Part{
            {Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: payload}}},
        },
    }
    msgBytes, err := proto.MarshalOptions{Deterministic: true,}.Marshal(msg)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal message for signing: %w", err)
    }

    sigMetadata, err := signedStruct(c.key, msgBytes)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal session: %w", err)
    }

    return c.SendMessage(ctx, &a2a.SendMessageRequest{
        Request: msg,
        Metadata: sigMetadata,
    })

}

// Request presents the session key and the RFC‑9421 parameters.
func (c *Client) Request(ctx context.Context, reqMsg RequestMessage, edPeerPub crypto.PublicKey) (*a2a.SendMessageResponse, error) {
    reqBytes, err := json.Marshal(reqMsg)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal: %w", err)
    }

    packet, err := keys.EncryptWithEd25519Peer(edPeerPub, reqBytes)
    if err != nil {
        return nil, fmt.Errorf("failed to encrypt: %w", err)
    }
    packetB64 := base64.RawURLEncoding.EncodeToString(packet)
    payload, err := b64ToStructPB(packetB64)
    if err != nil {
        return nil, err
    }

	msg := &a2a.Message{
        MessageId: uuid.NewString(),
        ContextId: reqMsg.ContextID,
        TaskId:    generateTaskID(Request),
        Role:      a2a.Role_ROLE_USER,
        Content: []*a2a.Part{
            {Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: payload}}},
        },
    }

    msgBytes, err := proto.MarshalOptions{Deterministic: true,}.Marshal(msg)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal message for signing: %w", err)
    }
    sigMetadata, err := signedStruct(c.key, msgBytes)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal session: %w", err)
    }

	return c.SendMessage(ctx, &a2a.SendMessageRequest{
        Request: msg,
        Metadata: sigMetadata,
    })
}

// Response finalizes the session information received from the client’s Request 
// and establishes the parameters in accordance with the RFC‑9421 specification.
func (s *Server) Response(ctx context.Context, resMsg ResponseMessage, edPeerPub crypto.PublicKey) (*a2a.SendMessageResponse, error) {
    resBytes, err := json.Marshal(resMsg)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal: %w", err)
    }

    packet, err := keys.EncryptWithEd25519Peer(edPeerPub, resBytes)
    if err != nil {
        return nil, fmt.Errorf("failed to encrypt: %w", err)
    }

    packetB64 := base64.RawURLEncoding.EncodeToString(packet)
    payload, err := b64ToStructPB(packetB64)
    if err != nil {
        return nil, err
    }
     
    msg := &a2a.Message{
        MessageId: uuid.NewString(),
        ContextId: resMsg.ContextID,
        TaskId:    generateTaskID(Response),
        Role:      a2a.Role_ROLE_AGENT,
        Content: []*a2a.Part{
            {Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: payload}}},
        },
    }

    msgBytes, err := proto.MarshalOptions{Deterministic: true,}.Marshal(msg)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal message for signing: %w", err)
    }
    sigMetadata, err := signedStruct(s.key, msgBytes)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal session: %w", err)
    }

    return s.SendMessage(ctx, &a2a.SendMessageRequest{
        Request: msg,
        Metadata: sigMetadata,
    })

}

// Complete sends a completion notification message without a payload.
// It constructs an A2A message indicating the operation is finished.
func (c *Client) Complete(ctx context.Context, compMsg CompleteMessage) (*a2a.SendMessageResponse, error) {
    payload, err := toStructPB(compMsg)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal session: %w", err)
    }

	msg := &a2a.Message{
        MessageId: uuid.NewString(),
        ContextId: compMsg.ContextID,
        TaskId:    generateTaskID(Complete),
        Role:      a2a.Role_ROLE_USER,
        Content: []*a2a.Part{
            {Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: payload}}},
        },
    }

    msgBytes, err := proto.MarshalOptions{Deterministic: true,}.Marshal(msg)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal message for signing: %w", err)
    }
    sigMetadata, err := signedStruct(c.key, msgBytes)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal session: %w", err)
    }

    return c.SendMessage(ctx, &a2a.SendMessageRequest{
        Request: msg,
        Metadata: sigMetadata,
    })
}

func signedStruct(k sagecrypto.KeyPair, msg []byte) (*structpb.Struct, error) {
    signature, err := k.Sign(msg)
    if err != nil {
        return nil, err
    }

    sigB64 := base64.RawURLEncoding.EncodeToString(signature)
    m := map[string]interface{}{
        "signature": sigB64,
    }
    return structpb.NewStruct(m)
}