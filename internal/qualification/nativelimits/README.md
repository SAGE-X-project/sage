# Native resource primitive observations

The opt-in Darwin test runs five fixed controls in disposable C children. It
checks address-space admission and refusal, open-file exhaustion and cleanup,
file-size termination, cumulative CPU-time termination, and cancellation after
an actual child readiness marker. The cancellation control has a ten-second
alarm fallback and uses lifecycle-protected cancellation of its single child. Its
returned attempt must be read back exactly from the parent's journal while
the original parent owner remains valid. This is a finite single-child
observation, not a claim about arbitrary descendant processes. The parent keeps
one common owner deadline before the package timeout, with a separate journal
window. It records each completed child attempt immediately, including failures.
The test does not change the parent process's resource limits.

Each observation records the actual operating-system version, kernel release,
Go version, compiler identity and hash, probe source and binary hashes, child
status, elapsed time and CPU accounting. The actual boundary file remains in
the evidence directory with its size, hash and read-completion/error state.
Partial and absent originals remain partial or absent. Atomic journal updates
keep earlier completed observations available when a later operation fails.
Forced runner loss or non-returning filesystem I/O can leave an incomplete
journal; neither that journal nor a successful finite observation qualifies
the complete workload. The remote CI artifact also pins the
tested revision and source tree. No network or Registry call is performed.

These finite controls are an initial check of native enforcement mechanisms.
They do not prove a complete Registry profile fits, reserve physical memory or
storage, qualify the user's Mac, cover descendant processes, or fund compiler
bootstrap and cleanup. They issue no lease, source-derivation witness, Source
or operating authority. A full qualification entry still needs its own pinned
backend, complete analytic bounds, adequate resources, owning lifetime and
actual complete-profile results.

The `registry_native_limits` build tag keeps these native controls opt-in. The
dedicated remote job runs them on a Darwin runner and retains failed evidence.
The native process boundary is required because resource-limit reductions
cannot safely be undone in a shared Go test process.

Apple's pinned XNU source provides the starting mechanism references:
[resource limits](https://github.com/apple-oss-distributions/xnu/blob/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/bsd/kern/kern_resource.c#L1394)
and [address-space limits](https://github.com/apple-oss-distributions/xnu/blob/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/bsd/kern/kern_resource.c#L1560).
The runner's observed kernel and actual results determine the scope of the
artifact; this reference is not a claim that every installed Mac matches it.


### Ordinary-user capability observations

The native job also queries only its own process memory-limit properties and
observes a 32 KiB preallocation on an empty private file inherited as descriptor
3. It preserves populated file contents, records API absence or permission
denial, and never changes process memory limits or privileges. Readback checks
cover the same inode, unchanged logical length, allocated blocks and file sync.

Each compiler-version, compilation and observer attempt saves at most 4 KiB of
stdout and stderr before decoding or checking its result. The attempt record
includes observed byte counts, truncation, exit/signal/context state, and the
new observer source, binary and compiler identities. Overflow cancels the
command within the original common deadline. Compile failures retain the same
records; inability to preserve them fails the test. These are observed build
identities, not authenticated toolchain attestations. A missing binary before
compilation is recorded as absent and never treated as a valid binary identity.

The resulting capability JSON is partial evidence. An address-space setting, a
memory-limit query, or backing allocated to this single file does not qualify
the complete Registry execution, its whole RAM/backing lifetime, or the storage
used by both Registry roots. `full_native_qualification` remains false. The
ordinary-user observation is skipped when the test process is privileged; that
skip is not a successful capability observation.

API references: [Apple fcntl documentation](https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man2/fcntl.2.html)
and [the pinned XNU memory-status header](https://github.com/apple-oss-distributions/xnu/blob/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/bsd/sys/kern_memorystatus.h).
