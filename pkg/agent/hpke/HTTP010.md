# HTTP signatures for authenticated session messages

Bind an unused authenticated completion to a trusted canonical HTTPS endpoint
with `BindHTTP` (Go) or `bind_http` (Rust), then use the HTTP request/response
seal/open methods. Binding is permanent: bare session record and response methods
reject calls. The handshake itself is still carried by the existing plain
completion API; this addition does not wrap that handshake in HTTP.

The transport supplies exact content bytes, uncombined header occurrences and
actual method/absolute target/authority or response status. These values must
come from the trusted HTTP stack. Deserializing peer JSON into the message type
is not a transport integration. Do not derive the receiving endpoint from Host,
Forwarded or X-Forwarded fields. TLS server authentication, HTTP framing and
routing remain the transport's responsibility. In particular, already combined
headers cannot be used to claim duplicate-field or request-smuggling protection.

## Admitted profile

- Ed25519, exactly one `sig1`, the chapter 03 ordered request/response coverage,
  required `keyid`, `alg`, `created`, `expires`, `nonce`, `tag` parameters and the
  final `@signature-params` line. Received parameter order is preserved.
- The canonical Structured Fields serialization subset: single SP between covered
  components, bare `;req`, no optional whitespace, escaped strings or alternate
  integer spelling. Parameters may occur in any order. This is not a general
  RFC 8941 parser and does not claim acceptance of every RFC 9421 serialization.
- HTTPS POST to one configured canonical absolute URI with a path, lowercase
  authority and omitted default port. Other methods, redirects, target rewriting
  and general URI normalization are outside this API.
- Exact `application/json`, `x-sage-version: 0.10.0`, one SHA-256 Content-Digest,
  canonical standard-base64 HTTP signature, matching DID and envelope parameters.
  Optional message/context/task projections must match existing body fields.
- Session content at most 32 KiB, at most 256 field occurrences, total field size
  at most 32 KiB counted as name + value + 4 bytes, and each signature field at
  most 8 KiB. These are deliberately stricter admission limits than the general
  16 MiB transport bound. No content coding, transfer coding or trailers are
  admitted at this boundary. A supplied Content-Length must match exact content.
- Duplicate critical fields are rejected case-insensitively. Multipart,
  X-SAGE-Meta projections, unsigned/empty 204 success, additional signature labels,
  unknown/duplicate parameters and control bytes fail closed.
- Session freshness remains strict `now < expires`, with at most 300 seconds
  lifetime and the existing future-created allowance. The HTTP late grace does
  not relax the inner session envelope's earlier rejection boundary.

## Acceptance and retained request state

Content-Digest is checked against exact bytes before JSON parsing. Outer HTTP
signature, header/envelope agreement, inner signature, pinned tuple/current keys,
AEAD and response correlation must all succeed before the existing replay store's
single final publication callback. One acceptance consumes one ID/nonce reservation
and one record sequence, confirms a provisional responder if applicable, and
marks a response terminal. Failed validation does not release plaintext or consume
these states. Registry/lifetime failures may close the session as before.

The HTTP request's method, target, authority, digest, exact Signature field and
version are copied into private retained request state. Response `;req` components
use that state, never caller-provided hashes or a reconstructed response context.
The returned message may be modified by its caller without modifying retained
context; modified transmissions will not validate as the original request.

Go holds the endpoint mutex throughout the operation; Rust requires exclusive
ownership. The existing five-second operation gate and final key/lifetime checks
cover HTTP processing as well. Outbound record creation is irreversible: on a
later emission failure the session is closed, and retries reuse previously
returned exact message bytes rather than resealing.

The test replay store, registry and clocks remain synthetic. Tests do not establish
production persistence, restart quarantine, chain finality, TLS, WebSocket,
application authorization, hook coverage or host-compromise resistance.

## Verification

`http-session010.json` enumerates 45 common unit scenarios. The Inspector runs
180 process scenarios across all four Go/Rust pairings, computes HTTP signature
bases and body hashes in Python, and independently signs/verifies public fixture
keys with Node crypto. Negative cases are bounded local test inputs; no external
targets, production credentials or host-bypass programs are involved.

The local `rfc9421` repository's `en.txt` is the RFC reference. Its legacy builder
and parser are not a SAGE conformance oracle: they do not enforce this exact
profile or its envelope/replay transaction. Historical Inspector FAIL and
UNSUPPORTED observations remain unchanged. Full conformance is NOT_ESTABLISHED.
