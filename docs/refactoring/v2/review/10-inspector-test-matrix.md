# 10. Conformance inspector test matrix

Status: review document, 2026-09-12. Builds on `09-protocol-flows.md`
(same commits: `sage` 878932d, `sage-spec` a34dd49, `rs-sage-core` 206bbbb,
`sage-inspector` 05b890d). No code was changed.

Conventions as in 09. Check IDs reuse the procedure letters of 09 §2
(a = RFC 9421, b = DID/card, c = HPKE, d = session, e = registration,
x = boundary/edge). Columns: **Expected** is the verdict a conforming
inspector must give; **Go error** is the text the Go core returns today
(the inspector propagates it verbatim, `insp/http.go:156,218`); **Vector**
names a `sage-spec/vectors/*.json` entry or `none`; **Inspector** cites
where `sage-inspector` implements the check today, `via core` when the
Go core check fires only with `-key`, or `no`; **Go=Rust** says whether
the two cores give the same verdict (error text always differs, see
`rs/error.rs:10-86`). Confidence [High] unless marked.

---

## 1. What a conformance inspector must check

From 09 §2 the inspector has five inputs and one oracle per input:

| Input | Oracle in the Go core | Layers (spec) |
|---|---|---|
| raw HTTP request / response | `rfc9421.HTTPVerifier.VerifyRequest/VerifyResponse` with strict options (`sage/pkg/agent/core/rfc9421/verifier_http.go:189-304`) | 03, 01, 08 §3 |
| agent card JSON | `did.ParseA2AAgentCardWithProof`, `VerifyA2ACardProof[WithDID]` (`sage/pkg/agent/did/a2a_proof.go:64-262`) | 07, 02, 06 §3 |
| HPKE init payload and ack envelope JSON | `hpke.ParseHPKEInitPayloadWithEphCFromJSON`, `Server.validateInitEnvelope`, `Client.parseServerSignedResponse` + `verifySignature` (`sage/pkg/agent/hpke/common.go:255-297`, `server.go:244-277`, `client.go:341-477`) | 04, 02 |
| session record bytes + seed/sid/role | `session.NewSecureSessionFromExporterWithRole`, `DecryptInbound` (`sage/pkg/agent/session/session.go:248-281,908-930`) | 05 |
| registry record (resolved) | `ethereum.selectAgentKeys`, `ResolvePublicKey` (`sage/pkg/agent/did/ethereum/key_policy.go:34-69`, `chainclient.go:39-66`) | 06 §3 |

Requirements a conformance inspector must meet, each with the reason:

1. Every negative case in §2 must produce `fail` with the reason, and the
   process must exit non-zero (`sage-inspector/cmd/sage-inspector/main.go:184-191`
   already does this for `fail` checks).
2. The expected sender identity must be an operator input, not derived
   from the message: today `opts.ExpectedDID = r.DID` where `r.DID` comes
   from the message's own `keyid` (`insp/http.go:149,208,307`), so the
   `ExpectedDID` check can never fail.
3. Replay must be checkable across a capture set, since a single captured
   message cannot show a repeated nonce; today `DisableReplayCheck = true`
   (`insp/http.go:148,207`).
4. Checks that the Go core skips must be done by the inspector itself:
   `created` absent (core skips timing, `verifier_http.go:355,362`; inspector
   flags it, `insp/http.go:289-290`), `X-SAGE-DID` versus `keyid` (core has
   no check; inspector has one, `insp/http.go:313`), the exact `;req`
   set (inspector accepts any `;req`, `insp/http.go:188-197`; the core
   requires three, `verifier_http.go:274-280`).
5. Results must be comparable with a second implementation (the Rust
   core) so that a "pass" is not merely "the Go core agrees with itself";
   the vector runner today compares Go with Go (`insp/vectors.go:95-105`
   requires the vector input to equal the Go generator's fixed input).
6. HPKE, session and registry inputs are not inspectable at all today
   (`sage-inspector/README.md` roadmap; no file under `pkg/inspect`
   references `hpke` or `session`).

---

## 2. Check matrix

### 2a. RFC 9421 request (`sage-inspector request`)

| ID | Case | Input | Expected | Go error / core behaviour | Vector | Inspector | Go=Rust |
|---|---|---|---|---|---|---|---|
| a-p1 | positive | vector request headers + key | pass | nil | `rfc9421/request-{ed25519,es256k,ecdsa-p256-sha256}` | `insp/http.go:125-161` | yes |
| a-n1 | missing `Signature-Input` | header removed | fail | `missing Signature-Input header` (`verifier_http.go:313`) | none | `insp/http.go:227-234` | yes |
| a-n2 | missing `Signature` | header removed | fail | `missing Signature header` (`:322`) | none | `insp/http.go:255-257` | yes |
| a-n3 | malformed `Signature-Input` | unbalanced parens / bad `created` | fail | `failed to parse Signature-Input: ...` (`parser.go:123-233`) | none | `insp/http.go:227-234` | yes |
| a-n4 | `Signature` not a byte sequence | `sig1=abc` (no colons) | fail | `invalid byte sequence format for signature 'sig1'` (`parser.go:96`) | none | via core | yes (`rs/rfc9421/dictionary.rs:80-98`) |
| a-n5 | bad base64 in `Signature` | `sig1=:@@:` | fail | `failed to decode base64 for signature ...` (`parser.go:105`) | none | via core | yes |
| a-n6 | unknown parameter in `Signature-Input` | `;foo="x"` | pass in Go (silently skipped, `parser.go:208-209`); Rust rejects (`rs/rfc9421/components.rs:179-183`) | n/a | none | no | **no** |
| a-n7 | wrong signature bytes | last byte flipped | fail | `ed25519 signature verification failed` / `secp256k1 signature verification failed: ...` (`verifier_http.go:461-506`) | none | via core (`inspect_test.go:86-90` wrong key) | yes |
| a-n8 | wrong signature size | 63-byte Ed25519; 66-byte secp256k1 | fail | Ed25519: `ed25519.Verify` false; secp256k1: `ErrInvalidSignature` (`keys/secp256k1_keccak.go:48-53`) | none | via core | yes (`rs/crypto/signature.rs:59-105`) |
| a-n9 | DER-encoded secp256k1 / P-256 | ASN.1 sequence | fail in Go (`invalid ECDSA signature format` / `ASN.1 parsing not implemented`, `verifier_http.go:632-636`; secp256k1 path 64/65 only); Rust accepts DER | n/a | none | via core | **no** (spec 01 §2 SHOULD accept DER) |
| a-n10 | high-S secp256k1 | `s > N/2` | pass (normalised, `secp256k1_keccak.go:59`; spec 01 §2 MUST) | nil | none | via core | yes (`rs/crypto/keys.rs:397-410`) |
| a-n11 | expired | `created = now - 301` | fail | `signature expired: created 301 seconds ago (max 300)` (`:355-359`) | none (vectors run MaxAge 0) | `insp/http.go:294` own check + via core unless `-ignore-age` | yes |
| a-n12 | future-dated | `created = now + 301` | fail | `signature created in the future: ...` (`:362-370`) | none | `insp/http.go:294` + via core | yes |
| a-n13 | `expires` passed | `expires = now - 1` | fail | `signature expired at %d (now %d)` (`:372-374`) | none | via core only (no own check) | yes |
| a-n14 | `created` absent | no `created` | fail (inspector policy) | core skips timing entirely (`:355,362`) | none | `insp/http.go:289-290` | yes (both cores skip) |
| a-n15 | nonce missing | no `nonce` | fail (strict) | `signature nonce is required but missing` (`:386`) | none | `insp/http.go:299-301` + via core | yes |
| a-n16 | replayed nonce | same nonce, same keyid, twice within 5 min | fail on the second | `signature replay detected: nonce %q was already used for keyid %q` (`:433-439`) | none | **no** (`DisableReplayCheck`, `insp/http.go:148`) | yes |
| a-n17 | required component missing | `@authority` dropped | fail | `required component "@authority" is not covered by the signature` (`:382`) | none | via core | yes |
| a-n18 | body present, `content-digest` not covered | header set but not in list | fail (strict) | `required component "content-digest" ...` (`:207-209`) | none | via core; own digest check does not test coverage (`insp/http.go:263-282`) | yes (Rust only when body passed, `rs/rfc9421/verifier.rs:311-319`) |
| a-n19 | `Content-Digest` missing while covered | header removed | fail | `content-digest header missing while covered by signature` (`body_integrity.go:71-98`) | none | `insp/http.go:276` + via core | yes |
| a-n20 | digest mismatch / tampered body | body edited | fail | `content-digest mismatch: actual=... expected=... (body tampering detected)` | none | `insp/http.go:280` (`inspect_test.go:92-96`) | yes |
| a-n21 | `Content-Digest` with a second algorithm and correct sha-256 | `sha-512=:..:, sha-256=:..:` | pass | `equalDigestHeader` picks the `sha-256=` member (`body_integrity.go:233-248`) | none | `insp/http.go:277` uses `strings.Contains`, pass | yes (`rs/rfc9421/canonicalize.rs:19-33`) |
| a-n22 | tampered covered header | `Date` changed | fail | signature verification failed | none | via core (`integration_test.go:397`) | yes |
| a-n23 | tampered uncovered header | any other header | pass | nil | none | via core (`integration_test.go:424`) | yes |
| a-n24 | `;req` in a request signature | `"@method";req` | fail | `component ...: the req parameter is only valid in response signatures` (`canonicalizer.go:138-141`) | none | via core | yes (`rs/rfc9421/verifier.rs:152-156`) |
| a-n25 | `alg` contradicts key | `alg="ed25519"` with secp256k1 key | fail | `algorithm validation failed: algorithm mismatch: ...` (`algorithm_registry.go:298-323`) | none | via core (key type from `-key` length) | yes |
| a-n26 | `alg` absent | no `alg` | pass in both cores (`algorithm_registry.go:298-300`; `rs/rfc9421/verifier.rs:352-357`); spec 01 §3 silent on absence | nil | none | via core | yes |
| a-n27 | wrong key type supplied | P-256 key given without `p256:` prefix | fail with a misleading reason (parsed as secp256k1, `insp/http.go:80-83`) | `secp256k1 signature verification failed` | none | partial | n/a |
| a-n28 | `keyid` not a DID | `keyid="abc"` | fail | core: only `ExpectedDID` mismatch (`:406-407`); inspector own check | none | `insp/http.go:311-312` | yes |
| a-n29 | `keyid` DID differs from expected sender | `did:sage:ethereum:0xaaaa#key-1` when 0xbbbb expected | fail | `signature keyid %q does not belong to DID %q` | none | **no** (expected DID is taken from the message, `insp/http.go:149`) | yes |
| a-n30 | `X-SAGE-DID` differs from `keyid` DID | header changed (uncovered case) | fail | core: none; gateway `X-SAGE-DID does not match keyid` (`gw/verify/verify.go:101-103`) | none | `insp/http.go:313` | **no** (Rust core rejects, `rs/rfc9421/verifier.rs:159-169`; Go core does not) |
| a-n31 | inactive DID | resolver returns `is_active=false` | fail | `agent is deactivated` (`ethereum/chainclient.go:49-50`) | none | **no** (no resolver) | Rust has no resolver |
| a-n32 | key not verified on chain | only unverified keys | fail | `agent has no verified signing key` (`:52-53`) | none | **no** | Rust n/a |
| a-n33 | authority not served | `Host` differs from configured list | fail | `request authority %q is not served by this verifier` (`:415-428`) | none | **no** (no `-authority` flag) | yes |
| a-n34 | body over 16 MiB | 16 MiB + 1 | fail | `request body exceeds 16777216 bytes` (`body_integrity.go:200-205`) | none | via core | **no** (Rust has no cap) |
| a-n35 | two signatures, label unspecified | `sig1`, `sig0` | verify `sig0` (first lexicographic) | `verifier_http.go:329-339` | none | `insp/http.go:239-245` | yes (gateway differs, `gw/verify/verify.go:92-96`) |
| a-n36 | label requested but absent | `-label sigX` | fail | `signature 'sigX' not found in Signature-Input` | none | `insp/http.go:246-249` | yes |

### 2a'. RFC 9421 response (`sage-inspector response`)

| ID | Case | Input | Expected | Go error / core behaviour | Vector | Inspector | Go=Rust |
|---|---|---|---|---|---|---|---|
| r-p1 | positive | vector response + request + key B | pass | nil | `rfc9421/response-ed25519` | `insp/http.go:165-224` (`inspect_test.go:99-130`) | yes (Rust verifies its own headers only, `rs-sage-core/tests/spec_vectors.rs:324-377`) |
| r-n1 | no `;req` component | response signed over `@status`, `content-type` only | fail | `required component "@method";req ...` (`verifier_http.go:274-280`) | none | `insp/http.go:196` own check + via core | **no** (Rust: any `;req` suffices, `rs/rfc9421/verifier.rs:217-222`) |
| r-n2 | only one `;req` component | `"@method";req` present, others absent | fail (strict) | `required component "@target-uri";req ...` | none | own check passes (`insp/http.go:188-197`); fails only via core with `-key` | **no** |
| r-n3 | `@status` not covered | dropped | fail | `required component "@status" ...` | none | via core | yes |
| r-n4 | wrong request supplied | different `@target-uri` | fail | signature verification failed | none | via core (`-request`) | yes |
| r-n5 | request digest changed after signing | `content-digest;req` mismatch | fail | signature verification failed | none | via core | yes |
| r-n6 | tampered status | 200 -> 500 | fail | signature verification failed (`verifier_http_response_test.go:72`) | none | via core | yes |
| r-n7 | response body tampered | body edited | fail | `content-digest mismatch` (`body_integrity.go:128-156`) | none | `insp/http.go:198` | yes |
| r-n8 | no request given | `-request` omitted | fail | n/a | none | `main.go:115-117`, `insp/http.go:176-178` | n/a |
| r-n9 | `ExpectedDID` for the responder | wrong responder DID in keyid | fail | `signature keyid %q does not belong to DID %q` | none | **no** (tautological, `insp/http.go:208`) | yes |

### 2b. DID, resolution, PoP, A2A card (`sage-inspector card`)

| ID | Case | Input | Expected | Go error | Vector | Inspector | Go=Rust |
|---|---|---|---|---|---|---|---|
| b-p1 | DID grammar accepted | 5 valid forms | pass | nil | `did/parse` | `insp/vectors.go` (vector run only) | yes |
| b-n1 | DID rejected | 7 invalid forms | fail | `invalid DID format`, `unknown chain: ...`, `... empty identifier` (`manager.go:327-341`) | `did/parse.rejected` | vector run only | yes |
| b-n2 | chain alias / case / whitespace | `ETH`, ` Solana ` | pass | nil | `did/chain-aliases` | vector run only | yes |
| b-p2 | PoP Ed25519 / secp256k1 | vector `key_data`, `proof` | pass | nil | `did/pop-*` | vector run only; no `pop` command | yes |
| b-n3 | PoP wrong DID in challenge | other DID | fail | `Ed25519 PoP verification failed` / `ECDSA PoP verification failed` (`key_proof.go:120,153`) | none | **no** | yes |
| b-n4 | PoP high-S secp256k1 | `s > N/2` | Go fail (no normalisation before `ethcrypto.VerifySignature`, `key_proof.go:147-153`); Rust pass (`rs/did/proof.rs:53-86` normalises) | `ECDSA PoP verification failed` | none | **no** | **no** [Mid: relies on go-ethereum rejecting malleable `s`] |
| b-n5 | PoP on X25519 key | type 2 | skip (no proof) | nil (`key_proof.go:194-196`) | none | **no** | yes |
| b-p3 | card positive | vector `card_json` | pass | nil | `did/a2a-card-proof-ed25519` | `insp/card.go:20-53` | yes |
| b-n6 | card tampered field | `name` changed | fail | `proof verification returned false` (`a2a_proof.go:361-377`) | none (Rust test mutates locally, `spec_vectors.rs:703-721`) | `insp/card.go:42-47` (`inspect_test.go:146-149`) | yes |
| b-n7 | `proof` absent | member removed | fail | `card has no proof` | none | `insp/card.go:37-40` | yes |
| b-n8 | `verificationMethod` outside `<id>#` | other DID | fail | `verification method %s does not belong to DID %s` (`a2a_proof.go:231`) — only in `WithDID`; self-attested path returns `verification key not found in card` (`:197`) | none | via `VerifyA2ACardProof` (weak path) | yes (Rust `validate`, `rs/did/a2a.rs:221-259`) |
| b-n9 | `publicKeyHex` != `publicKeyBase58` | hex altered | Go pass (base58 wins, `a2a_proof.go:276-293`); spec 07 §1 MAY reject; Rust fail | nil | none | **no** | **no** |
| b-n10 | key `type` != proof `type` | Ed25519 key entry, `EcdsaSecp256k1Signature2019` proof | Go: decided by proof type only (`a2a_proof.go:297-350`), fails on key length; spec 07 §3.3 MUST match | `invalid public key length` | none | **no** explicit check | **no** (Rust checks, `rs/did/a2a.rs:263-296`) |
| b-n11 | secp256k1 card key 64 bytes (`x||y`) | no `04` prefix | Go pass (`a2a_proof.go:315-319`); Rust fail (`rs/crypto/keys.rs:139-151`) | nil | none | via core | **no** |
| b-n12 | card key unverified on chain | resolver says `verified=false` | fail | `verification key %s is not a verified key of %s on-chain` (`:259`) | none | **no** (no resolver, README) | Rust n/a |
| b-n13 | card DID inactive on chain | `is_active=false` | fail | `DID %s is not active on-chain` (`:248`) | none | **no** | Rust n/a |
| b-n14 | basic shape errors | no keys / no service / missing controller | fail | `at least one public key is required` etc. (`a2a.go:153-201`) | none | `insp/card.go:32-36` | yes |
| b-n15 | malformed JSON | `{` | fail | `invalid A2A card JSON: ...` | none | `insp/card.go:22-26` (`inspect_test.go:150-152`) | yes |
| b-n16 | duplicate JSON member in card | two `name` members | Go: last wins in `encoding/json`, JCS built from decoded map (`a2a_proof.go:77-89`) [Mid]; Rust JCS rejects (`rs/jcs/mod.rs:247-249`) | n/a | none | **no** | **no** [Mid] |

### 2c. HPKE handshake (no inspector command today)

| ID | Case | Input | Expected | Go error (server side unless noted) | Vector | Inspector | Go=Rust |
|---|---|---|---|---|---|---|---|
| c-p1 | info / exportCtx strings | ctx, DIDs | pass | nil | `hpke/info-and-export-context` | vector run only | yes |
| c-p2 | combiner, traffic keys, ack tag, export round trip | vector inputs | pass | nil | `hpke/{combine-secrets,traffic-keys,ack-tag,hpke-export-roundtrip}` | vector run only | yes |
| c-n1 | init member missing | no `ephC` | fail | `parse payload: missing ephC: ...` (`common.go:164,255-297`) | none | **no** | yes |
| c-n2 | `enc`/`ephC` wrong size | 31 B | fail | `bad enc length: 31` / `bad ephC length: 31` (`common.go:292-297`) | none | **no** | yes (`rs/hpke/types.rs:184-204`) |
| c-n3 | wrong base64 alphabet | padded / std alphabet | fail | decode error from `base64.RawURLEncoding` (`common.go:174`) | none | **no** | yes |
| c-n4 | `ts` not RFC 3339 | `1788609600` | fail | `bad ts: ...` | none | **no** | yes |
| c-n5 | `ts` outside ±2 min | now - 121 s | fail | `authentication failed` (logged `timestamp out of window`, `server.go:257-260`) | none | **no** | yes (`rs/hpke/server.rs:144-151`) |
| c-n6 | replayed init (same ctx + nonce within 10 min) | resend | fail | `authentication failed` (`replay detected`, `:261-263`) | none | **no** | yes |
| c-n7 | same nonce, different ctx | resend with new ctx | pass (scope is ctx) | nil | none | **no** | yes |
| c-n8 | `info` / `exportCtx` mismatch | ctx swapped | fail | `authentication failed` (`info mismatch` / `exportCtx mismatch`, `:264-271`) | none | **no** | yes |
| c-n9 | `initDid` != signing DID | forged | fail | `authentication failed` (`:249-251`) | none | **no** | yes (Rust trusts `sender_did` argument, `rs/hpke/server.rs:113-118`) |
| c-n10 | `respDid` != server | other DID | fail | `authentication failed` (`:254-256`) | none | **no** | yes |
| c-n11 | transport `signature` invalid | bit flipped | fail | `signature verification failed: ...` (`:222-241`) | none | **no** | **no** (Rust does not verify it) |
| c-n12 | wrong task | `hpke/complete@v2` | fail | `unsupported task: hpke/complete@v2` (`:142-144`) | none | **no** | yes |
| c-n13 | suite not allowed | whitelist without suite | fail | `suite not allowed` (`:273-275`) | none | **no** | yes |
| c-n14 | all-zero DH | crafted `ephC` = low-order point | fail | `invalid ECDH (all-zero)` (`:313`) | none | **no** | yes |
| c-n15 | cookie required but absent | no `metadata.cookie` | fail | `cookie required or invalid` (`:159-164`) | none | **no** | location differs (Rust reads `payload.cookie`, `rs/hpke/types.rs:135`) |
| c-a1 | ack `v` / `task` wrong (client) | `v2` | fail | `unsupported version/task: v2/...` (`client.go:437`) | none | **no** | yes |
| c-a2 | ack `ctx` differs | other ctx, valid sig/tag over it | Go pass (no explicit equality, tag uses `r.Ctx`, `client.go:158-164`); Rust fail (`rs/hpke/client.rs:150-209`) | nil | none | **no** | **no** [Mid] |
| c-a3 | ack tag wrong | byte flipped | fail | `ack tag mismatch` (`:501`) | none | **no** | yes |
| c-a4 | `infoHash` / `exportCtxHash` wrong | altered | fail | `pre-ack mismatch: info/exportCtx` (`:155`) | none | **no** | yes |
| c-a5 | `enc` / `ephC` echo differs | altered | fail | `enc mismatch` / `ephC mismatch` (`:166-178`) | none | **no** | yes |
| c-a6 | `ephS` all-zero | zeros | fail | `invalid ECDH (all-zero)` (`:492`) | none | **no** | yes |
| c-a7 | `sigB64` wrong / wrong signer | other key | fail | `server signature verify failed: ...` (`:475`) | none | **no** | yes |
| c-a8 | `did` != responder | other DID | fail | `response did %q does not match server DID %q` (`:465`) | none | **no** | yes |
| c-a9 | JCS of envelope: non-canonical spacing / key order from responder | reordered JSON | pass (verifier re-canonicalises, `client.go:388-394`; spec 02 §2) | nil | none (envelope vector missing) | **no** | yes |

### 2d. Session records (no inspector command today)

| ID | Case | Input | Expected | Go error | Vector | Inspector | Go=Rust |
|---|---|---|---|---|---|---|---|
| d-p1 | seed, sid, records at seq 0 and 256, directional + AAD | vector | pass | nil | `session/{seed-and-id,encrypt-decrypt,directional-with-aad}` | vector run only | yes |
| d-n1 | replayed record | `seq0` twice | fail | `session: replayed message` (`session.go:139,174`) | `encrypt-decrypt` verify step (`pkg/vectors/session.go:116-119`) | vector run only | yes |
| d-n2 | stale record | `seq = highest - 1024` | fail | `session: message outside replay window` (`:141,170`) | none | **no** | yes (`rs/session/secure_session.rs:57-72`) |
| d-n3 | out of order inside window | `seq 5` after `seq 9` | pass | nil (`session_replay_test.go:62`) | none | **no** | yes |
| d-n4 | record shorter than header | 19 B | fail | `session: data too short` (`:143,911`) | none | **no** | verdict yes; Rust min 36 B (`secure_session.rs:350-379`), so 20..35 B fail with different reasons |
| d-n5 | tampered `seq` header | seq byte flipped | fail (AAD) | `decryption failed: ...` (`:924`) | none | **no** | yes |
| d-n6 | tampered ciphertext | body byte flipped | fail; window unchanged | `decryption failed` (`session_replay_test.go:109`) | none | **no** | yes |
| d-n7 | rekey boundary | seq 255 (gen 0) then 256 (gen 1), and 256 before 255 | pass both orders | nil (`:847-879`; `session_replay_test.go:129`) | `encrypt-decrypt` covers 0 and 256 in order only | vector run only | yes |
| d-n8 | wrong direction key | c2s record decrypted as c2s by initiator | fail | `decryption failed` | `directional-with-aad` (positive only) | vector run only | yes |
| d-n9 | wrong AAD | other caller AAD | fail | `decryption failed` (`session_replay_test.go:189`) | none | **no** | yes |
| d-n10 | message limit | 1001st record | fail | `session expired` (`:502,647`) | none | **no** | yes (1000 accepted; Rust text `Message limit exceeded`) |
| d-n11 | seq wrap | `sendSeq = 2^64 - 1` then next | Go: silent wrap to 0 (`:887-888`, no check); nonce is random so AEAD nonce reuse does not follow, but replay window resets | n/a | none | **no** | not verified for Rust [Low] |

### 2e. Registration (resolver-level checks only)

| ID | Case | Input | Expected | Go error | Vector | Inspector | Go=Rust |
|---|---|---|---|---|---|---|---|
| e-n1 | DID after commit only | resolve | fail | `DID not found in registry` (`agentcard_client.go:378-380`) | none | **no** | Rust n/a |
| e-n2 | registered, not activated | resolve for signing | fail | `agent is deactivated` (`chainclient.go:49-50`) | none | **no** | Rust n/a |
| e-n3 | active, only unverified keys | resolve | fail | `agent has no verified signing key` | none | **no** | Rust n/a |
| e-n4 | active, verified ECDSA and Ed25519 | resolve | ECDSA chosen (`key_policy.go:41-58`) | nil | none | **no** | Rust n/a |
| e-n5 | KEM key missing | resolve KEM | fail | `agent does not have KME key registered` (`agentcard_client.go:506-552`) / `ErrInactiveAgent` | none | **no** | Rust n/a |

### 2x. Boundary and edge values

| ID | Case | Input | Expected | Go behaviour | Vector | Inspector | Go=Rust |
|---|---|---|---|---|---|---|---|
| x-1 | `created = now - 300` exactly | age == MaxAge | pass (`age > MaxAge` strict, `verifier_http.go:357`) | nil | none | own check uses `age > 300` (`insp/http.go:294`), consistent | yes (`rs/rfc9421/verifier.rs:337` uses `>`) |
| x-2 | `created = now + 300` exactly | skew edge | pass | nil (`> now + skew`) | none | consistent | yes |
| x-3 | HPKE `ts = now - 120 s` exactly | skew edge | fail in Go (`Before(now - maxSkew)` is false at equality, so pass) [Mid: nanosecond `ts` makes exact equality unreachable in practice] | nil | none | no | yes |
| x-4 | empty body with `Content-Digest` of empty string | `sha-256=:47DEQpj8...=:` | pass; digest not required (`requestHasBody` false, `:453-458`) | nil | none | `insp/http.go:266-271` reports info | Rust: digest only when body passed |
| x-5 | empty body, `content-digest` covered but header absent | covered list includes it | fail | `content-digest header missing while covered` | none | via core | yes |
| x-6 | duplicate `Date` headers | two `Date` lines | signature base joins `", "` (`canonicalizer.go:268-276`); signer/verifier consistent | nil | none | `insp/http.go` uses `h.Get` (first only) for `Content-Digest`, `X-SAGE-DID` | yes (`rs/rfc9421/canonicalize.rs:124-139`) |
| x-7 | duplicate `Signature-Input` headers | two headers | Go `Header.Get` takes the first; second ignored | n/a | none | same (`insp/http.go:227`) | not verified for Rust [Low] |
| x-8 | `@query` empty | `/path?` vs `/path` | `?` in both (`canonicalizer.go:249-254`) | nil | none | via core | yes |
| x-9 | `@query-param` percent-encoded name/value | `a%20b=1` | Go decodes via `url.Query()` (`canonicalizer.go:281-296`); Rust matches raw text (`rs/rfc9421/canonicalize.rs:75-86`) | n/a | none | via core | **no** [Mid] |
| x-10 | JCS UTF-16 sort, escapes, numbers | RFC 8785 samples | pass | nil | `jcs/{rfc8785-appendix-a,rfc8785-unicode-sorting,numbers}` | vector run only | yes |
| x-11 | JCS integer > 2^53 | `9007199254740993` | `9007199254740992` (f64) in both (`jcs.go:85`; `rs/jcs/mod.rs:424-443`) | n/a | none | no | yes |
| x-12 | JCS lone surrogate / raw control char / duplicate key | `"\ud800"` | Rust rejects (`rs/jcs/mod.rs:164-186,247-249`); Go decodes with `encoding/json` + `UseNumber` (`jcs.go:58-59`), which keeps the last duplicate and substitutes U+FFFD for a lone surrogate [Mid] | n/a | none | no | **no** [Mid] |
| x-13 | secp256k1 public key with leading zero X byte | crafted key | 65-byte SEC1 keeps the zero; 64-byte on-chain form too (`utils.go:46-55`) | nil | none | `insp/http.go:78-81` accepts 33/65 | yes |
| x-14 | low-S boundary `s = N/2` | crafted | accepted (`LowS` flips only `s > N/2`, `keys/ecdsa_encoding.go:85-90`) | nil | none | via core | yes (`normalize_s`) |
| x-15 | `s = 0` or `r = 0` | crafted | fail (curve library) | verification failed | none | via core | yes |
| x-16 | HPKE `ctxID` containing `|` | crafted | Go scope `(ctx, nonce)` pair; Rust key `"{ctx}|{nonce}"` (`rs/hpke/server.rs:154`) can collide | n/a | none | no | **no** (edge) |
| x-17 | transport body 1 MiB + 1 | HTTP transport | `413 request body too large` (`transport/http/server.go:63,102-108`) before any signature work | n/a | none | no | Rust n/a |

---

## 3. Prioritised gaps

Effort: S under one day, M one to three days, L more than three days.
Priority: P1 = a verdict today can be wrong or is impossible; P2 = a spec
MUST is unchecked; P3 = completeness.

### Missing inspector checks

| Pri | Gap | IDs | Change | Effort |
|---|---|---|---|---|
| P1 | Expected DID is derived from the message; add `-did` (and `-authority`) inputs and pass them as `ExpectedDID` / `ExpectedAuthorities` | a-n29, a-n33, r-n9 | `insp/http.go:149,208` | S |
| P1 | Replay across a capture set: accept several `-f` files, run one `HTTPVerifier` with the replay guard enabled, report the second occurrence | a-n16, c-n6 | new option in `insp/http.go:146-160` | M |
| P1 | Response binding: check the exact `;req` set without `-key`, not "any `;req`" | r-n1, r-n2 | `insp/http.go:187-197` | S |
| P1 | HPKE inspection: `init -f payload.json -ctx -sender-did -server-did` (member set, sizes, base64url alphabet, `ts` window, info/exportCtx recomputation) and `ack -f envelope.json -key` (JCS re-canonicalisation, `sigB64`, hashes, `ephS` non-zero, echo) | c-n1..c-n15, c-a1..c-a9 | new `insp/hpke.go` calling `hpke.ParseHPKEInitPayloadWithEphCFromJSON`, `hpke.DefaultInfo`, `jcs.Marshal`; ack tag needs the seed, so report `skip` for the tag unless `-seed` is given | M |
| P1 | Session record inspection: `record -f rec.bin -seed -sid -role -rekey` decrypting with `DecryptInbound`, reporting seq, generation, replay/stale against a supplied window state | d-n2..d-n11 | new `insp/session.go` | M |
| P2 | `expires` own check and `created`-absent policy documented; clock override `-now` exposed on the CLI (exists in `MessageOptions.Now`, `insp/http.go:60`, not wired in `main.go`) | a-n13, a-n14, x-1, x-2 | `insp/http.go:284-304`, `main.go:90-97` | S |
| P2 | Content-Digest coverage: report whether `content-digest` is in the covered list, and use `equalDigestHeader` semantics instead of `strings.Contains` | a-n18, a-n21 | `insp/http.go:263-282` | S |
| P2 | Card: hex/base58 consistency, key `type` versus proof `type`, `verificationMethod` prefix, duplicate members | b-n9, b-n10, b-n8, b-n16 | `insp/card.go:20-53` (checks the Go core does not make) | S |
| P2 | Card and request on-chain cross-check `-network sepolia` via `did.Manager` + `VerifyA2ACardProofWithDID` / `ResolvePublicKey` | a-n31, a-n32, b-n12, b-n13, e-n1..e-n5 | new resolver wiring as in `gw/resolve/resolve.go:173-199` | M |
| P2 | Explicit `-key-type` instead of length inference | a-n27 | `insp/http.go:65-86` | S |
| P2 | `pop` command (`-did -key-type -key -proof`) and `jcs` command (canonicalise stdin, print hex and SHA-256) | b-n3..b-n5, x-10..x-12 | new files, `did.VerifyKeyProofOfPossession`, `jcs.Canonicalize` | S |
| P3 | Duplicate header detection (`Signature-Input`, `Signature`, `Content-Digest`, `X-SAGE-DID`) reported as `info` | x-6, x-7 | `insp/http.go:226-261` | S |
| P3 | Built-in mutation suite: for every deterministic vector apply the negative mutations of §2 and assert `fail`, so a "26 passed" run also proves rejection | all a-n, b-n, d-n | `insp/vectors.go` | M |
| P3 | Cross-implementation mode: run the same inputs through `rs-sage-core` (FFI header `include/sage_crypto.h` or WASM) and diff verdicts | all `Go=Rust` = no | new binary or CI job | L |

### Missing spec vectors (sage-spec)

| Pri | Suite | Vector to add | Mode | Covers | Effort |
|---|---|---|---|---|---|
| P1 | rfc9421 | `request-rejected`: list of mutated requests (expired, future, missing nonce, missing component, `;req` in request, digest mismatch, alg mismatch, wrong DID, bad base64, high-S secp256k1 accepted, DER) with expected verdict per entry, like `did/parse.rejected` | deterministic | a-n3..a-n26 | M (needs a `rejected` schema in `pkg/vectors/vectors.go:199-227`) |
| P1 | hpke | `init-payload` (JSON bytes for fixed ctx/DIDs/labels, `enc` from a fixed KEM scalar and fixed HPKE randomness is not reproducible, so verify mode) and `response-envelope` (JCS bytes, `sigB64`, verify mode) | verify | c-n1..c-n10, c-a1..c-a9 | M |
| P1 | hpke | `init-rejected`: ts out of window, replay, info mismatch, wrong sizes, wrong alphabet | deterministic | c-n2..c-n8 | S |
| P2 | session | `stale-and-order`: records at seq 0, 1, 1025 then 0 again (stale), 1024 before 1025 (in window), 255/256 both orders, 19-byte record | verify | d-n2, d-n3, d-n4, d-n7 | S |
| P2 | did | `pop-secp256k1-high-s` (must fail in Go today; decides b-n4), `a2a-card-proof-secp256k1`, `a2a-card-rejected` (hex/base58 mismatch, type mismatch, foreign `verificationMethod`) | mixed | b-n4, b-n9, b-n10 | S |
| P2 | jcs | `edge-cases`: lone surrogate, raw control char, duplicate key, `-0`, `1e400`, `9007199254740993`, deeply nested object | deterministic with `rejected` | x-11, x-12 | S |
| P3 | crypto | `secp256k1-leading-zero` (key whose X starts with `00`), `low-s-boundary` (`s = N/2`), `der-encoded` (verify-only, documents the Go HTTP path rejection) | deterministic | x-13, x-14, a-n9 | S |
| P3 | rfc9421 | `request-empty-body`, `request-duplicate-headers`, `request-query-param-encoded` | deterministic | x-4, x-6, x-9 | S |

---

## Facts vs Opinions

Facts (verified in the cited files):

- `sage-inspector` disables the replay check and sets `ExpectedDID` from
  the message's own `keyid` (`insp/http.go:148-149,207-208`).
- Its request-binding check passes on any `;req` component
  (`insp/http.go:187-197`); the Go core requires `@method`, `@target-uri`,
  `@authority` with `;req` (`verifier_http.go:274-280`).
- Its digest check uses `strings.Contains` and does not test coverage
  (`insp/http.go:263-282`); its timing check is ±300 s on `created` and
  ignores `expires` (`insp/http.go:284-304`).
- `card` verifies self-attested proofs only (`insp/card.go:42`); no
  resolver is wired (README "What is and is not checked").
- No HPKE or session input can be inspected; `vectors` compares the Go
  core against vectors whose inputs must equal the Go generator's
  (`insp/vectors.go:95`).
- The only rejection list in the vectors is `did/parse.rejected`
  (`pkg/vectors/did.go:21,43-50`); the only in-vector negative assertion
  is the replay in `session/encrypt-decrypt` (`pkg/vectors/session.go:116-119`).
- Go and Rust give different verdicts for: unknown `Signature-Input`
  parameters, DER signatures, `X-SAGE-DID` mismatch (core), body size cap,
  weak `;req` binding, high-S PoP, card hex/base58 and type mismatches,
  64-byte card keys, ack `ctx` equality, `@query-param` decoding.

Opinions:

- [High] The three P1 inspector items (`-did` input, replay over a
  capture set, exact `;req` set) are each under a day and turn the
  current tool from "explains" into "judges"; without them a pass is not
  evidence of conformance for a-n16, a-n29, r-n1.
- [High] A `rejected`-style schema for rfc9421 and hpke vectors is the
  cheapest way to pin the Go=Rust "no" rows, because both cores already
  load the vector files in CI.
- [Mid] The high-S PoP divergence (b-n4) should be settled in the spec
  text (06 §4 does not say whether verifiers normalise); the Keccak path
  says MUST normalise (01 §2), so aligning Go's `key_proof.go:153` with
  `VerifySecp256k1Keccak`'s normalisation is the smaller change.
- [Mid] The cross-implementation runner (L) is worth deferring until the
  vector `rejected` lists exist; most divergences are then caught in
  each core's own CI.
- [Low] Sequence wrap (d-n11) and `ctxID` containing `|` (x-16) are
  theoretical at current message rates; document rather than test.
