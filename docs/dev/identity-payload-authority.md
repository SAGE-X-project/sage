# Informational identity data in protected session payloads

The bounded SAGE 0.10.0 HTTP session APIs authenticate the configured participant
and exact registry signing key before returning application bytes. An application
payload may contain JSON with a DID document or alias-looking members. Those bytes
are opaque to these APIs and cannot supply a key, select a different participant,
or override a failed registry observation.

The scenario tests use three fixed, inert payloads: a document-shaped object,
alias-looking members, and their combination. For each payload they first complete
a genuine handshake, HTTP-bound signed/encrypted request and correlated response,
compare all returned payload bytes, check the configured sender/recipient/key tuple,
and refuse request replay and a second terminal response. They then seal a new
request before simulating either participant's signing-key revocation or a Source
read failure. Receive must fail without releasing plaintext or publishing another
record acceptance. Every combination starts with fresh endpoints and journals.

This is a defensive unit scenario using the actual cryptographic implementation,
not an exploit program, external service test, or DID-document consumer. The
Source supplies prevalidated fixture snapshots; it does not prove deployed
registry trust, immutable history, finality, custody, or operating permissions.
The early Source refusal has no new acceptance; a different storage failure may
retain durable denial staging and is not required to roll it back.

Payload non-authority is only a supporting boundary for DIDP-04. The session API
does not accept a peer DID document as a separate key-authority input. These tests
therefore do not establish arbitrary document/alias consumption behavior, close
the historical projection subject's two unsupported verifier cases, or certify an
A2A production ingress. No new document-ignoring wrapper or authority API is added.
Application dispatch and local authorization remain separate host responsibilities.

The focused CI evidence is retained alongside the ordinary full repository gates.
No production code, dependency pin, protocol version, or frozen Inspector result is
changed by this test group. The matching Go and Rust tests exercise the same nine
payload/refusal combinations; a Rust test group is not counted as nine test cases
in the test harness inventory.
