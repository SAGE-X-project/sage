# Owning Registry worker candidate

This internal component owns one disposable child invocation. It starts its
common deadline before executable copies and input copying, leaves a separate
cleanup interval, spends the attempt before input validation, copies both
pinned executables into a private directory, and returns detached completed
attempt data. It removes only its own two executable copies and directory.
It never retries, restores a spent attempt, accepts a serialized derivation,
or returns a Registry execution capability.

The child is an implementation detail of the Agent's own invocation. It is not
a separate Source service and introduces no Source administrator or signing key.

Caller cancellation stops the action. The already held cleanup reservation
keeps its original finite deadline, bounded also by the caller's original
deadline. It is never renewed. An expired cleanup owner performs no further
directory deletion and cannot report cleanup completion.

The Darwin launcher lowers and reads back the child process's CPU-time,
per-file-size, descriptor and core-dump limits before executing the selected
worker. The parent owns cancellation and bounded input, result, diagnostic and
readiness I/O. The launcher closes its readiness descriptor on exec. A failed
copy, launch, command, limit acknowledgment, output boundary or cleanup prevents
successful result publication. Already completed diagnostic data may remain.

These controls have specific scopes. CPU time is cumulative process CPU time,
not guaranteed scheduler service. A file-size limit is per file, not a total
storage reservation. The parent deadline requires the selected operating
system to schedule the parent and return its filesystem operations. Neither
Go's heap target nor an address-space ceiling is represented as physical RAM.
This candidate does not promise kernel memory, reserve physical backing,
qualify an arbitrary descendant process tree or tolerate a hostile host that
can modify its private directory or replace the operating system.

The selected worker must be a single process and must not spawn descendants.
Cancellation uses Go's lifecycle-protected process handle, including after
the child has been reaped; no stored numeric process-group ID is signaled.
Executable hashes are caller-selected input at this layer. They still need
an authenticated build and deployment binding before a higher-level compiler
can rely on them. This component does not itself approve those identities.

No operating backend, host, resource profile or policy is selected by adding
this component. The original Registry source compiler, independent native cost
model, complete profile sufficiency and feasibility, private derivation issuer,
original owning caller and retained terminal store must still be joined. A
successful finite worker control does not replace that complete qualification.
The ordinary SDK neither imports this package nor changes its Go runtime.

The new component is not yet wired into the production derivation issuer.
Its source and safe owned runtime tests require independent review and actual
remote CI results before any execution or qualification claim is made.
The native test's compiler preparation uses trusted remote CI tools. Its
cancellation covers the compiler driver only; failed preparation relies on
runner teardown for descendants. It is not production compiler-family cleanup.
