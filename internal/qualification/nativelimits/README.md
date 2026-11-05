# Native resource primitive observations

The opt-in Darwin test runs four fixed controls in disposable C children. It
checks address-space admission and refusal, open-file exhaustion and cleanup,
file-size termination, and cumulative CPU-time termination. The parent keeps
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
