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

An explicit positive `ChildCleanupReserve` uses part of that same reservation
for a cooperative SIGTERM stop. It must leave a positive parent cleanup
interval. A child that handles the request can finish its terminal writes;
an early cancellation is still capped by this child interval, and the fixed
hard-stop deadline is set before bootstrap. Both soft and hard termination use
the owning Go Process handle, including after exit. No numeric PID is signaled.
Zero retains immediate termination. Grace never extends the original lifetime.
`CleanupComplete` reports only removal of the parent's executable copies. A
normal child exit or that flag does not authenticate a child journal, make
incomplete Registry custody complete, or certify that Go defers ran after a
forced stop or host loss.

An explicit `TerminalEvidence` configuration adds a separate inherited file
descriptor (4) to this same invocation. The parent creates a fresh private file
relative to an already opened private retention root before startup. Creation
and directory-entry sync refer to that same root even if its name moves. A
changed original parent/file name is refused at readback rather than reported
as a usable retained path. The launcher uses
descriptor 3 only for readiness and leaves descriptor 4 open for the child.
After Wait or failed startup, the parent first fixes the action's outcome,
then syncs, reads a bounded prefix, syncs
the directory entry, and closes both handles within the original cleanup
lifetime. It never deletes that file when removing executable copies. An
ordinary output refusal, cancellation, overflow or later executable cleanup
failure keeps the existing terminal bytes; a failed readback leaves its path
and reports incomplete status. Empty bytes are empty DATA, not a terminal
journal. No previous attempt is restored or retried.

`EvidenceResult.ReadbackComplete` describes only this raw-file readback. It
does not mean that the child's journal has a valid terminal record, that a
Registry result can be published, or that custody is complete. The independent
store selection, journal interpretation, complete compiler/native resource
funding, retention ownership, authenticated build and original Agent caller
still need their full joined qualification. `MaximumBytes` bounds parent
readback memory/I/O; the actual child file can grow to the separate native
per-file ceiling. That ceiling is not a total-storage reservation. Sync and
close depend on the selected trusted operating system and storage semantics;
no readback completion is reported if those operations fail or exceed the
original cleanup lifetime. A hostile host or host loss is not repaired by
this file mechanism. Adding the option selects no operating retention policy.

The Darwin launcher lowers and reads back the child process's CPU-time,
per-file-size, descriptor and core-dump limits before executing the selected
worker. The parent owns cancellation and bounded input, result, diagnostic and
readiness I/O. The launcher closes its readiness descriptor on exec. A failed
copy, launch, command, limit acknowledgment, output boundary or cleanup prevents
successful result publication. Already completed diagnostic data may remain.

These controls have specific scopes. CPU time is cumulative process CPU time,
not guaranteed scheduler service. A file-size limit is per file, not a total
storage reservation. The parent deadline requires the selected operating
system to schedule the parent and return its native operations, including
filesystem and process-control calls. Neither
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
