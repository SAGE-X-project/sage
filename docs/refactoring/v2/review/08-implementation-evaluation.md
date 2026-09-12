# 08. Implementation evaluation against common practice (2026-09-12)

Scope: how each technology in SAGE is implemented compared with the usual
implementation of the same technology (standard library, widely used library,
or the RFC's recommended profile), and how much of it is exercised on real
request paths versus tests. Read-only review of the Go core at `e98b42b`,
`rs-sage-core` at `206bbbb`, `sage-spec` and `sage-gateway` (`4e72668`) as
checked out on 2026-09-12. Nothing was executed except one scratch program
importing `pkg/agent/crypto/jcs` to observe outputs.

Path prefixes: `G/` = `sage` (Go core), `R/` = `rs-sage-core`, `S/` =
`sage-spec`, `GW/` = `sage-gateway`. Confidence `[High]` = read in the cited
lines; `[Mid]` = one inference from cited evidence; `[Low]` = not verified.
Severity `[critical]` = exploitable or data loss; `[major]` = wrong against
spec/RFC, or a real security or interoperability defect; `[minor]` = hygiene.

Headline: the primitives, the RFC 9421 profile and the HPKE handshake use
mainstream libraries and mostly follow the RFCs, and the Rust core passes
all 26 spec vectors. The gap is in what is wired together. Only RFC 9421
signing reaches a shipped request path (through `sage-gateway`); the HPKE
handshake and the session record layer, which carry the "E2E encryption"
claim, have no caller outside tests and examples, and the Go HTTP/WebSocket
transports neither sign, verify, nor encrypt anything themselves.

---

## 1. Cryptographic primitives and key handling

**Common practice.** Ed25519 from the platform library (RFC 8032 §5.1.7;
prehash discouraged, §8.5). secp256k1 in the Ethereum convention: Keccak-256,
RFC 6979 §3.2 nonces, low-S, 65-byte `r||s||v`. P-256 via `crypto/ecdsa`
(hedged randomness; RFC 6979 when `rand` is nil, https://pkg.go.dev/crypto/ecdsa).
X25519 via `crypto/ecdh`; the all-zero check is a MAY in RFC 7748 §6.1. Key
ids by RFC 7638 §3 thumbprint. Keys at rest encrypted, files 0600, wiped in
memory (`sodium_memzero`, https://doc.libsodium.org/usage). Constant-time
comparison for every tag.

**SAGE.**
- Ed25519: Go `crypto/ed25519`, no prehash (`G/pkg/agent/crypto/keys/ed25519.go:23,38,70,76`);
  `filippo.io/edwards25519` only for the Ed25519 to X25519 conversion
  (`keys/x25519.go:35,326-330`). Rust `ed25519-dalek 2.1` with `verify`, never
  `verify_strict` (`R/src/crypto/keys.rs:390-395`). [High]
- secp256k1: decred keygen (`keys/secp256k1.go:25,38`); go-ethereum sign/verify
  over Keccak-256, RFC 6979, v in {0,1} (`keys/secp256k1_keccak.go:35-39`);
  verify accepts 64/65 bytes and normalises high-S instead of rejecting
  (`secp256k1_keccak.go:49-60`; `ecdsa_encoding.go:85-101`). Rust `k256`
  `sign_prehash_recoverable` + `normalize_s` (`R/src/crypto/keys.rs:349-364,398-406`). [High]
- P-256: Go randomised (`keys/p256.go:110-119`); Rust deterministic (`R/src/crypto/p256.rs:116-118`). [High]
- X25519: `crypto/ecdh` (`keys/x25519.go:26,52,112-118`); zero check in
  `sharedSecret()` (`x25519.go:333-341`) and the HPKE combiner
  (`G/pkg/agent/hpke/common.go:99-105`). Rust `diffie_hellman` has none
  (`R/src/crypto/x25519.rs:231-245`); the HPKE layer checks (`R/src/hpke/common.rs:155-157`). [High]
- RSA: PKCS#1 v1.5 (`keys/rs256.go:79,91`; `G/pkg/agent/core/rfc9421/verifier_http.go:180,496`)
  advertised as `rsa-pss-sha256` (`G/pkg/agent/crypto/algorithm_registry.go:100-104`). [High]
- Key id `hex(SHA-256(pub)[:8])` (`keys/keyid.go:33-36`), not RFC 7638; the
  secp256k1 key pair hashes the compressed key (`keys/secp256k1.go:46-47`),
  while the spec, the vector and Rust hash the uncompressed key
  (`G/pkg/vectors/crypto.go:82-92`; `R/src/crypto/keys.rs:100-107,148-149`). [High]
- Storage: Go writes plaintext JWK with `d`, dir 0700, file 0600
  (`crypto/storage/file.go:53,83-104`; `formats/jwk.go:49,85`); the AES-GCM +
  PBKDF2 vault (`crypto/vault/secure_storage.go:36,103,197`) has no importer.
  Rust `fs::write` with no permission call (`R/src/crypto/storage/file.rs:96-103`),
  raw scalars under `PRIVATE KEY` labels, not PKCS#8 (`R/src/formats/mod.rs:388-401`). [High]
- No `math/rand` in non-test Go; Rust `OsRng`. Tags compared with
  `subtle`/`hmac.Equal` (`G/pkg/agent/hpke/client.go:152-153,450-500`;
  `G/pkg/agent/session/session.go:715,773`; `R/src/hpke/common.rs:147,156`). [High]
- Zeroisation: Go wipes HPKE intermediates only (`G/pkg/agent/hpke/server.go:180-215`).
  Rust `PrivateKey` derives `Debug`/`Clone`, no `Zeroize` (`R/src/crypto/keys.rs:176-184`). [High]

| # | Finding | Severity | Conf. |
|---|---|---|---|
| 1.1 | Rust file store writes private keys with umask permissions; Go sets 0600 | [major] | [High] |
| 1.2 | RSA signs PKCS#1 v1.5 under `rsa-pss-sha256`; a PSS-implementing RFC 9421 peer rejects every signature (spec O-3) | [major] | [High] |
| 1.3 | Go secp256k1 `KeyPair.ID()` disagrees with the spec key id used by vectors and Rust | [minor] | [High] |
| 1.4 | Rust `PrivateKey` prints scalar bytes via `Debug` and is never zeroised | [minor] | [High] |
| 1.5 | Rust uses non-strict Ed25519 `verify`; Go stdlib criteria undocumented; neither rejects small-order `R` | [minor] | [Mid] |
| 1.6 | Go P-256 randomised vs Rust RFC 6979; verify-compatible, but P-256 vectors are verify-only for this reason (O-2) | [minor] | [High] |
| 1.7 | Default key store is plaintext; the encrypted vault is dead code | [minor] | [High] |

**Exercised.** `keys` is on every signing path. P-256 is generated by
`G/pkg/agent/core/core.go:59-60` and verified in rfc9421; RSA has no
production key-generation caller. `rotation/` and `storage/` are used only by
`cmd/sage-crypto`; `vault/` by nothing. Fuzzing covers Ed25519 and JWK/PEM
round trips only (`G/pkg/agent/crypto/fuzz_test.go:30-234`).

**Recommendation.** Set 0600 in the Rust store and add `Zeroize` to
`PrivateKey` (no trade-off). Rename RSA to `rsa-v1_5-sha256` or drop it as
Rust did: renaming keeps legacy keys but leaves a deprecated algorithm in a
new spec; dropping loses nothing observable. Hash the uncompressed key in
`secp256k1KeyPair.ID()`, gated by the storage version since stored ids change.

---

## 2. RFC 9421 HTTP Message Signatures

**Common practice.** RFC 9421 §2.3 parameters, `;req` binding (§2.4),
verification steps (§3.2), nonce replay (§7.2.2), RFC 9530 `Content-Digest`
(§7.2.8), algorithm from key material and configuration rather than `alg`
alone (§7.3.6). `github.com/yaronf/httpsign` v0.6.0 parses with RFC 8941,
supports `;req`, `tag`, `@query-param`, automatic `Content-Digest`, and
`SetNotOlderThan` (default 10 s), `SetKeyID`, `SetNonceValidator`
(https://pkg.go.dev/github.com/yaronf/httpsign). Mastodon signs draft-cavage
`(request-target) host date digest` with a 12-hour Date window and enabled
RFC 9421 by default in 4.5 (https://docs.joinmastodon.org/spec/security/).
AWS SigV4 is the contrast: HMAC with a derived key chain, canonical headers,
hashed payload, no nonce, freshness by date skew only
(https://docs.aws.amazon.com/IAM/latest/UserGuide/create-signed-request.html).

**SAGE.**
- Base: one line per component plus `@signature-params`
  (`G/pkg/agent/core/rfc9421/canonicalizer.go:66-83`); derived components
  `@method @target-uri @authority @scheme @request-target @path @query
  @query-param @status` (`canonicalizer.go:152-161,183-297`). [High]
- Hand-rolled parsing, not RFC 8941 (`parser.go:39-70,146-303`); `;req`
  supported and rejected on requests (`canonicalizer.go:104-150`); `;sf`,
  `;bs`, `;key` absent; unknown parameters copied verbatim (`canonicalizer.go:131,179`). [High]
- `@signature-params` is re-serialised from the parsed struct in fixed order
  and `tag` is dropped (`canonicalizer.go:300-326`; `parser.go:219-238`);
  Rust reuses the raw member text (`R/src/rfc9421/dictionary.rs:59-66`). [High]
- `alg` must match the key type; the key type selects the algorithm
  (`algorithm_registry.go:298-324`; `verifier_http.go:463-503`). MaxAge and
  skew 5 min (`verifier_http.go:41-44,355-374`). [High]
- `Content-Digest` sha-256 only, 16 MiB cap, verified when covered
  (`body_integrity.go:45,73-76,170-174,200-216`). Responses cover `@status`
  and `;req` method/target/authority/digest/signature (`response_signer.go:96-111`),
  no nonce or expires (`response_signer.go:65-70`). [High]
- Replay: TTL map by full `keyid`, checked after the signature verifies, no
  size cap (`verifier_http.go:242,303,433-440`; `G/pkg/agent/session/replay.go:42-80`). [High]
- `X-SAGE-DID` is not checked by the Go verifier; Rust checks
  (`R/src/rfc9421/verifier.rs:159-169`), the gateway checks before calling it
  (`GW/pkg/gateway/verify/verify.go:97-102`). Errors are step-specific
  (`verifier_http.go:313-436`); the MCP example returns them to the client
  (`G/examples/mcp-integration/basic-tool/calculator_tool.go:104-106`), the
  gateway returns a generic 401 (`verify.go:73-76`). [High]

| # | Finding | Severity | Conf. |
|---|---|---|---|
| 2.1 | Go `@request-target` is `"METHOD path?query"` (`canonicalizer.go:232-241`), the draft-cavage form; RFC 9421 §2.2.5 is the request-target alone; Rust emits `path?query` (`R/src/rfc9421/canonicalize.rs:60-67`). Signatures covering it cannot interoperate | [major] | [High] |
| 2.2 | Re-serialised `@signature-params` and dropped `tag` reject conformant third-party signers (§2.3, §7.2.7) | [major] | [High] |
| 2.3 | Hand-rolled Structured Fields; no `;sf`/`;bs`/`;key`; `@authority` not lowercased (`canonicalizer.go:216-220`) | [minor] | [High] |
| 2.4 | Go verifier does not enforce `X-SAGE-DID == keyid` DID (spec 03 §3) | [minor] | [High] |
| 2.5 | Gateway takes the label from a map iteration (`verify.go:93-96`; `ParseSignatureInput` returns a map, `parser.go:40-41`): nondeterministic with two signatures (§7.2.6) | [minor] | [High] |

**Exercised.** Inside `sage` no `cmd/` binary and neither transport calls
`SignRequest`/`VerifyRequest` (`G/pkg/agent/transport/http/client.go:21-31`,
`server.go:21-30`). The real path is `sage-gateway`: `sign.Transport` covers
`@method @target-uri @authority content-type content-digest x-sage-did date`
with a UUID nonce (`GW/pkg/gateway/sign/sign.go:62-98`); `verify.Middleware`
runs strict verification with `ExpectedDID` and on-chain or static keys
(`GW/pkg/gateway/verify/verify.go:84-131`). Vectors are checked in CI by the
generating code (`G/.github/workflows/test.yml:169-192`) and independently by
Rust (`R/tests/spec_vectors.rs:261-389`).

**Recommendation.** Fix 2.1 and 2.2 before any external peer exists: emit
the raw `Signature-Input` member into the base and drop the method from
`@request-target`. Trade-off: existing Go signatures over those components
stop verifying, but no shipped peer covers `@request-target`. Replacing
`parser.go` with an RFC 8941 parser (httpsign's or `dunglas/httpsfv`) removes
a class of edge cases at the cost of a dependency.

---

## 3. HPKE and the 1-RTT handshake

**Common practice.** RFC 9180 Base mode has no sender authentication (§9.1,
Table 6); the exporter (§5.3) is the sanctioned way to feed other protocols,
as in Oblivious HTTP `Export("message/bhttp response", ...)` (RFC 9458 §4.4)
and MLS `EncryptWithLabel` with the `"MLS 1.0 "` prefix (RFC 9420 §5.1.3);
bidirectional keys via export (§9.8). Go: `cloudflare/circl/hpke` v1.6.5
(https://pkg.go.dev/github.com/cloudflare/circl/hpke); Rust: `hpke` 0.14.1
with `OpModeS::Base` and `ExportOnlyAead` (https://docs.rs/hpke). Noise NK
(§7.5) is `e, es / e, ee`; its first message is "vulnerable to replay"
(§7.7). TLS 1.3 guards 0-RTT with single-use tickets, ClientHello recording
and freshness checks (RFC 8446 §8.1-8.3).

**SAGE.**
- circl Base mode, X25519/HKDF-SHA256/ChaCha20-Poly1305, export only
  (`G/pkg/agent/crypto/keys/x25519.go:359-363,370,385,411,424`); Rust the same
  through the `hpke` crate (`R/src/hpke/common.rs:9-12,29-35,50-59`). [High]
- Combiner `HKDF-Extract(exporter||ssE2E, salt=exportCtx)` then Expand with
  `"SAGE-HPKE+E2E-Combiner"` (`G/pkg/agent/hpke/common.go:81-90`; `R/src/hpke/common.rs:64-77`);
  all-zero `ssE2E` rejected (`common.go:99-105`; `client.go:491`; `server.go:313`). [High]
- Traffic keys use a counter HMAC, not RFC 5869 §2.3 (`common.go:187-201`),
  and are unused: the only non-test caller of `DeriveTrafficKeys` is the
  vector generator (`G/pkg/vectors/hpke.go:108`); sessions re-derive from the
  seed with HKDF (`G/pkg/agent/session/session.go:376,409`); same in Rust
  (`R/src/hpke/mod.rs:37,92-105`). [High]
- Ack tag over a SHA-256 transcript of info, exportCtx, enc, ephC, ephS and
  both DIDs, compared with `hmac.Equal` (`common.go:217-231`; `client.go:500`). [High]
- Envelope signed over JCS; client checks `v`, `task`, hashes, echoed
  `enc`/`ephC`, DID, signature, ack (`server.go:365-393`; `client.go:436-475`).
  Rust re-serialises a typed struct, dropping unknown members (`R/src/hpke/types.rs:255-261`). [High]
- Server: nonce store per `ctxID`, 10 min TTL, no cap (`server.go:116-118,261`);
  ±2 min timestamp (`server.go:113-115,257-260`); `initDid` must equal the
  transport signer (`server.go:249`). DID resolution and signature verification
  precede the cookie check (`server.go:147-164`), reverse of spec 04 §7 (O-6). [High]
- The initiator signs the raw init payload with its DID key (`client.go:259-276`);
  its static key enters no DH. The responder proves identity with `sigB64`
  plus the ack tag and binds the session before any initiator confirmation
  (`server.go:193,320-337`). [High]

**Structural assessment.** The shape is Noise NK (`e, es` to the registry
KEM key, then `ee`) with SIGMA-style signatures on both sides rather than
IK; closer to TLS 1.3 with a static server KEM than to IK. [Mid] The first
message has NK's replay property; the per-context nonce store plus the
timestamp window is the RFC 8446 §8.2/§8.3 pattern without single-use
tickets. [High] Missing initiator key confirmation is normal for 1-RTT; its
effect is a responder-side session the initiator may never use. [Mid]

| # | Finding | Severity | Conf. |
|---|---|---|---|
| 3.1 | Public-key work (resolution, signature verify, parse) precedes the cookie check; the cookie cannot shield the responder from unauthenticated floods (O-6) | [major] | [High] |
| 3.2 | The counter expansion and its five outputs are specified, vectored and unused; two KDFs for one secret (O-7) | [minor] | [High] |
| 3.3 | Rust verifies over a re-serialised struct; a Go responder adding a member breaks Rust clients | [minor] | [High] |
| 3.4 | Go returns distinct errors for parse, signature, cookie, suite, open (`server.go:155,162,238,274,292`); only `validateInitEnvelope` is uniform | [minor] | [High] |
| 3.5 | Ephemeral `*ecdh.PrivateKey` is never wiped; Rust uses `Zeroizing` | [minor] | [High] |

**Exercised.** `hpke.NewServer`/`NewClient` have no caller outside the
package tests (grep over `pkg cmd examples sdk`); transports carry
`signature` opaquely (`G/pkg/agent/transport/wire.go:31`); `sage-gateway`
excludes HPKE (`GW/README.md`, "What it does not do"). Tests are thorough
for the code that exists (`security_test.go:227-764`); the six `hpke`
vectors pass in Rust CI.

**Recommendation.** Make the cookie check the first step and mandatory on
public listeners (one HMAC per request). Remove the counter expansion from
spec and cores (a MAJOR spec bump by 00 §5). Decide whether the handshake is
a product: wire it into one transport with a Go-Rust conformance test
(backlog F-03b) or mark it experimental.

---

## 4. Session layer

**Common practice.** TLS 1.3: nonce = IV XOR sequence (RFC 8446 §5.3),
2^64-1 record limit and KeyUpdate (§5.5, §4.6.3), HKDF-Expand-Label (§7.1).
QUIC: same XOR nonce (RFC 9001 §5.3), unique packet numbers (RFC 9000 §12.3).
DTLS 1.3 anti-replay bitmap window (RFC 9147 §4.5.1). Noise `Rekey()` sets
`k = REKEY(k)` so old keys are not derivable (Noise §5.1). RFC 8439 §4
recommends counters over random nonces.

**SAGE.**
- `x/crypto/chacha20poly1305` (`G/pkg/agent/session/session.go:36`); Rust
  `chacha20poly1305 0.10` (`R/src/session/secure_session.rs:16-19`). Record
  `be64(seq) || nonce[12] || AEAD`, random nonce, `aad = be64(seq) || callerAAD`
  (`session.go:896-903,933-937`; `secure_session.rs:326-344`). [High]
- One `sendSeq` under a mutex shared by every entry point, no overflow check
  (`session.go:76-77,886-889`). 1024-slot bitmap marked after `Open` succeeds
  (`session.go:151-196,914-929`); single-winner concurrency test
  (`session_replay_test.go:166-187`). [High]
- Rekey per 256 records from the retained seed, not the previous key
  (`session.go:835-867,615`; `secure_session.rs:278-283`). [High]
- Directional and shared keys on one object with one counter and window;
  `EncryptAndSign` always uses the shared key (`session.go:262-279,685-688,731,748`);
  Rust's MAC path uses `covered` as AAD and directional keys
  (`secure_session.rs:489-503`). [High]
- Lifetime (1 h / 10 min / 1000, `manager.go:46-51`) is checked only in
  `Encrypt`/`Decrypt`/`EncryptAndSign`/`DecryptAndVerify` (`session.go:646,663,682,704`);
  `EncryptWithAAD`, `EncryptOutbound`, `DecryptInbound` and AAD variants never
  call `IsExpired` (`session.go:730-816`). Rust checks in `seal`/`open`
  (`secure_session.rs:317-318,351,376`). [High]

| # | Finding | Severity | Conf. |
|---|---|---|---|
| 4.1 | Directional entry points (the ones a role session from the handshake would use) bypass MaxAge, IdleTimeout and MaxMessages; the Manager re-checks only on `GetSession` (`manager.go:294`) | [major] | [High] |
| 4.2 | No forward secrecy between generations: every generation derives from the retained seed, unlike Noise `REKEY(k)` or TLS KeyUpdate chaining | [minor] | [High] |
| 4.3 | Random 96-bit nonce plus explicit 8-byte sequence costs 20 bytes/record where TLS/QUIC cost 0; safe at 256 records per key, but two uniqueness mechanisms | [minor] | [High] |
| 4.4 | Shared and directional keys share one counter and window with no guard, contradicting spec 05 §2 "MUST NOT mix" | [minor] | [High] |
| 4.5 | Go and Rust `EncryptAndSign` records are not interoperable | [minor] | [High] |
| 4.6 | `sendSeq` wraps silently; TLS mandates closure at 2^64-1 | [minor] | [High] |

**Exercised.** Production code only creates sessions
(`G/pkg/agent/hpke/client.go:508-518`; `server.go:321-336`). No non-test code
in `pkg/`, `cmd/`, `internal/` calls `Encrypt*`/`Decrypt*`; `pkg/agent/transport`
has no session reference; the only caller is `G/examples/metrics-demo/main.go:139-146`,
which is also the only place the Prometheus session adapter is wired
(`G/pkg/telemetry/metrics/session_adapter.go:28`; `main.go:122`).

**Recommendation.** Route every entry point through one `seal`/`open` that
checks lifetime and counts messages, as Rust does. Consider `IV XOR seq`
nonces in a v2 record (MAJOR spec bump; loses the "nonce independent of seq"
property the vectors pin). Chain rekeys from the previous key if forward
secrecy matters; otherwise state in the spec that it does not.

---

## 5. JSON Canonicalization (RFC 8785)

**Common practice.** `github.com/gowebpki/jcs` (https://pkg.go.dev/github.com/gowebpki/jcs)
and `serde_jcs` with `ryu-js` (https://docs.rs/serde_jcs): UTF-16 key order
(§3.2.3), ECMA-262 numbers (§3.2.2.3), escapes (§3.2.2.2).

**SAGE.** Both cores hand-write the canonicaliser. Go decodes with
`UseNumber`, routes every number through `float64` and rebuilds the ES6
layout from `FormatFloat('e')` (`G/pkg/agent/crypto/jcs/jcs.go:58-59,85,177-225`),
sorts by UTF-16 units (`jcs.go:132-140`), escapes per the RFC (`jcs.go:143-174`).
Observed: `1E30 -> 1e+30`, `-0 -> 0`, `9007199254740993 -> 9007199254740992`,
`1e400` rejected; duplicate keys accepted last-wins and lone surrogates
become U+FFFD (stdlib decoder). Rust has its own parser and rejects
duplicates, lone surrogates and raw control characters
(`R/src/jcs/mod.rs:166-175,186,247-249,296-380`). [High]

| # | Finding | Severity | Conf. |
|---|---|---|---|
| 5.1 | Go accepts inputs Rust rejects; a Go-signed object built from such JSON verifies in Go and fails in Rust | [minor] | [High] |
| 5.2 | Two custom parsers where audited libraries exist; RFC Appendix B number samples absent from the vectors | [minor] | [High] |

**Exercised.** HPKE envelope (`G/pkg/agent/hpke/server.go:368`; `client.go:394`)
and A2A card proof (`G/pkg/agent/did/a2a_proof.go:77-88,127`): the proof
through `cmd/sage-did` and vectors, the envelope through tests only.

**Recommendation.** Reject duplicate keys and invalid surrogates in Go (a
decoder pre-pass; no wire change). Switching both cores to `gowebpki/jcs`
and `serde_jcs` removes about 600 lines at the cost of two dependencies.

---

## 6. did:sage, proof of possession, registration, resolution

**Common practice.** DID Core §3.1 `idchar = ALPHA / DIGIT / "." / "-" /
"_" / pct-encoded`; resolution returns a DID Document (§7.1) with
verification methods and relationships (§5.2, §5.3). did:ethr resolves from
ERC-1056 events with no registration and no commit-reveal
(https://github.com/decentralized-identity/ethr-did-resolver/blob/master/doc/did-method-spec.md);
did:pkh is generative and documents no rotation or deactivation
(https://github.com/w3c-ccg/did-pkh/blob/main/did-pkh-method-draft.md);
ERC-8004 registers ERC-721 agents via `register(agentURI)` with no
commit-reveal (https://eips.ethereum.org/EIPS/eip-8004, Draft).

**SAGE.**
- Grammar: split at three colons, chain aliases, identifier unrestricted
  (`G/pkg/agent/did/manager.go:313-341`; `R/src/did/mod.rs:53-84`). [High]
- Resolution returns `AgentMetadata`, not a DID Document (`resolver.go:38-52`;
  O-8). No cache in the core (grep of `pkg/agent/did` finds only a comment,
  `types_v4.go:153`); the gateway adds a TTL cache with negative entries
  (`GW/pkg/gateway/resolve/resolve.go:106-155`). `is_active` enforced in
  `ResolvePublicKey`/`ResolveKEMKey` (`resolver.go:121-141`). Key selection
  trusts the on-chain `verified` flag (`ethereum/key_policy.go:22-35,75-78`). [High]
- PoP: `SHA-256("SAGE-PoP:"||DID||":"||hex(key))`, Ed25519 over the digest,
  secp256k1 over SHA-256 (`key_proof.go:51,60,68,108,117,153,218-219`); no
  chain id, registry or nonce. Verified only by `cmd/sage-did/key.go:610` and
  the vector generator; never on the resolve path. [High]
- On chain (pinned `sage-contracts`), `_verifyKeyOwnership` checks ECDSA keys
  by `ecrecover` over a message without the key, marks Ed25519 keys
  `verified` after `length == 64` only ("can't verify on-chain"), verifies
  X25519 by owner signature (`AgentCardRegistry.sol:187-193,418-465`). [High]
- Commit-reveal: `keccak256(abi.encode(did, keys, owner, salt, chainId))`,
  1-60 min reveal, 1 h activation, salt persisted by the CLI at 0600
  (`ethereum/agentcard_client.go:115-251`; `cmd/sage-did/commit.go:178-218`);
  the CLI `commit` sends empty key arrays (`commit.go:126-128`, TODO), which
  the contract rejects (`AgentCardRegistry.sol:154`). [High]
- A2A card: `<DID>#key-<n>`, 2019/2020 suite names, base58 + hex, proof over
  JCS without `proof`; `created` lives inside `proof`, so it is unsigned and
  unchecked (`a2a.go:48,108-115`; `a2a_proof.go:77-88,142-167`). With a
  resolver, the key must be `verified` and the agent active (`a2a_proof.go:230-261`).
  Rust has no resolver step; `BlockchainDIDResolver` always returns
  `Unsupported` (`R/src/did/resolver.rs:30-36`). [High]

| # | Finding | Severity | Conf. |
|---|---|---|---|
| 6.1 | Ed25519 keys are `verified` on chain without proof and the off-chain PoP is not checked at resolve time: anyone can register another agent's Ed25519 key under their own DID. Exploitability is limited because signed objects bind the sender DID, but the flag the resolver trusts is unearned | [major] | [High] |
| 6.2 | PoP is not bound to chain id or registry (valid across mainnet, Sepolia, Kaia and every deployment); secp256k1 PoP uses SHA-256 unlike every other path (O-1) | [major] | [High] |
| 6.3 | No DID Document projection; not consumable by standard DID resolvers (O-8) | [minor] | [High] |
| 6.4 | Identifier accepts any bytes; no core cache despite CLAUDE.md "resolver.go: DID 문서 해석 + 캐싱" | [minor] | [High] |
| 6.5 | `sage-did commit` cannot succeed against the pinned contract (empty keys) | [minor] | [High] |
| 6.6 | Commit-reveal with stake and 1 h activation is unusual (did:ethr, did:pkh, ERC-8004 have none); prevents name front-running at the cost of three transactions and an hour | [minor] | [High] |

**Exercised.** The resolver is on the HPKE and legacy handshake paths and,
in production, on the gateway verify path (`GW/pkg/gateway/resolve/resolve.go:173-204`).
Registration is CLI-only. Rust DID code is used only by FFI/WASM and vectors.

**Recommendation.** Bind the PoP to `chainid || registry` and verify it in
`key_policy.go`; cost is one verification per resolved key, amortised by the
gateway cache. Publish an additive DID Document projection from
`AgentMetadataV4`.

---

## 7. Transport, API surface, observability, FFI/WASM

**Common practice.** Go servers set `ReadHeaderTimeout`/`ReadTimeout`/
`WriteTimeout`/`IdleTimeout` and TLS minimums; gorilla/websocket allows one
concurrent writer per connection and needs ping/pong. Generic errors to
clients, details to `log/slog`. Cross-language crypto libraries follow
libsodium (`sodium_init`, 0/-1 returns, caller buffers, `sodium_memzero`,
https://doc.libsodium.org/usage) or BoringSSL (1/0 returns, error queue,
`_new/_free`, `out/out_len/max_out`,
https://github.com/google/boringssl/blob/master/API-CONVENTIONS.md); Rust FFI
wraps entries in `catch_unwind` because unwinding out of `extern "C"` aborts.

**SAGE (Go).**
- HTTP: JSON `WireMessage` to `/messages` with `X-SAGE-*` headers
  (`G/pkg/agent/transport/http/client.go:85-114`); header/body disagreement
  rejected (`server.go:166-179`); 1 MiB cap (`server.go:63,102-108`); handler
  errors returned verbatim as HTTP 200 `success:false` (`server.go:199-206`).
  No RFC 9421 call; no TLS configuration in `pkg/`, `cmd/`, `internal/`;
  `http://` registered as a transport (`http/register.go:29-34`); `HTTPServer`
  is only an `http.Handler` (`server.go:93,221-223`). [High]
- WebSocket: 1 MiB read limit, 60 s read deadline (`websocket/server.go:78,217,233`);
  no ping/pong; `Close()` calls `WriteMessage` on every connection while the
  read loop may be in `WriteJSON` (`server.go:294-305,329-344`); no auth at
  upgrade; missing `Origin` accepted by default (`server.go:168-172`). [High]
- Logging is `fmt.Printf` in transports and the session manager
  (`http/server.go:116,216`; `websocket/server.go:242,297,303`; `session/manager.go:311,425,460,482`);
  no secrets printed. Prometheus `sage_*` families exist
  (`G/pkg/telemetry/metrics/{handshake,crypto,message,session}.go`); no OpenTelemetry. [High]
- Usage: no `cmd/` binary starts either transport; only `internal/app/register.go:31-32`
  imports them. The SDKs reimplement crypto and post to an A2A demo endpoint
  (`G/sdk/python/sage_client/client.py:136-290`; `G/sdk/typescript/package.json:46-53`);
  the TypeScript SDK signs secp256k1 over SHA-256 (`G/sdk/typescript/src/crypto.ts:37-59`). [High]
- `sage-gateway`: listener timeouts (`GW/cmd/sage-gateway/main.go:225`),
  `slog`, generic 401, 16 MiB cap; no TLS. [High]

**SAGE (Rust FFI/WASM).**
- 52 `extern "C"` functions; `c_int` codes with a thread-local last error
  (`R/src/ffi/mod.rs:25-78,142-187`); type-specific free functions; caller
  buffers with in/out length (`R/src/ffi/signature.rs:115-138`; `spec.rs:18-29`);
  null checks on every entry; no `catch_unwind` in `R/src`; `sage_keypair_import`
  trusts the caller's length for `from_raw_parts` (`keypair.rs:180`); many
  failures return a code without a message (`keypair.rs:25,183`; `signature.rs:34`). [High]
- cbindgen header tracked and checked in CI (`R/cbindgen.toml`;
  `R/include/sage_crypto.h`; `R/.github/workflows/ci.yml:58-66`). Exposed: keys,
  sign/verify, formats, RFC 9421, JCS, DID, PoP, A2A, `SecureSession`. Not
  exposed: HPKE (grep `hpke` in `src/ffi`, `src/wasm` empty). Private keys
  exported unzeroised (`R/src/ffi/keypair.rs:140-146`; `R/src/wasm/keypair.rs:60-76`).
  No `wasm-bindgen-test`; CI checks only that the `.wasm` exists (`ci.yml:95-99`). [High]

| # | Finding | Severity | Conf. |
|---|---|---|---|
| 7.1 | No TLS and no session encryption on any transport: payloads on `http://`/`ws://` are signed but plaintext, contradicting the E2E claim | [major] | [High] |
| 7.2 | Rust FFI has no `catch_unwind`; any panic aborts the host process | [major] | [High] |
| 7.3 | WebSocket server writes from two goroutines without a lock (`Close` vs `sendResponse`) | [major] | [High] |
| 7.4 | SDKs do not bind the Rust core; four crypto reimplementations to keep in sync, one already diverged (SHA-256 secp256k1) | [major] | [High] |
| 7.5 | Internal error strings returned to peers with HTTP 200; no structured logging | [minor] | [High] |
| 7.6 | `sage_keypair_import` reads `private_key_len` bytes unchecked against the key type | [minor] | [High] |

**Recommendation.** Wrap every FFI entry in `catch_unwind` returning an
`Internal` code (one macro). Add a per-connection write mutex and ping/pong.
Decide TLS policy: require `https`/`wss` unless `AllowInsecure` is set, or
document that transport confidentiality depends on wiring the session layer
(§4). Point the SDKs at the WASM/FFI surface; the trade-off is a native build
step per SDK against one implementation to audit.

---

## 8. Test strategy

**Common practice.** Unit and integration tests; published vectors consumed
by every implementation; fuzzing in CI with a time budget; property tests for
encoders and parsers (`proptest`, `rapid`); cross-implementation runs; a
symbolic model (Tamarin, ProVerif, Verifpal) for a new handshake, as Noise,
TLS 1.3, HPKE and MLS had.

**SAGE.**
- Go: 396 `Test`, 12 `Fuzz`, 34 `Benchmark` under `pkg/`+`cmd/`; CI runs
  `go test -race` with coverage upload and no threshold
  (`G/.github/workflows/test.yml:42-45`), lint, gosec, govulncheck, CodeQL,
  gitleaks, Trivy. Rust: 481 `#[test]`, three criterion benches, `proptest`
  declared and unused (`R/Cargo.toml:78`; grep empty), Miri weekly with
  `continue-on-error: true` (`R/.github/workflows/extended.yml:113`). [High]
- Vectors: 26 in six suites. Go checks them in CI with the generating code
  (`G/.github/workflows/test.yml:169-192`; `G/pkg/vectors/vectors.go:161-197`);
  Go unit tests only round-trip a temp dir (`G/pkg/vectors/vectors_test.go:11-31`).
  Rust loads the files and runs every suite (`R/tests/spec_vectors.rs:27-39,57-700`).
  Go never consumes Rust output; `Check` rejects unknown vectors
  (`vectors.go:186-190`); no workflow builds both toolchains. [High]
- Hardhat: `make test-integration` passes `-tags=integration` but no file
  carries that tag (only `e2e`, `G/tests/integration/e2e_sepolia_test.go:1`);
  tests gate at runtime on `SAGE_RPC_URL`/`-short` (`G/tests/integration/test_helper.go:59-70`). [High]
- Fuzz: two files (crypto, session); nothing for the RFC 9421 parser, JCS,
  DID grammar, HPKE envelope or session records. `run-fuzz.sh` swallows
  failures with `|| true` and runs in no workflow; `cmd/random-test` and
  `make random-test` in CLAUDE.md do not exist. [High]
- Security tests: HPKE tamper/replay/downgrade (`G/pkg/agent/hpke/security_test.go:227-764`),
  rfc9421 replay/skew/body swap (`verifier_http_security_test.go:44-197`),
  session replay (`session_replay_test.go:42-216`). Go's "timing" test only
  asserts inequality (`body_integrity_edge_test.go:271-289`); Rust measures a
  100-iteration ratio (`R/tests/security_tests.rs:7-41`). High-S acceptance is
  tested as intended (`G/pkg/agent/crypto/keys/verify_test.go:53-56`). [High]

| # | Finding | Severity | Conf. |
|---|---|---|---|
| 8.1 | Interoperability is one-directional (Go generates, Rust verifies); no Rust-to-Go test and no live Go-Rust handshake (F-03b open) | [major] | [High] |
| 8.2 | Go's vector check is self-referential; a Go regression that regenerates vectors passes Go CI and is caught only by Rust CI | [major] | [High] |
| 8.3 | No parser fuzzing and no fuzz job in CI | [major] | [High] |
| 8.4 | No property tests despite `proptest`; no formal model of the handshake | [minor] | [High] |
| 8.5 | Only `did.parse` has negative vectors | [minor] | [High] |
| 8.6 | `-tags=integration` matches nothing; no coverage threshold; Miri cannot fail | [minor] | [High] |

**Recommendation.** Add a `verify`-mode ingestion path to `sage-vectors check`
that accepts outputs from another generator, and a `sage-spec` CI job that
runs both cores against a Rust-generated set. Add `FuzzParseSignatureInput`,
`FuzzJCS`, `FuzzParseDID`, `FuzzHPKEEnvelope`, `FuzzOpenRecord` with a 60 s
budget in the weekly workflow. A Verifpal model of the 1-RTT handshake is a
few dozen lines and would settle the key-confirmation question before 1.0.

---

## 9. Summary

| Area | Verdict | Most important finding | Severity | Effort |
|---|---|---|---|---|
| 1 Primitives | Mainstream libraries, correct conventions; hygiene gaps | Rust key files without 0600; RSA alg id mismatch | [major] | low |
| 2 RFC 9421 | Working profile, exercised by the gateway; two interop-breaking deviations | `@request-target` includes method; `@signature-params` re-serialised | [major] | low-medium |
| 3 HPKE | RFC 9180 via circl/hpke crate, sound combiner; not wired | Public-key work before the DoS cookie (O-6) | [major] | low |
| 4 Session | Solid record layer with replay window; not wired | Directional entry points skip lifetime limits | [major] | low |
| 5 JCS | Correct on valid input; two custom parsers | Go accepts what Rust rejects | [minor] | low |
| 6 DID | Bespoke method; trust flags unearned | Ed25519 `verified` without proof; PoP not chain-bound | [major] | medium |
| 7 Transport/FFI | Transports neither authenticate nor encrypt; FFI can abort host | No TLS or session encryption; no `catch_unwind` | [major] | medium |
| 8 Tests | Broad unit coverage, vectors in CI; no parser fuzz, one-way interop | Self-referential Go vector check; no Rust-to-Go direction | [major] | medium |

No finding is [critical]: nothing found lets an unauthenticated party forge
a signature, decrypt a record or take over a DID. The [major] items concern
what is not wired (encryption, cookie order, lifetime checks), what will not
interoperate (RFC 9421 base, RSA id, key ids, MAC path), and what the trust
model assumes but does not verify (Ed25519 PoP).

---

## 10. Facts vs Opinions

**Facts** (read in the cited lines or observed in the scratch run)
- Go HTTP and WebSocket transports contain no RFC 9421 call and no session
  encryption; no `cmd/` binary starts them.
- `hpke.NewServer`/`NewClient` and `session.Encrypt*` have no non-test caller
  in `pkg/`, `cmd/`, `internal/`; `sage-gateway` uses only RFC 9421.
- Go `@request-target` emits `"METHOD path?query"`; Rust emits `path?query`.
- Go rebuilds `@signature-params` from parsed fields and drops `tag`; Rust
  reuses the raw member text.
- The HPKE server verifies the sender signature before checking the cookie.
- Session `EncryptWithAAD`/`EncryptOutbound`/`DecryptInbound` never call
  `IsExpired`; rekey keys derive from the retained seed.
- RSA verifies PKCS#1 v1.5 under `rsa-pss-sha256`.
- Go `secp256k1KeyPair.ID()` hashes the compressed key; the vector, Rust and
  the spec hash the uncompressed key.
- Rust file store sets no permissions; `PrivateKey` derives `Debug`; no
  `catch_unwind` exists in `R/src`.
- The pinned `AgentCardRegistry.sol` marks Ed25519 keys verified after a
  length check; the Go resolve path never calls `VerifyKeyProofOfPossession`.
- Go JCS accepts duplicate keys and lone surrogates; Rust rejects both.
- No file carries `//go:build integration`; `proptest` is unused; no fuzz job
  runs in CI; `cmd/random-test` does not exist; the core DID resolver has no cache.

**Opinions**
- [High] The two RFC 9421 deviations will surface as soon as a non-SAGE
  client (httpsign, Mastodon-style) talks to the gateway; fix before spec 1.0.
- [High] The E2E claim is carried by tests and examples, not shipped code;
  wire the session layer into a transport or reword the claim.
- [Mid] The 1-RTT design is sound as an NK-shaped exchange with signatures
  and replay/freshness guards; missing initiator confirmation is a resource
  concern, not a confidentiality one, but a formal model should confirm this.
- [Mid] Identity misbinding via unverified Ed25519 keys is hard to exploit
  because signed objects carry the sender DID, but it voids the meaning of
  `verified`, which the resolver and the A2A verifier rely on.
- [Mid] Replacing the hand-written Structured Field and JCS parsers with
  maintained libraries removes more risk than any single fix above, at the
  cost of two dependencies and one vector re-verification.
- [Low] Random record nonces were likely chosen to survive counter bugs;
  with rekey at 256 this is safe, but the spec should say so.
