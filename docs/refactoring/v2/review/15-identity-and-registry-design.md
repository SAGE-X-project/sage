# Identity and registry: design proposal

Status: proposal for the specification's design stage, 2026-09-13. It
answers the two questions the charter left under study: how the registry is
described without tying it to one chain, and how far the identifier scheme
follows the decentralised identity standards. Evidence comes from
`12-did-standards-research.md` (standards, grounded in the maintainer's own
study at `0xmhha/study/projects/go-did`), `13-revocation-research.md`
(immediacy) and `14-registry-coupling-analysis.md` (what the code couples
to a chain). Nothing here is normative until it lands in `spec/`.

## 1. Defects that force the decision

| # | Defect | Evidence | Severity |
|---|---|---|---|
| 1 | An identifier does not name the registry that holds the record. The specification says the network is configuration, not part of the identifier, so an agent at address `0x…` on a test network and a different agent at the same address on the main network carry the same identifier, and a resolver cannot tell them apart | `sage-spec/spec/06-did-sage.md` §2; `12` §1; `14` §5 | Security, high confidence |
| 2 | The proof of possession is therefore replayable across networks: the challenge is the identifier and the key bytes, neither of which names the network, so a proof produced for a test registration is byte-identical for a main-network registration of the same address and key | `06-did-sage.md` §4; `12` §1 | Security, high confidence |
| 3 | Two proof schemes exist and neither matches the other. The text signs `"SAGE-PoP:" ‖ DID ‖ ":" ‖ hex(key)` with the key being registered; the contract recovers an Ethereum signature over `"SAGE Agent Registration:" ‖ chainId ‖ contract ‖ owner`, which proves the owner signed, not that the key holder did. Because that proof names neither the key nor the identifier, one signature authorises every later key addition by the same owner | `sage-contracts/ethereum/contracts/AgentCardRegistry.sol:418-465`; `14` §1 | Security, high confidence |
| 4 | An Ed25519 key is accepted with a length check and no verification, then read back as proven | `AgentCardRegistry.sol:441-443`; `14` §7 | Security, high confidence |
| 5 | Chain aliases mean two identifier strings denote one agent, which the resolution contract forbids unless one form is canonical | `06-did-sage.md` §2; `12` §1 | Correctness |
| 6 | The key verification in the contract is a chain of conditions with no final branch, so a key type added later is accepted without any check and then read back as proven. The path meant to give cryptographic agility is itself the hazard | `AgentCardRegistry.sol:418-465`; `14` §6 | Security, high confidence |
| 7 | One chain family covers registries with different contracts: a second Ethereum-compatible network is typed as Ethereum and points at a different registry, so which agent an identifier denotes depends on an operator flag | `14` §5 | Security, high confidence |

Defects 1 to 4, 6 and 7 are reasons to change the wire format and the
registry before the text is frozen, not after. Three further defects are
implementation faults rather than design faults and should be filed on
their own: the agent identifier computed by the client cannot match the one
the contract stores, so updates and deactivations address a record that
cannot exist; the helper that builds an identifier from an address
lower-cases it and corrupts base58 addresses; and the Solana client encodes
its calls as JSON where the chain requires a binary encoding (`14` §7).

## 2. What a verifier actually needs

Derived from every call site that resolves before accepting a message
(`14` §3), the read surface is:

1. given an identifier, a set of keys, each with raw public key bytes, a key
   type and a statement that the registry accepted it;
2. a liveness statement for the agent;
3. the key-agreement key, which is the X25519 member of (1);
4. endpoint, name and capabilities, used only for metadata checks.

Items 1 and 2 decide whether a message is accepted. Nothing in the read
path needs an owner account, a transaction, a block or a chain identifier.
That is the whole protocol-level contract, and it is what makes a registry
that is not a blockchain possible.

## 3. Identifier scheme

The identifier must name the registry, because that is what makes it unique
and what tells a resolver where to look. Two ways to write it.

### Option A: self-describing locator (recommended)

```
did:sage:<kind>:<locator>:<agent-id>
```

`kind` names a registry family and is registered in this specification.
`locator` is defined by the profile for that kind and identifies the
registry instance. `agent-id` is unique within that instance.

| Kind | Locator | Example |
|---|---|---|
| `eip155` | the chain identifier of CAIP-2 and the registry contract address | `did:sage:eip155:11155111:0xC7eC…a808:0x1234…` |
| `solana` | the chain identifier and the program address | `did:sage:solana:EtWTRAB…:Prog…:Agent…` |
| `web` | a domain name that serves the registry document | `did:sage:web:agents.example.com:alice` |

Cost: identifiers are long, and existing identifiers change. Gain: no
central table to maintain, uniqueness holds by construction, two registries
on one chain are distinguishable, and the chain part matches what
`did:pkh` already uses so external tooling can read it.

### Option B: registered short token

```
did:sage:<registry-token>:<agent-id>          did:sage:eth-sepolia-v1:0x1234…
```

The specification maintains a table from token to (chain, network, registry
address). Cost: a private registry with governance, and no path for a tool
that has never heard of SAGE to work out where to look. Gain: short,
readable identifiers and a smaller edit to the current grammar.

Either option is a breaking change to the identifier, which is acceptable
only before `1.0.0`. Both close defects 1 and 2, because the proof of
possession is bound to an identifier that now names the registry.

## 4. The registry model

One abstract model, one profile per registry kind. The model states what
every registry must offer; the profile states how that kind does it.

### What the model requires

| Operation | Requirement |
|---|---|
| Read | Given an identifier, return the key set, the per-key acceptance state and the liveness state, or say the identifier is unknown |
| Create | Bind an identifier to an initial key set and a controller, with a proof of possession per key, and refuse an identifier that is already bound |
| Activate | Make a record usable, as a step distinct from creation |
| Add key | Accept a new key with its own proof, under the controller's authority |
| Revoke key | Mark one key unusable, keeping its identifier so that a verifier can distinguish revoked from never present |
| Deactivate | Make the whole record unusable |
| Authorise | Let the controller name another party who may perform the writes |

### What each profile must declare

1. The locator syntax and how an agent identifier is formed within it.
2. The key types and signature algorithms it accepts, and the encoding of
   each. This is where chains differ, and the difference stops here.
3. How control of the record is proven for writes.
4. The observation point: what counts as the current state, and how a
   reader obtains it (§6).
5. Whether records expire, and if so the maximum lifetime.
6. The cost and latency of each operation, published as measurements.

### The profile for a registry that is not a blockchain

A record is a document served over HTTPS at a path derived from the
identifier, signed by the controller's key, carrying an expiry and the same
key set and liveness statement. Control of the locator is proven by control
of the domain, exactly as `did:web` does; revocation is the publication of
a new document. This profile satisfies the same read contract, so a
verifier does not know or care which kind it is talking to. It needs one
extra rule that chains give for free: a record must carry an expiry, since
there is no block height to bound staleness.

## 5. Following the decentralised identity standards

The maintainer's study of the standards is the source here
(`0xmhha/study/projects/go-did`, sections quoted in `12`).

| Thing to adopt | Why | Cost |
|---|---|---|
| Publish `did:sage` as a method specification and reserve the name | Without it no external resolver can be conformant, and five agent-oriented method names are already taken | A specification section and a registry pull request |
| Resolve to a document rather than to a private record shape | The document is the registry-neutral contract: the chain profile and the hosted profile produce the same document, which is what makes a non-blockchain registry a profile rather than a second design | A mapping: keys become verification methods, the endpoint becomes a service, the controller becomes the controller, liveness becomes resolution metadata |
| Per-key state instead of one liveness flag | The controlled-identifier model carries expiry and revocation per key, which is what makes rotation expressible; today only the whole agent can be deactivated | A record field and the rules that go with it |
| Key material as a standard encoding rather than numeric types | Numeric key types are a private table that every implementation and the contract must agree on, and they are the place a new algorithm has to be added in six files | A migration of the record encoding |
| Canonical and equivalent identifier metadata | Resolves the alias defect: one form is canonical, the others are declared equivalent | Two metadata fields |
| Structured resolution errors and the deactivated-is-not-an-error rule | A deactivated agent is a successful resolution that returns a deactivated document; today it is an error, which loses the distinction from "unknown" | Error mapping |

What not to adopt: the agent-specific drafts. The registry proposal that
names agents on Ethereum is a draft that does not require decentralised
identifiers; the agent identity Internet-Draft is an individual submission
that invents its own method; the agent card specification signs cards but
says nothing about identifiers. The stable ground is the identifier,
controlled-identifier and credential recommendations.

## 6. Making revocation immediate

The charter requires that a verifier never accept a message authenticated
by a revoked key. The research (`13`) turns that into three rules.

1. **The verifier grants no grace.** A cache may remember that a key is
   revoked; it must not remember that a key is valid. Negative caching only.
2. **The observation point is stated per profile.** For a chain, liveness
   and revocation are read at the most recent state, while key material is
   read at the finalised state. A reorganisation then fails safe in both
   directions: an unseen revocation causes over-rejection, and a key that is
   not yet final is not yet usable. On the main Ethereum network this is the
   difference between about twelve seconds and about sixteen minutes of
   exposure.
3. **Unreachable means reject.** If the registry cannot be read, the
   verifier rejects rather than falls back to a cached acceptance.

Consequence to accept knowingly: verification does a registry read per
message, so the requirement that cheap checks come first becomes
load-bearing, and verifier availability is tied to the registry endpoint.
The alternatives that the rest of the industry uses, a bounded cache with
event invalidation and a registry-signed freshness statement attached to
the message, are both excluded by the immediacy rule, because a bound is an
extension and a proof of inclusion at one height is not a proof of currency
at the next.

## 7. One proof of possession

Replace the two schemes with one, bound to what makes it unique:

```
challenge = "sage-pop-v1" ‖ registry-id ‖ agent-id ‖ key-type ‖ key-bytes
```

signed by the key being registered, verified by the registry and by any
reader of the record. Every key type is verified, including Ed25519; a
registry that cannot verify a type does not accept keys of that type, which
is the rule that closes defect 6: an unknown type is refused rather than
accepted unchecked. The digest convention follows the key's own chapter
rather than being special. Because the proof names the registry, the agent
and the key, it authorises exactly one key in one registry, which is what
defect 3 fails to do.

## 8. What changes in the specification

| Chapter | Change |
|---|---|
| 00 Overview | Conformance levels gain the registrar level; the sentence that makes an implementation normative is deleted when the design stage closes |
| 06 did:sage | Rewritten around the identifier scheme of §3 and the method specification of §5; alias handling becomes canonical and equivalent identifiers |
| New: registry and lifecycle | The model and operations of §4, the state machine, the immediacy rules of §6, the single proof of §7, and one section per profile: chain and hosted |
| New: document projection | The mapping of §5, which is also the interface a resolver implements |
| 01 Crypto | Key material encoding replaces the numeric key type table; the proof of possession digest is unified |
| 03, 04, 05 | Unchanged by this proposal, beyond the corrections already queued |

## 9. Decisions needed

| # | Question | Option 1 | Option 2 |
|---|---|---|---|
| 1 | Identifier form | Self-describing locator (§3 A). Cost: long identifiers, every existing identifier changes. Gain: uniqueness by construction, readable by external tooling, no governance | Registered short token (§3 B). Cost: a table only this project maintains. Gain: short identifiers, smaller edit |
| 2 | The document projection | Normative in `1.0.0`, as the registry-neutral read contract. Cost: another encoding to keep in step with the record. Gain: the hosted profile and the chain profile become one design, and external resolvers work | Deferred past `1.0.0`, keeping the private record shape. Cost: the hosted profile needs its own contract, and the standards alignment is postponed |
| 3 | Ownership proof for the hosted profile | Control of the domain, as the web method does. Cost: ties identity to the naming system. Gain: no new machinery | A controller key published out of band. Cost: bootstrapping problem. Gain: independent of the naming system |
| 4 | Per-key revocation state | Adopt per-key expiry and revocation now. Cost: a record migration and a contract change. Gain: rotation becomes expressible and revocation stops meaning "the key vanished" | Keep one liveness flag for `1.0.0`. Cost: rotation stays unexpressible |
