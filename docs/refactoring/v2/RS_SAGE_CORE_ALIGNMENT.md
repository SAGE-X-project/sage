# rs-sage-core alignment plan (F-03)

Survey date 2026-09-12, crate `sage_crypto_core` 0.3.0 at `c86da99`.
Target: every suite in `sage-spec/vectors` passes in the Rust core's CI,
and the crate exposes the C header and WASM artifacts the SDKs need.

## 1. Baseline (done in rs-sage-core PR #18)

- `handshake` (1 182 lines), `transport` (1 315) and `blockchain` (4 926)
  removed with their examples, benches and tests; `reqwest`, `alloy*`,
  `solana-*` gone, and with them six of the seven `cargo audit` findings and
  all three `cargo deny` advisories except `rsa`.
- Toolchain pinned to Rust 1.88.0; CI runs one stable toolchain; `cargo fmt`
  and `clippy -D warnings` clean on all targets and features; the
  `wasm32-unknown-unknown` build fixed (`uuid` `js` feature).
- 589 unit tests plus integration and doc tests pass.

## 1a. Progress (2026-09-12)

| Step | State | Where |
|---|---|---|
| 1 Baseline | merged | rs-sage-core #18 |
| 2 Vector harness + JCS | in review | rs-sage-core #23 (consolidated; the stacked #19-#22 were closed after #18 was squash-merged) |
| 3 Crypto | in review | #23 |
| 4 RFC 9421 | in review | #23 |
| 5 Session | in review | #23 |
| 6 HPKE | committed locally, PR after #23 | branch `feat/hpke-align-2` |
| 7 did:sage and A2A | committed locally, PR after #23 | branch `feat/hpke-align-2` |
| 8 C header / WASM surface | open | |

With steps 2-7 the vector harness passes all 26 sage-spec vectors
(`jcs` 4, `crypto` 4, `rfc9421` 4, `hpke` 6, `session` 3, `did` 5), byte
for byte against the Go core where the vectors are deterministic. Two
specification corrections came out of the work: the Ethereum address is
lower-case hex, not EIP-55 (sage-spec #2), and the HPKE init payload
carries `info` and `exportCtx` as plain strings (sage-spec #3).

## 2. Divergences from sage-spec, by module

| Area | Today | Required (spec section) | Effort |
|---|---|---|---|
| secp256k1 signing | SHA-256 digest, 64-byte `r‖s`, no low-S, compressed 33-byte public keys | Keccak-256, RFC 6979, low-S, 65-byte `r‖s‖v`, uncompressed 65-byte keys (01 §2) | medium |
| RFC 9421 `alg` | `ecdsa-secp256k1-sha256`, `rsa-v1_5-sha256`, `rsa-pss-sha512` | `es256k`, `rsa-pss-sha256` (01 §3) | low |
| `Signature` header | `sig1=:<b64>` without the closing `:` | RFC 8941 byte sequence `sig1=:<b64>:` (03 §1) | low |
| `Content-Digest` | absent | `sha-256=:<b64>:`, covered when a body exists, 16 MiB cap (03 §1, §5) | medium |
| `nonce` | never emitted, never parsed | required by strict verification, per-keyid replay guard, `MaxAge` window (03 §5) | medium |
| `keyid` | local key id (`hex(SHA-256(pub)[0:8])`) | `<DID>` or `<DID>#fragment`, `ExpectedDID`, `X-SAGE-DID` consistency (03 §3) | medium |
| Response signing | `@status` + `content-type` only | `;req` binding of method/target/authority/digest/signature (03 §4) | medium |
| Labels, components | only `sig1`; no `@query-param`; two inconsistent base builders | any label (first lexicographically by default), all derived components (03 §2) | low |
| JCS | absent | RFC 8785 for the HPKE envelope and A2A proof (02) | medium-high |
| HPKE KEM | hand-rolled X25519 + HKDF; no DHKEM labels | RFC 9180 base mode, `Export(exportCtx, 32)` (04 §1, §3) | high |
| HPKE traffic keys | RFC 5869 HKDF-Expand | HMAC counter expansion `HMAC(seed, label ‖ be32(i))` (04 §4) | low |
| HPKE init payload | no `initDid`, `respDid`, `ts`; byte fields as JSON arrays | all members, base64url-raw bytes, ±2 min `ts`, per-`ctxID` 10 min replay (04 §6) | medium |
| HPKE response envelope | no `sigB64`; hashes hex | detached JCS signature; hashes base64url-raw (04 §6) | medium |
| Session AEAD and record | AES-256-GCM, `nonce ‖ ct`, derived nonce | ChaCha20-Poly1305, `be64(seq) ‖ nonce[12] ‖ ct`, random nonce, seq as AAD (05 §3) | high |
| Session key schedule | keys straight from the HPKE traffic keys | seed/id derivation, `sage-session-keys-v1`, `sage-directional-keys-v1`, `sage-session-rekey-v1` (05 §1, §2, §5) | high |
| Replay window, rekey | none | 1024-slot bitmap, generation = seq / R (05 §4, §5) | medium |
| `did:sage` | `did:sage:key:` / `did:sage:chain:` | `did:sage:<ethereum|solana>:<identifier>`, aliases, rejection list (06 §1, §2) | medium |
| Key proof of possession | absent | `SAGE-PoP:` challenge, Ed25519 and secp256k1 (06 §4) | low |
| A2A card proof | absent | JCS form without `proof`, base58 `proofValue` (07) | medium |
| Test vectors | none | load `sage-spec/vectors/*.json` in CI, deterministic and verify modes | medium |
| C header | `build.rs` prints a warning; no `include/`, no cbindgen | generated header as a CI artifact | low |
| WASM surface | message-level sign/verify only | request/response signing and verification, HPKE, session | medium |

Sources: `src/rfc9421/{signer,verifier,canonicalize,components,mod}.rs`,
`src/crypto/keys.rs`, `src/hpke/{common,client,server,types}.rs`,
`src/session/secure_session.rs`, `src/did/mod.rs`, `src/ffi/*.rs`,
`src/wasm/*.rs`, `Cargo.toml`.

## 3. Order of work

Each step is one pull request, keeps CI green, and ends with the
corresponding sage-spec suite passing in a vector harness added in step 2.

1. **Baseline** (done): prune, pin, format, lint, WASM build.
2. **Vector harness**: a `tests/spec_vectors.rs` that loads
   `sage-spec/vectors/*.json` (checked out by CI at main) and dispatches by
   suite and vector name; suites not yet implemented are skipped explicitly
   and the skip list shrinks with every following step. Adds the
   `jcs` suite first because it has no dependencies: implement RFC 8785
   (UTF-16 key order, ECMAScript escapes and number formatting).
3. **Crypto**: Keccak-256 secp256k1 signing with low-S and `r‖s‖v`,
   uncompressed public keys, key ids over those bytes, P-256 raw low-S,
   remove RSA (closes RUSTSEC-2023-0071), `es256k` identifier. Passes the
   `crypto` suite.
4. **RFC 9421**: Content-Digest, byte-sequence `Signature`, nonce and replay
   guard, `MaxAge`, DID-shaped keyid and `ExpectedDID`, `;req` response
   binding, any label, `@query-param`, one signature-base builder. Passes
   the `rfc9421` suite (the P-256 request vector is verify-only).
5. **Session**: ChaCha20-Poly1305 records with the sequence header, random
   nonces, seq as AAD, the HKDF key schedule with the three labels, replay
   bitmap and generation rekey; seed and id derivation. Passes the
   `session` suite.
6. **HPKE**: DHKEM(X25519, HKDF-SHA256) export through the `hpke` crate,
   HMAC counter expansion for traffic keys, full init payload and signed
   response envelope over JCS, per-context replay window. Passes the
   `hpke` suite.
7. **did:sage and A2A**: grammar and aliases, key proof of possession, agent
   card proof; drop the DID-Document-based KEM lookup in favour of the
   registry record shape. Passes the `did` suite.
8. **Surface**: cbindgen header in CI, WASM bindings for request/response
   signing and verification, HPKE and session; then decide the SDK
   repositories (F-04).

Dependency bumps (`k256` 0.11 to 0.13, `signature` 1 to 2, `http` 0.2 to 1,
`base64` 0.21 to 0.22, `pem` 1 to 3) happen in the step that touches the
module using them.

## 4. Trade-offs

- Steps 5 and 6 replace the session and HPKE layers wholesale; the
  existing 20 + 22 unit tests test the old behaviour and are rewritten,
  not kept. Interoperability with anything built on 0.3.0 is not preserved,
  which the README banner already states.
- The `hpke` crate (RFC 9180) adds a dependency; hand-rolling DHKEM to avoid
  it would repeat the mistake that caused the divergence.
- Removing RSA is a functional loss for any 0.3.0 user of RSA keys; the Go
  core still verifies RSA, so mixed deployments keep working on the Go side.
