# Embedding SAGE in an Agent

SAGE provides security functions used by Agent application code. The Agent owns
message routing, policy, HTTP parsing and the decision to invoke these functions.
Registry observation and validation can run inside the same Agent process through
an imported library. The term **Source** describes the trusted Registry observation
boundary; it does not require another service, repository, server or administrator
key merely to run a demonstration.

## Current profile and intended application behavior

The normative comparison below uses sage-spec
`a22a275b4d88f1d415b53cdffc5bed41231b77a0`, especially
`spec/03-rfc9421.md` and `profiles/sage-0.11.0/registry-management.md`.
This document does not change their requirements.

The adopted 0.11.0 V1 selection in `profiles/sage-0.11.0/selection.json`
also retains its pinned HTTP component rules with the explicitly selected
version and domain substitutions. It does not introduce variable coverage or
Accept-Signature negotiation. The implementation must use new matching entry
points and signed objects; translating an existing 0.10.0 signature or merely
changing a Registry address is not migration.

| Concern | Current contract | Integration consequence |
| --- | --- | --- |
| Which messages use signatures | The 0.10.0 HTTP profile describes protected messages. It does not require signing every unrelated Agent communication. | The Agent routes operations according to its approved policy. A protected endpoint cannot retry an authentication failure as unsigned traffic. |
| Which components are signed | RFC 9421 supports application-selected components. MSG-02 and MSG-03 currently fix exact request and response component lists. Content-Digest covers the complete received HTTP content bytes. | Arbitrary partial JSON signing or a variable component list is not existing SAGE 0.10.0 conformance. Do not silently relax that profile. |
| Peer signature requests | RFC 9421 section 5 defines Accept-Signature. The current SAGE HTTP profile does not define a negotiation procedure for variable coverage. | The intended Agent-side request handling needs an explicit subsequent profile decision and Inspector tests before it is claimed as supported. |
| Cryptographic algorithms | MSG-01 permits only ed25519 and ecdsa-p256-sha256 for this HTTP profile, bound to the exact active authorized key. Other profiles have different roles and suites. | A peer's header is input to local policy, not permission to use an arbitrary algorithm, substitute another key or downgrade. Signature algorithms and HPKE KEM/AEAD suites remain separate. |
| Registry Source | CM-09 specifies observation identity, finality, continuity, recovery and freshness. It does not mandate a separate Source deployment. | Put validation and its necessary retained state inside the Agent integration. A successful RPC query alone does not authorize messages. |

RFC 9421's generic capabilities and SAGE's selected profile are different scopes.
Unsigned components are not authenticated merely because another component verifies.
Accept-Signature expresses a requested signature; the received Signature-Input and
Signature still need verification against application requirements. No such header
alone establishes support, trust, authority or successful key agreement.

## Agent integration and demonstration sequence

An importing Agent supplies explicit trusted configuration, an approved Registry
reader and its required local state. Its Registry code checks the selected instance,
the observed record and history, active key roles, continuity and freshness. It then
provides eligible identity information to the Agent's message protection path.
The application's configured policy determines what protected operation may execute.

For an outbound protected operation, Agent code selects the applicable profile and
permitted key, constructs the covered content and headers through SAGE, and sends
the message. For an inbound operation, it parses the transport headers, checks the
profile's required coverage and algorithms, resolves the exact active key and uses
SAGE to verify before dispatch. Response verification retains the original request
context. Unprotected application traffic, if allowed, has its own explicit route;
it never becomes a fallback for failed protected traffic.

Demonstrations and examples must import these APIs, rather than reproduce the
cryptography or replace Registry eligibility with an accept-all callback. The
existing work order remains:

1. Finish the adopted Registry connection, continuity and recovery components.
2. Bind the Go/Rust/ADK protected paths to the actual Registry observations and
   test a normal Agent request and its correlated response.
3. Update Inspector evidence and the separate host assurance checks.

Use the selected Sepolia Registry and the sole approved RPC from
[the Registry integration notes](registry-source-checks.md). Keep old V4/local-node
examples separate; changing their address does not convert them to the new binding.
State whether a run used the actual chain or an owned fixture. A loopback exchange
with test keys is useful runtime evidence, but it is not actual Registry-authorized
messaging. An absent record must prevent that protected exchange, not select an
unrelated local key. Registering demo identities or submitting additional transactions
requires the separately reviewed concrete execution proposal; read-only connection
tests require no deployment key.

## Recovery inside an Agent

Normal startup requires no new administrative signature. A policy change that
resolves quarantined Registry history is a separate authenticated recovery action.
`PlanJournalReconfiguration` and `RebuildReplayJournal` prepare a candidate using an
explicit public verification key. They do not load a private key, enroll that key
as an administrator, activate a generation, consume a nonce or make a Source ready.
All retained evidence must replay under both configurations with the exact externally
retained checkpoint. Unknown files, changed marker membership, missing history and
concurrent retained-prefix changes refuse. The old directory is never repaired,
truncated or unlocked by reconstruction.

The candidate is created in a separate empty directory. The caller still owns
authenticated approval, durable audit and checkpoint installation, nonce consumption,
old-generation fencing and fresh verification before using a recovered generation.
These are responsibilities of the Agent's trusted local integration, not an implicit
requirement to deploy a new security service. A demo does not establish protection
against compromise of that trusted integration or its underlying storage.

References: [RFC 9421 components](https://www.rfc-editor.org/rfc/rfc9421.html#section-2),
[application verification requirements](https://www.rfc-editor.org/rfc/rfc9421.html#section-3.2.1),
and [signature requests](https://www.rfc-editor.org/rfc/rfc9421.html#section-5).
