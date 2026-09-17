# HTTP signatures for authenticated session messages

Bind an unused authenticated completion to a trusted canonical HTTPS endpoint
with `BindHTTP` (Go) or `bind_http` (Rust), then use the HTTP request/response
seal/open methods. Binding is permanent: bare session record and response methods
reject calls. To protect the handshake as well, bind the CompletionEndpoint
before any handshake attempt and use StartHTTP/RespondHTTP/CompleteHTTP in Go,
or start_http/respond_http/complete_http in Rust. Their resulting sessions inherit
the HTTP binding automatically. The bare handshake entry points then reject.

The transport supplies exact content bytes, uncombined header occurrences and
actual method/absolute target/authority or response status. These values must
come from the trusted HTTP stack. Deserializing peer JSON into the message type
is not a transport integration. Do not derive the receiving endpoint from Host,
Forwarded or X-Forwarded fields. TLS server authentication and routing remain the transport's responsibility.
The new raw HTTP/1.1 codec supplies bounded framing checks. Already combined
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

## HTTP handshake and raw framing

HTTP completion retains the exact emitted HTTP request in private pending state.
Both HTTP and envelope signatures must pass before the existing handshake replay
reservation. The completion signature, ACK, current pinned keys and final lifetime
gates remain mandatory. Invalid completion consumes one-shot pending material;
there is no unsigned-error or bare-API fallback. Delayed handshake storage may
retain a denial entry while returning no session, as in the original handshake
contract. This does not create a second HTTP replay reservation.

Go `ParseHTTP010` / `EncodeHTTP010` and Rust `parse_http_010` / `encode_http_010`
accept a deliberately bounded HTTP/1.1 carriage: POST origin-form to the configured
path/query, one matching Host, canonical Content-Length, exact body length and
one message per connection. Responses derive status from their status line.
Raw field occurrences are preserved until duplicate rejection; optional leading
and trailing SP/HTAB field whitespace is removed during extraction. Raw field
bytes, including each field line's CRLF, are capped at 32 KiB before whitespace
removal. The start line is capped at 4096 bytes, and normalized field and content
limits still apply. Extra bytes, truncation, obs-fold, malformed names, transfer
coding, Expect, Upgrade, conflicting lengths and non-close Connection values fail.
The encoder validates names/values before serialization to reject header injection.

These codecs do not establish TLS and do not turn unauthenticated bytes into
trusted transport data. A caller must obtain bytes from an authenticated TLS
connection, use locally configured routing, impose absolute I/O deadlines and
close after one exchange. Never dispatch a second request from leftover bytes.
HTTP/2, HTTP/3, pipelining, chunking and general Structured Fields serialization
remain outside this subset. Verify certificates and hostnames; never use an
insecure-skip-verification option to make a fixture or deployment pass.

The Inspector uses a Python/OpenSSL loopback transport with a fresh local test CA,
TLS 1.3 and ALPN `http/1.1`. Actual received bytes enter these core codecs and core
handshake/session APIs. It checks both cores in all four pairings, including
untrusted CA and hostname mismatch with zero HTTP-handler invocations. This proves
the controlled transport integration, not a production Go/Rust TLS service or
host isolation. Malformed framing is exercised offline only, not over sockets.

`http-handshake010.json` adds 22 shared handshake scenarios and records 24 framing
rejection categories. The Inspector executes 88 handshake process scenarios and
16 TLS scenarios; its bounded transport helpers have eight offline unit tests.
Production replay durability, quarantine, registry finality and host enforcement
are still separate requirements, and full conformance remains NOT_ESTABLISHED.
