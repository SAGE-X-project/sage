# Compiler storage under the original lifetime

This work in progress provides storage for the compiler's actual byte and
offset representation. It is an internal library component. It selects no
operating principal, policy, host, profile or storage directory.

The Darwin implementation maps and locks the bytes that producers and
consumers use. A separate locked buffer cannot qualify allocations elsewhere.
Offsets contain no Go pointers. Access remains checked against the original
owner, committed range, action context and deadline. Closing waits for active
access and spends cleanup once under the original cleanup context.

Acquisition, use and cleanup can fail. A native call depends on the trusted
operating system returning; a context check does not make that call interruptible.
The fixed process owner still provides the original hard stop. A cleanup failure
withholds completion and may leave a mapping until process termination. It is
never reported as released memory.

The region does not back Go wrappers, runtime bootstrap, stacks, maps, strings or
other objects allocated outside it. Those responsibilities remain open. A small
native test proves only its actual region and operations.

Failure units use a private simulated-operation seam with small Go buffers.
Public acquisition always selects the native implementation. Simulated results
do not establish physical backing; the separate Darwin runtime tests use the
public acquisition path and actual native operations.

`ParseJSON` now uses the actual region for its input, node records and bounded
nonrecursive parser stack. It seals the region and exposes checked object,
array, string-comparison and scalar readers. Integer readers consume the same
original spans without a temporary string or floating-point representation.
Decoded duplicate keys, malformed UTF-8, trailing data and exhausted limits
refuse the parse. An entered parse failure leaves the region unusable until
cleanup. These library components are not yet the preparation envelope's exact
schema or the configuration, frozen plan and issuer connection below.

The complete connection must include these changes before it can qualify a
compiler or issue a private derivation:

1. Acquire actual bootstrap and compiler storage before session construction.
2. Decode into the region, using explicit parser scratch and byte/offset records.
3. Keep the selected configuration and plan in that representation, and make
   the original consumers read it directly throughout the same lifetime.
4. Close the remaining Go allocation, bootstrap and overlap obligations for
   the selected build. Retaining or reconstructing ordinary Go objects does
   not discharge those obligations.
5. Consume the same plan once through the internal issuer and existing run
   path, then validate cleanup and retained results in the existing runner.

The existing Go diagnostic path remains unchanged while this connection is
incomplete. No Source readiness, resource qualification, derivation or execution
permission is produced by this package.
