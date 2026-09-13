# 14. Registry coupling: what is tied to Ethereum and what decoupling costs

Status: review document, 2026-09-13. No code was changed. Sources (commit at survey time):
`sage` dfd3b08 (Go core), `rs-sage-core` 206bbbb, `sage-contracts` d9f313b, `sage-gateway`
4e72668, `sage-spec` a34dd49.

Conventions. Unprefixed paths are in the Go core `sage/`, with `did/` = `pkg/agent/did/`,
`eth/` = `pkg/agent/did/ethereum/`, `sol/` = `pkg/agent/did/solana/`, `crypto/`, `chain/`,
`keys/`, `core/`, `rfc9421/`, `hpke/`, `handshake/` likewise under `pkg/agent/`. Bare `*.sol`
files are in `sage-contracts/ethereum/contracts/`. `rs/` = `rs-sage-core/`, `gw/` =
`sage-gateway/pkg/gateway/`. `spec NN §k` cites `sage-spec/spec/NN-*.md`. [High] read directly
in the cited file; [Mid] inferred by joining two cited places; [Low] not verified by execution.
Plain statements with a citation are facts; judgements carry a label.

The specification wants to describe the agent registry without tying it to one blockchain,
because chains differ in cryptography and some deployments will want no chain at all. This
reports what is coupled today and what decoupling costs.

---

## 1. Where a chain, address format, key type or algorithm is decided

Not in one place. Six layers decide, and they disagree with each other in ways that matter for
a specification.

### DID grammar and parser

One parser per core; both hard-code the method name and chain vocabulary.

| What | Where | Hard-coded |
|---|---|---|
| Go parser | `did/manager.go:327-341` | literal `"did"`/`"sage"`; split on `:`; identifier is everything after the third colon, never checked against an address shape |
| Go length/prefix | `did/did.go:26-37` | `len < 10`, `did[:4] != "did:"` |
| Go chain aliases | `did/manager.go:313-322` | `ethereum`/`eth`, `solana`/`sol`; anything else errors |
| Go chain enum | `did/types.go:84-87` | `"ethereum"`, `"solana"` |
| Rust parser | `rs/src/did/mod.rs:68-84` | same shape: `len < 10`, `starts_with("did:")`, `splitn(4, ':')`, literal `"did"`/`"sage"` |
| Rust chain aliases and names | `rs/src/did/mod.rs:26-43`, `:53-59` | same two chains and aliases |

The identifier is opaque to both parsers: the published vector accepts
`did:sage:ethereum:agent-one` (`sage-spec/vectors/did.json`, `parse`). So the grammar already
permits a non-address identifier — nothing ties `did:sage:ethereum:` to an Ethereum account.
[High]

Two normalisation problems follow. `eth` and `ethereum` parse alike but are different strings,
and the verifier compares DIDs as bytes (`rfc9421/verifier_http.go:401-408`). The on-chain key
is derived from the DID *string* (`eth/agentcard_client.go:596-598`), so `did:sage:eth:0xab…`
and `did:sage:ethereum:0xab…` are two records for one agent. The vector also accepts
`did:sage:ETHEREUM:agent-one`. [High]

### Address derivation

| Chain | Where | What is fixed |
|---|---|---|
| Ethereum, Go | `keys/keyid.go:38-47` | `"0x" + hex(Keccak-256(X‖Y)[12:32])`, lower-case, secp256k1 only |
| Ethereum, Go DID helper | `did/utils.go:217-228` | refuses anything but `crypto.KeyTypeSecp256k1` |
| Ethereum, Rust | `rs/src/crypto/keys.rs:109-123` | same formula, `PublicKey::Secp256k1` only |
| Ethereum validation | `chain/ethereum/provider.go:92-113` | strips `0x`, requires exactly 40 hex chars |
| Solana, Go | `chain/solana/provider.go:55-78,100-119` | base58 of the raw Ed25519 key, 32 bytes after decode |
| Solana, Rust | `rs/src/did/mod.rs:100-103` | base58 inline; no method, no validation |
| DID construction | `did/utils.go:138-148,175-185` | branches on `chain == ChainEthereum` for the `0x` prefix, then lower-cases the whole identifier |

That last row is a defect: `GenerateAgentDIDWithAddress` lower-cases a Solana address
(`utils.go:146`), and the doc comment at `utils.go:133-134` shows the mangled result as if
correct. base58 is case-sensitive. [High]

### The key policy that selects a signing key

`eth/key_policy.go:34-69` is the only such policy, and it lives in the Ethereum package: only
`Verified` keys are eligible (`:37-39`); the signing key is the first verified ECDSA key, else
the first verified Ed25519 (`:41-58,65-67`); the KEM key is the first verified 32-byte X25519
(`:59-62`). `applyKeyPolicy` (`:75-89`) writes the result into the legacy single-key view every
consumer reads.

The Solana client ignores it, fabricating one Ed25519 key marked `Verified: true`
(`sol/client.go:270-284`). Rust has a third policy keyed by a separate `Protocol` enum:
secp256k1 then P-256 for Ethereum, Ed25519 for Solana (`rs/src/crypto/multi_key.rs:247-275`).
"Which key signs" is answered three times, differently, and none is specified. [High]

### Proof of possession — three incompatible definitions

| Definition | Where | Message signed | Digest |
|---|---|---|---|
| Go core | `did/key_proof.go:218-221,46-82` | `"SAGE-PoP:" ‖ DID ‖ ":" ‖ hex(key)` | SHA-256, then Ed25519 or `ethcrypto.Sign` |
| Rust core | `rs/src/did/proof.rs:12-14,17-50` | identical string | SHA-256, low-S, `r‖s‖v` |
| Specification | spec 06 §4 | identical string | SHA-256; flagged as diverging from Keccak (spec 01 §5, O-1) |
| Contract, ECDSA | `AgentCardRegistry.sol:424-438` | `"SAGE Agent Registration:" ‖ block.chainid ‖ address(this) ‖ expectedOwner` | Keccak-256 then EIP-191 `"\x19Ethereum Signed Message:\n32"` |
| Contract, X25519 | `AgentCardRegistry.sol:443-464` | `"SAGE X25519 Ownership:" ‖ keyData ‖ block.chainid ‖ address(this) ‖ expectedOwner` | Keccak-256 then EIP-191 |
| Contract, Ed25519 | `AgentCardRegistry.sol:439-442` | nothing | none — only `require(signature.length == 64)` |

`SAGE-PoP` appears nowhere in `sage-contracts`, and `"SAGE Agent Registration:"` nowhere in
either core (grep across both repositories). The proof the specification describes is an
off-chain construct the registry never checks; the proof the registry checks is unspecified.
[High]

Three consequences. The contract's ECDSA proof binds the *owner account*, not the key and not
the DID (`:428-437`), so one signature authorises every `registerAgentWithParams`, `addKey` and
`updateKEMKey` by that owner on that chain and contract. The X25519 proof does bind `keyData`
(`:447-452`), so the branches differ in security properties. The Ed25519 branch marks the key
`verified = true` (`:192`, `:288`) having checked nothing, because the EVM has no Ed25519
precompile; the owner-pre-approval design in the comments (`AgentCardStorage.sol:76`;
`AgentCardRegistry.sol:440-441`) is not implemented anywhere. [High]

The Go `KeyRegistry` interface still carries `ApproveEd25519Key` (`did/registry.go:46-55`) for
that unimplemented design, and the CLI prints instructions for it
(`cmd/sage-did/key.go:321-326`). No chain client implements `KeyRegistry`, so `Manager.AddKey`,
`RevokeKey` and `ApproveEd25519Key` (`did/manager.go:344-398`) always fail their type assertion
— `sage-did key add` (`cmd/sage-did/key.go:309-315`) cannot work. [High]

### The agent card

The card carries no chain field; the chain is implicit in the DID string
(`did/types_v4.go:92-103`; `rs/src/did/a2a.rs:56-86`). What is chain-flavoured is the suite and
digest: key-type strings `Ed25519VerificationKey2020`, `EcdsaSecp256k1VerificationKey2019`,
`X25519KeyAgreementKey2019` (`did/a2a.go:108-119`; `rs/src/did/a2a.rs:115-130`); proof types
`Ed25519Signature2020` and `EcdsaSecp256k1Signature2019` (`did/a2a_proof.go:136-158`,
`:311-340`; `rs/src/did/a2a.rs:203-207`); and the secp256k1 card proof signs **Keccak-256** of
the canonical JSON (`a2a_proof.go:150` via `keys/secp256k1_keccak.go:35-39`), unlike the PoP
above, which signs SHA-256. Two proofs over the same registry, two digests. [High]

`ValidateA2ACardWithDID` (`did/a2a.go:271-330`) and `VerifyA2ACardProofWithDID`
(`a2a_proof.go:218-261`) require `Keys[].KeyData`, `Keys[].Verified` and `IsActive` —
registry-shaped requirements, not chain-shaped ones (section 3).

### The registry client and the on-chain record

| Decision | Where |
|---|---|
| `ethclient.Dial`, `NetworkID`, `HexToECDSA` key, `common.HexToAddress` contract | `eth/agentcard_client.go:77-110` |
| commitment `keccak256(abi.encode(did, keys, owner, salt, chainId))` | `agentcard_client.go:556-594`, matching `AgentCardRegistry.sol:142-150` |
| agent id `keccak256(did)` | `agentcard_client.go:596-598` |
| `bind.NewKeyedTransactorWithChainID` | `agentcard_client.go:639` |
| owner is `common.Address` rendered `.Hex()` | `agentcard_client.go:333`, `:390` |
| `ListAgentsByOwner` requires `common.IsHexAddress` | `eth/chainclient.go:116-120` |
| `Search` impossible on chain | `chainclient.go:25-27,136-138` |
| `Register` refused; commit-reveal only | `chainclient.go:19-23,34-36` |

The agent id is a second defect. The Go client hashes the DID alone, but the contract computes
`keccak256(abi.encodePacked(params.did, msg.sender, block.timestamp))`
(`AgentCardRegistry.sol:164`). `Update` and `Deactivate` (`chainclient.go:197`, `:204`) address
an id that cannot exist. Reads are unaffected: they go through `getAgentByDID`
(`chainclient.go:39-41`), which resolves `didToAgentId` on chain
(`AgentCardRegistry.sol:391-398`). [High]

### Network presets

`chain/presets.go:56-66` is the single table, mixing chains, chain ids, public RPC URLs and
deployed contract addresses: `local` (31337), `sepolia` (11155111, `AgentCardRegistry`
`0xC7eCF7…a808`), `ethereum-mainnet` (1), `kairos` (1001, legacy `SageRegistryV2`
`0x4Ba6Fc…6545`), `cypress` (8217), `solana-devnet`, `solana-mainnet`. The alias table
(`presets.go:69-79`) is where the vocabulary leaks: `kaia`, `klaytn` and `mainnet` all resolve
to `cypress`; `ethereum` resolves to `ethereum-mainnet`.

Kaia is typed `ChainTypeEthereum` (`presets.go:61,63`), so a Kaia deployment mints DIDs saying
`did:sage:ethereum:` — see section 5. `RegistryAddress()` (`presets.go:104-109`) silently
prefers `AgentCardRegistry` over `SageRegistryV2`, so one preset name can mean two registry
ABIs. CLI defaults are Sepolia and devnet (`cmd/sage-did/register.go:175-184`);
`internal/config/blockchain.go:48-70` layers gas parameters on top. [High]

The Rust core has **no** preset table, chain id, RPC URL or contract address anywhere;
`BlockchainDIDResolver::resolve` returns `Error::Unsupported("Blockchain feature not enabled")`
(`rs/src/did/resolver.rs:30-36`), and its only resolvers are in-memory and mock (`:38-200`).
The comment claiming an alloy integration (`rs/Cargo.toml:61-62`) is stale. [High]

### The gateway

`sage-gateway` owns no chain constants but hard-wires the chain at its composition step:
`gw/resolve/resolve.go:173-199` calls `chain.PresetFor(opts.Network)` then
`m.Configure(did.ChainEthereum, …)` unconditionally. `-network solana-devnet` selects a preset
with no registry address and fails at `resolve.go:185-187`; there is no path to a non-Ethereum
resolver. Flags are `-network`, `-rpc`, `-registry` (`gw/cmd/main.go:57-59`), help text naming
`sepolia` and `kairos`. It pins `sage v1.5.3-0.20260912041026-b04477bc0840` (`gw/go.mod:10`)
and depends on go-ethereum directly (`gw/go.mod:8`) purely to parse keys by byte length — 32 →
Ed25519, 33 or 65 → secp256k1 (`gw/resolve/resolve.go:69-79`) — a heuristic that cannot
represent X25519 or P-256. Signing-side algorithm selection is a separate switch
(`gw/keyfile/keyfile.go:83-94`). [High]

---

## 2. Per-chain differences that reach the protocol

Ethereum is implemented end to end. Solana is a Go client whose serialisation is a placeholder,
plus a name in both enums.

| Dimension | Ethereum, as implemented | Solana, as far as it exists |
|---|---|---|
| Identifier syntax | lower-case `0x` + 40 hex, optional `:<nonce>` (`did/utils.go:138-185`; spec 06 §1) | base58 of the Ed25519 key, but `GenerateAgentDIDWithAddress` lower-cases and breaks it (`utils.go:146`) — **wrong** |
| Address derivation | `Keccak-256(X‖Y)[12:32]`, secp256k1 only (`keys/keyid.go:38-47`; `rs/src/crypto/keys.rs:109-123`) | base58 of the raw key; no derivation function, no checksum (`chain/solana/provider.go:55-78`) |
| Signing key and digest | secp256k1 preferred, Ed25519 fallback (`key_policy.go:41-58`); Keccak-256, `r‖s‖v`, low-S (spec 01 §2; `rs/src/crypto/keys.rs:349`) | Ed25519 only, pure, no pre-hash; client hard-codes one key (`solana/client.go:270-284`) |
| PoP digest | contract: Keccak + EIP-191 over owner (`AgentCardRegistry.sol:424-438`); cores: SHA-256 over `SAGE-PoP:` (`key_proof.go:218`; `rs/src/did/proof.rs:12-19`) | **none** — no PoP generated, verified or stored; `Verified: true` asserted (`solana/client.go:277`) |
| Key-agreement key | X25519 in a dedicated `kemPublicKey` field, proven by an ECDSA signature over the key bytes (`AgentCardStorage.sol:55-68`; `AgentCardRegistry.sol:443-464`) | **absent**; `ResolveKEMKey` returns `nil` (`solana/resolver.go:47-61`) |
| Record storage | `mapping(bytes32 => AgentMetadata)` by `agentId`, plus `didToAgentId` and `ownerToAgents` (`AgentCardStorage.sol:142-164`); keys by `keccak256(keyData)` (`AgentCardRegistry.sol:171`) | a PDA from seeds `["agent", DID]` under the program id (`solana/client.go:241-250`) |
| Record read | `getAgentByDID` then `getKey` per hash (`agentcard_client.go:371-425`) | `GetAccountInfo`, then **`json.Unmarshal`**; the code says "In production, use proper borsh deserialization" (`solana/client.go:567-571`) — **incomplete** |
| "Verified" | per-key bool set once `_verifyKeyOwnership` does not revert, cleared only by `revokeKey` (`AgentCardRegistry.sol:192,288,327`); for Ed25519 it means "64 bytes long" | not modelled; fabricated by the client |
| "Active" | `active=false` at registration (`:210`), flipped by `activateAgent` after `activationDelay` (default `1 hours`, `:41,220,249`); **anyone may call it** (`:245-256`) | a bool in the account struct (`solana/client.go:55`), no delay, no defined transition |
| What authorises a write | `onlyAgentOwner` = record owner **or** approved operator (`:57-64`); registry admin `onlyOwner` for stake, delay, hook, pause (`:583-602`) | the fee payer signs; `Deactivate` accepts a keypair the Ethereum path then ignores (`chainclient.go:200-205`) |
| Registration | three phases, stake `0.01 ether` (`:40`), commit window 1 min to 1 h (`AgentCardStorage.sol:217-224`), max 24/day (`:253`) | one instruction with a human-readable message (`solana/client.go:551-554`) and `json.Marshal` as encoding (`:561-565`) — **incomplete** |

Solana's incompleteness is structural: JSON where Borsh is required means no Solana program can
read what this client writes, and no account it reads was written by a real program. [High]
Treat the Solana path as unimplemented for specification purposes. [Mid]

Two further Ethereum facts reach the protocol. `AgentMetadata.chainId` is stored
(`AgentCardRegistry.sol:211`) but never compared, so it documents rather than enforces.
`_recoverSigner` (`:467-491`) normalises `v` but does not reject high-`s`, so the registry's
own check is malleable; key reuse is blocked separately by `publicKeyUsed`. [High]

---

## 3. What the protocol actually needs from a registry

Derived from consumers, not from the declared interface. `Resolver` (`did/resolver.go:26-52`)
is wider than any verifier uses.

### Reads a verifier performs before accepting a message

| Call site | Read | Fields used |
|---|---|---|
| Gateway request | `ResolvePublicKey` (`gw/verify/verify.go:104`) | signing key; `IsActive` and "has a signing key" enforced inside (`eth/chainclient.go:44-56`) |
| Gateway response | same (`gw/sign/sign.go:133`) | same |
| HPKE server | `ResolvePublicKey` (`hpke/server.go:230`) | signing key |
| HPKE client, peer KEM | `ResolveKEMKey` (`hpke/client.go:200`) | X25519 key; `IsActive` enforced at `chainclient.go:59-68` |
| HPKE client, server identity | `ResolvePublicKey`, asserted `ed25519.PublicKey` (`hpke/client.go:443-451`) | signing key and its **type** |
| Handshake server | `ResolvePublicKey` (`handshake/server.go:182`) | signing key |
| Message verification | `ResolveAgent` (`core/verification_service.go:64`) | `IsActive` (`:70`), `Endpoint`/`Name` (`:79-82`), `Capabilities` (`:88`), `PublicKey` (`:92`), `Owner` reported only (`:108`) |
| Card validation | `Resolve` (`did/a2a.go:283-330`) | `Keys[].KeyData`, `Keys[].Verified`, `Endpoint`, `IsActive` |
| Card proof | `Resolve` (`did/a2a_proof.go:245-261`) | `IsActive`, `Keys[]` matched by fragment, `Keys[].KeyData` |

Collapsed, a verifier needs exactly this:

1. **given a DID, a set of keys**, each with raw public key bytes, a key type, and a boolean
   saying the registry accepted it;
2. **a liveness bit** for the agent;
3. **a KEM key**, which is just the X25519 member of (1) — `KEMKey()` already derives it that
   way (`did/types_v4.go:134-144`);
4. **endpoint, name and capabilities**, used only for metadata cross-checking
   (`verification_service.go:79-82`) and the capability gate (`did/verification.go:207-219`).

Items 1 and 2 are load-bearing, 3 is derived, 4 is advisory. [High] Nothing in the read path
needs an owner, transaction, block, chain id or address.

### Writes an owner performs

| Write | Needs | Where |
|---|---|---|
| Create a record with an initial key set | proof of control of each key; the DID unclaimed | `AgentCardRegistry.sol:122-238` |
| Make it live | a delay, or an explicit second step | `:245-256` |
| Add a key | possession proof for the new key, plus authority over the record | `:265-300` |
| Revoke a key | authority; at least one key must remain | `:307-330` |
| Replace the KEM key | proof binding the new key, plus authority | `:551-579` |
| Update endpoint and capabilities | authority | `:457-…` via `chainclient.go:160-198` |
| Retire the record | authority | `:361-379` |

"Authority over the record" is the only place a chain account is the mechanism, and that is a
*choice of mechanism*. The abstract interface is: **reads take a DID and return keys plus
liveness; writes take a DID, a change, and a proof the writer speaks for the record.**
Everything else in the Go `Registry`/`Resolver` is chain vocabulary — `RegistrationResult`
alone carries `TransactionHash`, `BlockNumber`, `GasUsed` and `Slot` (`did/types.go:65-71`).
[Mid] `Search` is already unsupported on Ethereum (`chainclient.go:136-138`) and
`ListAgentsByOwner` requires a hex address (`:116-120`); no verifier uses either, so neither
belongs in a protocol-level interface. [High]

---

## 4. What breaks with a registry that is not a blockchain

Take a signed JSON document over HTTPS, or a static file. The read path survives almost intact;
the write path and configuration do not.

The gateway's `Static` resolver is already a non-blockchain registry — a map from DID to public
key (`gw/resolve/resolve.go:96-104`), selected by `SAGE_TRUSTED_AGENTS` (`gw/cmd/main.go:56`).
The Rust core has only non-blockchain resolvers (`rs/src/did/resolver.rs:38-200`). The verifier
side has running proof it does not need a chain. [High]

| Assumption | Where | Protocol-level or implementation detail |
|---|---|---|
| A registry needs an RPC endpoint and contract address | `Manager.Configure` rejects either empty (`did/manager.go:58-63`) | **detail** — the check belongs in the Ethereum client |
| A registry is reached through a `Chain` | `Manager` keys by `Chain` (`manager.go:31-37`); `MultiChainResolver` dispatches via `ParseDID` (`resolver.go:87-112`) | **protocol-level**: the chain segment selects the registry |
| Gateway config is a chain preset | `gw/.../resolve.go:173-199`; `gw/cmd/main.go:57-59` | detail |
| A write yields a tx hash, block, gas, slot | `RegistrationResult` (`types.go:65-71`); `GetRegistrationStatus` reads a receipt (`chainclient.go:141-…`) | detail, but it sits in the *interface* (`registry.go:39-40`), so it leaks |
| The owner is a chain account | `AgentMetadata.Owner` is "Blockchain address of the owner" (`types.go:47`); `ListAgentsByOwner` needs hex (`chainclient.go:117`) | detail for reads (no verifier checks `Owner`); **protocol-level for writes**, being the current authorisation mechanism |
| Timestamps come from blocks | `CreatedAt`/`UpdatedAt` from `registeredAt`/`updatedAt` (`agentcard_client.go:335-336`) | detail; a document can carry its own |
| Activation is a delay after a transaction | `activationDelay` read from the contract (`agentcard_client.go:213-227`); `CanActivateAt` (`types_v4.go:236`) | protocol-level only in that a verifier needs *some* liveness bit; the delay is anti-Sybil for an open registry |
| Registration is commit-reveal with a stake | `PhaseCommitted/Registered/Activated` (`types_v4.go:242-248`); `Register` refuses otherwise (`chainclient.go:34-36`) | detail — front-running matters only when anyone may claim any name |
| Key acceptance is an on-chain flag | `Verified` drives key selection (`key_policy.go:37-39`) and card validation (`a2a.go:304-306`) | **protocol-level**: a verifier must learn which keys were accepted; how is the registry's business |
| Algorithms follow the chain | `chainKeyMap`/`chainSupportedKeys` (`chain/key_mapper.go:29-42`) | detail today, but the reason the chain segment implies a key type |

Reads are already chain-free; writes are not; and the genuinely protocol-level part is small —
a DID must say which registry to consult, a record must expose keys with an accepted flag and a
liveness bit, and a write must carry a checkable proof. [Mid] The concrete blockers for an
HTTPS-document registry are `Configure` demanding a contract address and RPC URL
(`manager.go:58-63`), `Chain` being a closed enum with no "http" or "file" member
(`types.go:84-87`, `manager.go:313-322`), and `RegistrationResult` being transaction-shaped.
None requires redesigning the verifier. [Mid]

One caution: a document registry gives up revocation freshness. The gateway caches for 5
minutes by default and evicts only on an explicit `Forget` (`gw/resolve/resolve.go:106-159`;
`gw/cmd/main.go:60-61`), so a revoked key already survives a cache lifetime on chain. A
document registry must carry an explicit expiry the verifier enforces, or revocation becomes
unbounded. **The specification needs to state this**; it is not an implementation choice. [Mid]

---

## 5. Uniqueness of identifiers across registries

**The scheme does not prevent two registries from naming the same agent, and the identifier
does not tell a verifier which registry to consult.** [High]

The DID is `did:sage:<chain>:<identifier>` (`did/manager.go:327-341`), and the chain segment
names a chain *family*, not a deployment.

Network is not in the DID — the specification says so: "Networks within a chain … are
configuration, not part of the DID" (spec 06 §2). The same `did:sage:ethereum:0xab…` denotes
one agent on Sepolia and another on mainnet, and which one a verifier gets depends on the
operator's `-network` flag (`gw/resolve/resolve.go:173-199`). Two gateways in one deployment
pointed at different networks accept different keys for one DID. [High]

Kaia is typed as Ethereum (`presets.go:61,63`), so a Kaia agent's DID reads
`did:sage:ethereum:`; the Kairos preset points at a different registry contract with a
different ABI (`presets.go:62`, `:104-109`). [High]

The same address may register on two chains, and the scheme invites it: one secp256k1 key
derives one Ethereum address (`keys/keyid.go:38-47`), so the same key registered on Ethereum
and on a Kaia deployment yields the *identical* DID string with two independent records, owners
and key sets. The registry's `chainId` (`AgentCardRegistry.sol:211`) is stored but never
compared, so it cannot adjudicate. On chain, uniqueness holds only within one contract, via
`didToAgentId` (`AgentCardStorage.sol:149`) and the `validDID` guard
(`AgentCardRegistry.sol:67-69`). [High] Add the normalisation gap from section 1 and one agent
can hold several distinct DID strings that parse identically while the on-chain key is the raw
string (`agentcard_client.go:596-598`). [High]

A chain-neutral scheme therefore needs a **registry identifier in the DID**, not a chain
family: something comparable without configuration, differing between Sepolia and mainnet,
between two contracts on one chain, and between a chain registry and an HTTPS one. That breaks
the grammar and every vector. [Mid] The non-breaking alternative is to keep the grammar and
require one registry per chain segment per verifier, stating plainly that a DID is only
meaningful relative to that configuration — cheaper, but it makes cross-organisation DIDs
unusable as names. [Mid]

---

## 6. Cryptographic agility

A new key type or algorithm must be added in three independently maintained places, one of
which is immutable once deployed.

### Go core

| Table or switch | Where |
|---|---|
| `crypto.KeyType` constants and the RFC 9421 algorithm table (the one designed extension point, `RegisterAlgorithm`) | `crypto/algorithm_registry.go:64-109`, `:132-…` |
| Key type inferred from a Go public key, incl. the secp256k1 field-prime check | `algorithm_registry.go:266-294` |
| Algorithm/key compatibility gate used by the verifier | `algorithm_registry.go:296-324`, called at `rfc9421/verifier_http.go:461-464` |
| Registry `KeyType` integers — "MUST match Solidity" | `did/types_v4.go:38-58` |
| `MarshalPublicKey` / `UnmarshalPublicKey` | `did/utils.go:35-105` |
| Legacy single-key coercion | `types_v4.go:250-269` |
| Key selection policy | `eth/key_policy.go:34-69` |
| PoP generate, verify, validate | `did/key_proof.go:46-82`, `:97-166`, `:233-272` |
| A2A verification-method strings | `did/a2a.go:108-119` |
| A2A proof types and digests | `did/a2a_proof.go:136-158`, `:311-340` |
| Chain-to-key-type maps | `chain/key_mapper.go:29-42` |
| CLI detection, JWK/PEM/raw parsing, size checks | `cmd/sage-did/helpers.go:185-269`; `cmd/sage-did/key.go:262-266` |

`algorithm_registry.go:64-109` already knows P-256 and RSA, but `did.KeyType`
(`types_v4.go:40-44`) has three members pinned to Solidity. The crypto layer is agile; the
registry layer is not. [High]

### Rust core

Twenty-one sites. The ones that would silently diverge if missed:

- `KeyType` (`rs/src/crypto/keys.rs:20-27`), the parallel `Algorithm`
  (`rs/src/crypto/mod.rs:28-35`) and their conversion (`keys.rs:29-37`);
- `PublicKey`/`PrivateKey`/`Signature` enums and every method match (`keys.rs:51-234`,
  `:247-262`, `:338-425`; `rs/src/crypto/signature.rs:14-105`);
- `AlgorithmRegistry::get_metadata`, whose arms carry `ethereum_compatible` and
  `solana_compatible` booleans per algorithm (`rs/src/crypto/algorithm_registry.rs:116-165`,
  fields `:81,84`, filters `:269,281`) — chain knowledge embedded in the crypto registry;
- `rfc9421::SignatureAlgorithm` (`rs/src/rfc9421/mod.rs:26-54`) and `algorithm_for`
  (`rs/src/rfc9421/signer.rs:61-67`), **duplicated** by a third literal table at
  `rs/src/core/message.rs:214-219`;
- A2A method and proof types (`rs/src/did/a2a.rs:115-130`, `:203-207`, `:276-280`), duplicated
  again at `rs/src/hpke/types.rs:373-377`;
- PoP (`rs/src/did/proof.rs:20-49`, `:53-85`);
- format export and OID tables (`rs/src/formats/mod.rs:96-398`), and the PEM tag table, which
  has **no P-256 arm today** (`rs/src/crypto/storage/file.rs:50-58`);
- the numeric enums at the FFI and WASM boundaries, `Ed25519 = 0, Secp256k1 = 1, P256 = 2`
  (`rs/src/ffi/mod.rs:85-89`; `rs/src/wasm/mod.rs:40-44`) — these **disagree with the
  registry's** `ECDSA = 0, Ed25519 = 1, X25519 = 2` (`did/types_v4.go:41-43`;
  `AgentCardStorage.sol:32-36`). Two numeric key-type spaces exist and neither is specified as
  authoritative. [High]
- chain selection by key type (`rs/src/did/mod.rs:94-108`) and the `Protocol` preference table
  (`rs/src/crypto/multi_key.rs:247-275`).

### Contracts

- the `KeyType` enum (`AgentCardStorage.sol:32-36`);
- `_verifyKeyOwnership`, an `if / else if / else if` chain **with no final `else`**
  (`AgentCardRegistry.sol:418-465`). A fourth enum member without a branch falls through
  without reverting and is then marked `verified = true` (`:192`, `:288`). **A new key type
  added to the enum without a matching branch is accepted with no validation at all** — a
  latent vulnerability in the agility path itself, not just maintenance burden. [High]
- the X25519-specific KEM extraction (`:181-185`) and `getKEMKey`/`updateKEMKey` (`:540-579`),
  which name X25519 in the signature and pass `KeyType.X25519` literally at `:571`;
- everything decoding the enum downstream: the generated bindings
  (`blockchain/ethereum/contracts/agentcardregistry/`) and the checked-in ABI
  (`eth/AgentCardRegistry.abi.json`).

A deployed contract cannot be edited, so a registry that enumerates key types caps agility at
deployment time. The alternative — the registry stores an opaque type label and delegates
validation to the reader — moves the acceptance decision off chain, which is where a
chain-neutral specification points anyway. [Mid]

---

## 7. What this means for the specification

1. **State what a registry is in terms of reads.** Section 3 is the whole surface: DID → (keys
   with type, bytes, accepted flag; liveness). Anything transaction-, block- or account-shaped
   stays out of the protocol text. [Mid]
2. **Decide the identifier question first.** The DID names a chain family, not a registry, and
   two registries can name one agent (section 5). Either put a registry identifier in the
   grammar and accept a breaking change, or state that DIDs are meaningful only under a named
   configuration. [Mid]
3. **Pick one proof of possession.** Three exist, no two agree (section 1). [High]
4. **Separate "the registry accepted this key" from "someone proved possession".** They are one
   flag on chain today (`AgentCardRegistry.sol:192`), which is why an unchecked Ed25519 key
   reads as verified. [High]
5. **Settle one numeric key-type space, or drop numeric types** (section 6). [High]
6. **Require an explicit expiry on any non-chain registry record** (section 4). [Mid]

Three implementation defects found while surveying should be filed separately, since they
change what "as implemented" means: `computeAgentID` disagrees with the contract, so `Update`
and `Deactivate` cannot work; `Manager.AddKey`/`RevokeKey`/`ApproveEd25519Key` have no
implementation behind them; and `GenerateAgentDIDWithAddress` corrupts Solana base58 addresses.
[High]

---

## Facts vs Opinions

**Fact** — verified by reading the cited file.

- Both DID parsers hard-code the method `sage` and a two-chain vocabulary, and neither checks
  the identifier against an address shape (`manager.go:313-341`; `rs/src/did/mod.rs:26-84`);
  `did:sage:ethereum:agent-one` is an accepted vector (`sage-spec/vectors/did.json`).
- Three incompatible proofs of possession exist: `SAGE-PoP` over SHA-256 in both cores
  (`key_proof.go:218`; `rs/src/did/proof.rs:12-19`) and spec 06 §4; `"SAGE Agent
  Registration:"` over Keccak + EIP-191, binding the owner account
  (`AgentCardRegistry.sol:424-438`); and `"SAGE X25519 Ownership:"`, binding the key bytes
  (`:443-464`). Neither core contains the contract's strings, nor the contract either core's.
- The contract accepts an Ed25519 key on a length check alone and marks it `verified`
  (`AgentCardRegistry.sol:439-442`, `:192`, `:288`); the owner-pre-approval design in the
  comments (`AgentCardStorage.sol:76`) is not implemented.
- `_verifyKeyOwnership` has no `else` branch (`AgentCardRegistry.sol:418-465`), so a new
  `KeyType` member without a branch is accepted unvalidated and marked verified.
- No chain client implements `KeyRegistry`, so `Manager.AddKey`, `RevokeKey` and
  `ApproveEd25519Key` (`manager.go:344-398`) always fail their type assertion, and `sage-did
  key add` (`cmd/sage-did/key.go:309-315`) cannot succeed.
- The Go client computes `agentId = keccak256(did)` (`agentcard_client.go:596-598`); the
  contract computes `keccak256(abi.encodePacked(did, msg.sender, block.timestamp))`
  (`AgentCardRegistry.sol:164`). `Update` and `Deactivate` use the former
  (`chainclient.go:197`, `:204`).
- `GenerateAgentDIDWithAddress` lower-cases the identifier for every chain (`utils.go:146`),
  corrupting Solana base58 addresses.
- Kaia presets are typed `ChainTypeEthereum`, one pointing at a different registry contract
  (`presets.go:61-63`, `:104-109`); network is configuration, not part of the DID (spec 06 §2).
- The Rust core has no chain client, chain id, RPC URL or contract address;
  `BlockchainDIDResolver::resolve` always errors (`rs/src/did/resolver.rs:30-36`), and its
  registry model is a W3C `DIDDocument` with no liveness flag, owner or per-key accepted flag
  (`rs/src/hpke/types.rs:296-310`, `:342-356`).
- The gateway configures `did.ChainEthereum` unconditionally (`gw/.../resolve.go:190`), reads
  only the signing key before accepting a request (`gw/.../verify.go:104-127`), and caches
  resolutions with a 5-minute default TTL (`gw/.../resolve.go:106-159`;
  `gw/cmd/main.go:60-61`).
- Two numeric key-type spaces disagree: `ECDSA = 0, Ed25519 = 1, X25519 = 2`
  (`types_v4.go:41-43`; `AgentCardStorage.sol:32-36`) against `Ed25519 = 0, Secp256k1 = 1, P256
  = 2` (`rs/src/ffi/mod.rs:85-89`; `rs/src/wasm/mod.rs:40-44`).
- `activateAgent` has no access control (`AgentCardRegistry.sol:245-256`), and `_recoverSigner`
  does not reject high-`s` (`:467-491`).
- The Solana client serialises with `json.Marshal`/`Unmarshal` where Borsh is required
  (`solana/client.go:561-571`), stores no KEM key (`solana/resolver.go:47-61`), and asserts
  `Verified: true` without any proof (`solana/client.go:277`).

**Opinion** — inference.

- [High] The Solana path should be treated as unimplemented for specification purposes:
  JSON-versus-Borsh means no real Solana program could interoperate in either direction.
- [High] The `verified` flag conflates "a proof was checked" with "the registry accepted this
  key", which is why an unchecked Ed25519 key is indistinguishable from a proven secp256k1 one
  in every consumer reading `Keys[].Verified`.
- [Mid] The read path is already chain-neutral in practice — the gateway's static resolver and
  the Rust in-memory resolvers are working non-blockchain registries — so decoupling cost falls
  almost entirely on writes, configuration and the identifier scheme.
- [Mid] A registry identifier in the DID grammar is the only option that makes a DID a name
  rather than a name-plus-configuration; the cost is a breaking grammar change and reissued
  vectors.
- [Mid] A deployed contract that enumerates key types caps agility at deployment time; an
  opaque type label validated by the reader suits a chain-neutral registry.
- [Mid] A document-based registry needs an explicit, verifier-enforced expiry, since the
  freshness a chain supplies implicitly would otherwise be unbounded.
- [Low] The alias and case-normalisation gaps are likelier to cause operational confusion (two
  records for one agent) than to be exploited, since creating the second record still requires
  passing the registry's own checks.
