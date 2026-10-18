package hpke

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"time"
	"unicode/utf8"
)

// HTTPApplicationOwner010 exclusively owns an unused HTTP session for a trusted
// application adapter. It is not a dispatch grant or a TLS/network transport.
// Old session aliases cannot send, receive, rebind or erase this owner's keys.
type HTTPApplicationOwner010 struct{ session *AuthenticatedCompletion010 }

// HTTPApplicationRequest010 owns accepted content and authenticated envelope
// fields. Its zero value is not an authentication or execution capability.
type HTTPApplicationRequest010 struct {
	payload                                   []byte
	metadata                                  map[string]string
	did, recipient, kid, id, context, session string
}

// Payload returns an owned copy of accepted application bytes.
func (v *HTTPApplicationRequest010) Payload() []byte { return append([]byte(nil), v.payload...) }

// Metadata returns owned data and preserves absent versus empty metadata.
func (v *HTTPApplicationRequest010) Metadata() map[string]string {
	if v.metadata == nil {
		return nil
	}
	result := make(map[string]string, len(v.metadata))
	for k, value := range v.metadata {
		result[k] = value
	}
	return result
}

// Sender returns the authenticated envelope sender.
func (v *HTTPApplicationRequest010) Sender() string { return v.did }

// Recipient returns the authenticated envelope recipient.
func (v *HTTPApplicationRequest010) Recipient() string { return v.recipient }

// KeyID returns the exact authenticated signing key URL.
func (v *HTTPApplicationRequest010) KeyID() string { return v.kid }

// ID returns this authenticated envelope identifier.
func (v *HTTPApplicationRequest010) ID() string { return v.id }

// ContextID returns the authenticated session context.
func (v *HTTPApplicationRequest010) ContextID() string { return v.context }

// SessionID returns the authenticated session identifier.
func (v *HTTPApplicationRequest010) SessionID() string { return v.session }

// HTTPApplicationResponse010 owns one accepted, exactly correlated response.
// Payload/Data and metadata are data; success does not authorize local effects.
type HTTPApplicationResponse010 struct {
	HTTPApplicationRequest010
	inReplyTo, code string
	success         bool
}

// Data returns an owned copy of accepted response bytes.
func (v *HTTPApplicationResponse010) Data() []byte { return v.Payload() }

// InReplyTo returns the exactly correlated request identifier.
func (v *HTTPApplicationResponse010) InReplyTo() string { return v.inReplyTo }

// Success reports the authenticated transport result, not permission for effects.
func (v *HTTPApplicationResponse010) Success() bool { return v.success }

// Error returns the authenticated transport error code, or an empty string.
func (v *HTTPApplicationResponse010) Error() string { return v.code }

// TakeHTTPApplication atomically transfers an unused session and fixes its exact
// trusted target. An existing HTTP binding must match. Failure does not transfer
// ownership. HTTP handshakes and provisional responder receive remain supported.
func (s *AuthenticatedCompletion010) TakeHTTPApplication(target string) (*HTTPApplicationOwner010, error) {
	if s == nil || s.endpoint == nil {
		return nil, errCompletion010
	}
	s.endpoint.mu.Lock()
	defer s.endpoint.mu.Unlock()
	authority, err := httpEndpoint010(target)
	if err != nil || s.application || s.unavailable010() || s.closed || s.endpoint.retired.Load() || (s.lifetime.http.Load() && s.httpTarget == "") || s.lifetime.used.Load() || len(s.sent)+len(s.received) != 0 || (s.httpTarget != "" && (s.httpTarget != target || s.httpAuthority != authority)) {
		return nil, errCompletion010
	}
	moved := *s
	moved.lifetime = newRecordLifetime010(s.created)
	moved.lifetime.confirmed.Store(s.confirmed)
	moved.lifetime.active.Store(s.lifetime.active.Load())
	moved.lifetime.wall.Store(s.lifetime.wall.Load())
	moved.lifetime.checked.Store(s.lifetime.checked.Load())
	moved.lifetime.http.Store(true)
	moved.httpTarget, moved.httpAuthority = target, authority
	moved.application = true
	s.lifetime.disabled.Store(true)
	return &HTTPApplicationOwner010{session: &moved}, nil
}
func (o *HTTPApplicationOwner010) usable(ctx context.Context) bool {
	return o != nil && o.session != nil && o.session.application && ctx != nil && ctx.Err() == nil
}

// Check revalidates the session against its configured current authority.
func (o *HTTPApplicationOwner010) Check(ctx context.Context) error {
	if !o.usable(ctx) {
		return errCompletion010
	}
	return o.session.Check(ctx)
}

// LocalNow is a bounded local clock sample, not current registry authority.
func (o *HTTPApplicationOwner010) LocalNow() (time.Duration, error) {
	if o == nil || o.session == nil || !o.session.application {
		return 0, errCompletion010
	}
	return (&NonHTTPOwner010{session: o.session}).LocalNow()
}

// Observe revalidates current registry bindings and returns the local operation
// start for diagnostics/budget checks, never a portable dispatch permit.
func (o *HTTPApplicationOwner010) Observe(ctx context.Context) (time.Duration, error) {
	if !o.usable(ctx) {
		return 0, errCompletion010
	}
	return (&NonHTTPOwner010{session: o.session}).Observe(ctx)
}

// Close retires this owner and every copy of it.
func (o *HTTPApplicationOwner010) Close() {
	if o != nil && o.session != nil && o.session.application {
		o.session.Close()
	}
}

// Participants returns fixed local and peer identities, not current authority.
func (o *HTTPApplicationOwner010) Participants() (string, string, error) {
	if o == nil || o.session == nil || !o.session.application {
		return "", "", errCompletion010
	}
	s := o.session
	s.endpoint.mu.Lock()
	defer s.endpoint.mu.Unlock()
	if s.closed || s.unavailable010() || s.endpoint.retired.Load() {
		return "", "", errCompletion010
	}
	if s.initiator {
		return s.tuple["initDid"], s.tuple["respDid"], nil
	}
	return s.tuple["respDid"], s.tuple["initDid"], nil
}

// SealRequest clones metadata and content before any provider call. Metadata is
// included in both the envelope signature and AEAD AAD. The existing total AAD
// cap of 4033 bytes can be stricter than the metadata-only JCS cap of 4096.
func (o *HTTPApplicationOwner010) SealRequest(ctx context.Context, data []byte, metadata map[string]string, ttl int64) (HTTPMessage010, error) {
	if !o.usable(ctx) || len(data) > 16348 {
		return HTTPMessage010{}, errCompletion010
	}
	owned, err := copyApplicationMetadata010(metadata)
	if err != nil {
		return HTTPMessage010{}, err
	}
	data = append([]byte(nil), data...)
	s := o.session
	s.endpoint.mu.Lock()
	defer s.endpoint.mu.Unlock()
	return s.sealHTTPRequestMetadata010(ctx, data, owned, ttl)
}

// OpenRequest releases no view before complete HTTP/envelope/AEAD/replay
// acceptance. Transport facts must come from authenticated TLS, not JSON hints.
func (o *HTTPApplicationOwner010) OpenRequest(ctx context.Context, message HTTPMessage010) (*HTTPApplicationRequest010, error) {
	if !o.usable(ctx) || len(message.Body) > 32768 || len(message.Headers) > 256 {
		return nil, errCompletion010
	}
	message.Body = append([]byte(nil), message.Body...)
	message.Headers = append([][2]string(nil), message.Headers...)
	s := o.session
	s.endpoint.mu.Lock()
	defer s.endpoint.mu.Unlock()
	payload, err := s.openHTTPRequest010(ctx, message)
	if err != nil {
		return nil, err
	}
	return applicationView010(payload, message.Body)
}

// SealResponse authenticates independent metadata and the retained request hash.
func (o *HTTPApplicationOwner010) SealResponse(ctx context.Context, messageID string, data []byte, metadata map[string]string, success bool, code string, ttl int64, status int) (HTTPMessage010, error) {
	if !o.usable(ctx) || len(data) > 16348 {
		return HTTPMessage010{}, errCompletion010
	}
	owned, err := copyApplicationMetadata010(metadata)
	if err != nil {
		return HTTPMessage010{}, err
	}
	data = append([]byte(nil), data...)
	s := o.session
	s.endpoint.mu.Lock()
	defer s.endpoint.mu.Unlock()
	return s.sealHTTPResponseMetadata010(ctx, messageID, data, owned, success, code, ttl, status)
}

// OpenResponse returns owned data only for one accepted correlated response.
func (o *HTTPApplicationOwner010) OpenResponse(ctx context.Context, message HTTPMessage010) (*HTTPApplicationResponse010, error) {
	if !o.usable(ctx) || len(message.Body) > 32768 || len(message.Headers) > 256 {
		return nil, errCompletion010
	}
	message.Body = append([]byte(nil), message.Body...)
	message.Headers = append([][2]string(nil), message.Headers...)
	s := o.session
	s.endpoint.mu.Lock()
	defer s.endpoint.mu.Unlock()
	response, err := s.openHTTPResponse010(ctx, message)
	if err != nil {
		return nil, err
	}
	view, err := applicationView010(response.Data, message.Body)
	if err != nil {
		return nil, err
	}
	return &HTTPApplicationResponse010{HTTPApplicationRequest010: *view, inReplyTo: response.MessageID, code: response.Error, success: response.Success}, nil
}

// Only the accepted, privately owned envelope from the receive methods above
// may reach this helper. It runs under endpoint serialization after acceptance;
// no caller-supplied authentication flag or external post-parse is involved.
func applicationView010(payload, accepted []byte) (*HTTPApplicationRequest010, error) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(accepted, &fields) != nil || str010(fields, "encoding") != "session" || str010(fields, "id") == "" || str010(fields, "did") == "" || str010(fields, "recipient") == "" || str010(fields, "kid") == "" || str010(fields, "context_id") == "" || str010(fields, "session_id") == "" {
		return nil, errCompletion010
	}
	metadata, err := readApplicationMetadata010(fields["metadata"])
	if err != nil {
		return nil, err
	}
	return &HTTPApplicationRequest010{payload: append([]byte(nil), payload...), metadata: metadata,
		did: str010(fields, "did"), recipient: str010(fields, "recipient"), kid: str010(fields, "kid"), id: str010(fields, "id"), context: str010(fields, "context_id"), session: str010(fields, "session_id")}, nil
}
func applicationMetadataName010(name string) bool {
	if len(name) < 1 || len(name) > 32 || name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for _, c := range []byte(name[1:]) {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}
func copyApplicationMetadata010(metadata map[string]string) (map[string]string, error) {
	if metadata == nil {
		return nil, nil
	}
	if len(metadata) > 32 {
		return nil, errCompletion010
	}
	owned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		if !applicationMetadataName010(key) || !utf8.ValidString(value) || len(value) > 1024 {
			return nil, errCompletion010
		}
		owned[key] = value
	}
	if len(canon010(owned)) > 4096 {
		return nil, errCompletion010
	}
	return owned, nil
}
func readApplicationMetadata010(raw json.RawMessage) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if len(raw) > 32768 || !utf8.Valid(raw) || !validApplicationEscapes010(raw) {
		return nil, errCompletion010
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errCompletion010
	}
	metadata := map[string]string{}
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			return nil, errCompletion010
		}
		key, ok := token.(string)
		if !ok {
			return nil, errCompletion010
		}
		if _, exists := metadata[key]; exists {
			return nil, errCompletion010
		}
		token, err = decoder.Token()
		value, ok := token.(string)
		if err != nil || !ok || len(metadata) >= 32 {
			return nil, errCompletion010
		}
		metadata[key] = value
	}
	if token, err = decoder.Token(); err != nil || token != json.Delim('}') {
		return nil, errCompletion010
	}
	if _, err = decoder.Token(); err != io.EOF {
		return nil, errCompletion010
	}
	return copyApplicationMetadata010(metadata)
}

// encoding/json replaces lone UTF-16 surrogates. Reject those escapes before
// decoding, as the existing Guard and Registry strict-input boundaries do.
func validApplicationEscapes010(raw []byte) bool {
	inside := false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '"' {
			inside = !inside
			continue
		}
		if !inside || raw[i] != '\\' {
			continue
		}
		i++
		if i >= len(raw) {
			return false
		}
		if raw[i] != 'u' {
			continue
		}
		if i+4 >= len(raw) {
			return false
		}
		first, err := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		i += 4
		if first >= 0xdc00 && first <= 0xdfff {
			return false
		}
		if first < 0xd800 || first > 0xdbff {
			continue
		}
		if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
			return false
		}
		second, err := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
		if err != nil || second < 0xdc00 || second > 0xdfff {
			return false
		}
		i += 6
	}
	return true
}
