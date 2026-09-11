# Security Policy

## Supported versions

| Version | Supported |
|---|---|
| 1.5.x (latest minor) | Yes |
| < 1.5 | No |

## Reporting a vulnerability

Please do not open a public issue for security problems.

Use GitHub's private vulnerability reporting for this repository:
https://github.com/SAGE-X-project/sage/security/advisories/new

Include the affected component (Go package, CLI, contract, workflow), a reproduction, and the impact you see. You will receive an acknowledgement within 5 business days. We aim to publish a fix and an advisory within 90 days of the report; if a fix takes longer we will tell you why and agree a disclosure date with you.

## Scope

- Go library and CLIs in this repository (`pkg/`, `cmd/`)
- Smart contracts in `contracts/`
- CI workflows and release artifacts

The multi-language SDKs under `sdk/` are experimental and not covered by this policy until they are released separately.

## Handling

Reports are triaged by the maintainers, fixed on a private branch, released as a patch version, and published as a GitHub Security Advisory with a CVE where applicable. Credit is given to reporters unless they ask otherwise.
