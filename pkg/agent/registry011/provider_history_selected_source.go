package registry011

import (
	"context"
	"strings"
	"sync/atomic"
	"time"
)

// Value DATA for the original source components selected by ONE config. This
// does not contain a derivation, owner or executable callback. The assigned
// prefix, full admitted future, failed reconstruction and successful final
// reader stay distinct. Compiler-own/cleanup/native/backing/lifetime RAM costs
// and whole-profile feasibility must still close before execution is granted.
type providerHistorySelectedSourceCost struct {
	limits              providerHistoryRoleSourceLimits
	clock               providerHistoryClockSelection
	bootstrap           providerHistoryCompiledBootstrapCost
	collection          providerHistoryUniformSourceCost
	catalog             providerHistoryCatalogRequirement
	recovery, finalRead providerHistoryReconstructionSourceCost
	finalization        providerHistoryFinalizationCost
	finalWork           candidateWorkVector
	assigned            providerHistoryStageWorkCost
}

// Bind the role inputs to their original store, sealed manifest and two path
// strings. No caller-supplied role maxima or small test defaults are accepted.
// These are pre-H0 numeric inputs: they neither authenticate the directories
// nor validate/mint the still-missing complete history derivation. The eventual
// enclosing compiler must fund and freeze the same private selection.
func providerHistorySourceLimitsFromConfig(config providerCustodyConfig) (providerHistoryRoleSourceLimits, bool) {
	var empty providerHistoryRoleSourceLimits
	b := config.bounds
	if b.history == nil || b.scan == nil || b.firstAdmission == nil || b.guardian != nil || !b.validStructure() ||
		b.history.derivation != nil || config.attempt != 0 || config.custodyDirectory == "" || config.evidenceDirectory == "" {
		return empty, false
	}
	h, manifest := b.history, b.history.collection
	if !manifest.validSnapshot() || config.capture != manifest.input.capture ||
		h.stages != manifest.counts.groups || h.stages == 0 || h.stages > (1<<20)/3 ||
		!providerHistoryCatalogEntriesSupported(h.catalogEntries) {
		return empty, false
	}
	names, namesOK := providerUpperProduct(h.catalogEntries, 255)
	directories, directoriesOK := providerUpperSum(uint64(len(config.custodyDirectory)), uint64(len(config.evidenceDirectory)))
	if !namesOK || h.catalogNameBytes < names || !directoriesOK || directories > 2<<20 {
		return empty, false
	}
	return providerHistoryRoleSourceLimits{
		legacy: providerHistoryLegacyControlLimits{
			metadata: b.maximumMetadataPerCommit, objectBytes: b.maximumObjectBytes,
			firstObjects: manifest.input.firstRead.objects, firstRaw: manifest.input.firstRead.raw,
			entries: uint64(b.maximumEntries), attempts: b.scan.maximumAttempts, chunks: b.scan.maximumChunks,
		},
		setupRecoveries: uint64(b.maximumSetupRecoveries), catalogEntries: h.catalogEntries,
		catalogNameBytes: h.catalogNameBytes, directoryBytes: directories, fileOverhead: b.fileOverheadBytes,
	}, true
}

// Compile every assigned source role from one unminted config and explicit
// clock snapshot. The eventual factory must freeze that SAME native backend;
// matching value DATA here cannot prove that a backend exists or qualify it.
// Every refusal returns zero, including any earlier positive child. No child
// result, profile or meter is changed to make a later role fit.
func providerHistoryCompileSelectedSource(config providerCustodyConfig, clock providerHistoryClockSelection) (providerHistorySelectedSourceCost, bool) {
	plan, ok := providerHistoryCompileSourcePlan(config, clock)
	if !ok {
		return providerHistorySelectedSourceCost{}, false
	}
	return plan.source, true
}

// The native implementation retains checked DATA through its original session.
// This private interface grants no execution authority and cannot be restored
// from a serialized plan or supplied through a public constructor.
type providerHistoryRetainedSelection interface {
	checkRetainedPlan(*providerHistorySourcePlan) error
}

// Own one exact detached Go config or the original retained native selection
// together with its source rows. Native mode leaves config zero. This private
// pre-H0 plan is not a D, backend qualification or execution permit; the eventual
// factory must separately close resource and authority prerequisites.
type providerHistorySourcePlan struct {
	config   providerCustodyConfig
	source   providerHistorySelectedSourceCost
	retained providerHistoryRetainedSelection
	// Only the owned compiler installs these. The numeric constructor keeps
	// them zero. A finite caller context is a cancellation boundary, not a
	// qualified native lease, resource budget or permission to open a store.
	clock            *providerOwnedClock
	compilerDeadline time.Time
}

func (*providerHistorySourcePlan) UnmarshalJSON([]byte) error { return ErrConfiguration }

func providerHistoryCompileSourcePlan(config providerCustodyConfig, clock providerHistoryClockSelection) (*providerHistorySourcePlan, bool) {
	for _, text := range [...]string{clock.id, clock.epoch, clock.policy} {
		if len(text) > providerHistorySourceFixedTextLimit {
			return nil, false
		}
	}
	frozen, ok := providerHistoryFreezeSourceConfig(config)
	if !ok {
		return nil, false
	}
	return providerHistoryCompileFrozenSourcePlan(frozen, clock)
}

// Capture the original lifetime BEFORE preparing the config. The compiler is
// private, one-use and process-owned; neither a copy nor decoded DATA can start
// another attempt. The original worker must still fund and enforce compiler,
// input preparation and cleanup resources. This object grants no native lease,
// derivation or permission to open a store.
type providerHistorySourceCompiler struct {
	owner    *providerHistorySourceCompiler
	used     *atomic.Bool
	ctx      context.Context
	origin   time.Time
	deadline time.Time
}

func (*providerHistorySourceCompiler) UnmarshalJSON([]byte) error { return ErrConfiguration }

func newProviderHistorySourceCompiler(ctx context.Context) (*providerHistorySourceCompiler, error) {
	return newProviderHistorySourceCompilerAtOrigin(ctx, time.Now())
}

// The fixed preparation child can retain its parent's original origin rather
// than restarting the input/compile window after process bootstrap. This is
// still DATA: the live parent must enforce the actual process lifetime and
// separately fund preparation before any complete execution is permitted.
func newProviderHistorySourceCompilerAtOrigin(ctx context.Context, origin time.Time) (*providerHistorySourceCompiler, error) {
	if ctx == nil {
		return nil, ErrConfiguration
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	deadline, finite := ctx.Deadline()
	if !finite || origin.IsZero() || !deadline.After(origin) {
		return nil, ErrConfiguration
	}
	c := &providerHistorySourceCompiler{used: new(atomic.Bool), ctx: ctx, origin: origin, deadline: deadline}
	c.owner = c
	return c, nil
}

func (c *providerHistorySourceCompiler) contextError() error {
	if err := c.ctx.Err(); err != nil {
		return err
	}
	deadline, finite := c.ctx.Deadline()
	if !finite || deadline != c.deadline {
		return ErrConfiguration
	}
	return nil
}

func (c *providerHistorySourceCompiler) compile(config providerCustodyConfig, maximum uint64, baseUncertaintyMS, readWindowMS int64) (*providerHistorySourcePlan, error) {
	return c.compileWithClockPolicy(config, nil, maximum, baseUncertaintyMS, readWindowMS)
}

// The representation adapter never owns the attempt or changes its lifetime.
// Temporary Go freeze/config and runtime backing remain separate obligations.
func (c *providerHistorySourceCompiler) compileWithClockPolicy(config providerCustodyConfig, reader providerOwnedClockPolicyReader, maximum uint64, baseUncertaintyMS, readWindowMS int64) (*providerHistorySourcePlan, error) {
	if c == nil || c.owner != c || c.used == nil || !c.used.CompareAndSwap(false, true) {
		return nil, ErrConfiguration
	}
	// Consume the attempt before validation or input copying. Failure and
	// cancellation never refund it or replace its original native origin.
	if err := c.contextError(); err != nil {
		return nil, err
	}
	frozen, ok := providerHistoryFreezeSourceConfig(config)
	if !ok {
		return nil, ErrConfiguration
	}
	selectedReader := reader
	if reader == nil {
		reader = providerOwnedClockGoPolicyReader{}
	}
	clock, err := newProviderOwnedClockFromPolicyReader(frozen.policy, reader, maximum, baseUncertaintyMS, readWindowMS, c.origin)
	if err != nil {
		return nil, err
	}
	plan, ok := providerHistoryCompileFrozenSourcePlanWithPolicyReader(c.ctx, frozen, providerHistoryClockSelectionOf(clock), selectedReader)
	if err := c.contextError(); err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConfiguration
	}
	plan.clock, plan.compilerDeadline = clock, c.deadline
	return plan, nil
}

// Existing callers already hold a config. The enclosing Agent preparation
// path must instead create and retain its compiler before decoding that input,
// then call compile on the same object without restarting its lifetime.
func providerHistoryCompileOwnedSourcePlan(ctx context.Context, config providerCustodyConfig, maximum uint64, baseUncertaintyMS, readWindowMS int64) (*providerHistorySourcePlan, error) {
	c, err := newProviderHistorySourceCompiler(ctx)
	if err != nil {
		return nil, err
	}
	return c.compile(config, maximum, baseUncertaintyMS, readWindowMS)
}

// Both constructors pass their own single frozen selection. This helper does
// not copy it again, construct another clock or replace a failed child role.
func providerHistoryCompileFrozenSourcePlan(frozen providerCustodyConfig, clock providerHistoryClockSelection) (*providerHistorySourcePlan, bool) {
	return providerHistoryCompileFrozenSourcePlanContext(context.Background(), frozen, clock)
}

// Cancellation observations belong to this compiler invocation, not to a
// history clock or a new budget. Nil preserves the numeric diagnostic path.
func providerHistoryCompileFrozenSourcePlanContext(ctx context.Context, frozen providerCustodyConfig, clock providerHistoryClockSelection) (*providerHistorySourcePlan, bool) {
	return providerHistoryCompileFrozenSourcePlanWithPolicyReader(ctx, frozen, clock, nil)
}

// Use the same owned policy for clock construction and source selection. The
// nil adapter preserves the original Go diagnostic policy matching semantics.
func providerHistoryCompileFrozenSourcePlanWithPolicyReader(ctx context.Context, frozen providerCustodyConfig, clock providerHistoryClockSelection, reader providerOwnedClockPolicyReader) (*providerHistorySourcePlan, bool) {
	if providerHistoryCompilerCanceled(ctx) {
		return nil, false
	}
	// A complete plan cannot fit ANY supported ledger when this already
	// assigned component alone exceeds its absolute payload ceiling. Refuse
	// before enumerating all prior cuts, rather than spending that compiler
	// work on a selection that no later native qualification could admit.
	if !providerHistorySourcePayloadDomainFits(frozen) {
		return nil, false
	}
	// Preserve the exact native origin/selection values, while retaining no
	// caller-owned string backing through this numeric clock DATA snapshot.
	clock.id, clock.epoch, clock.policy = strings.Clone(clock.id), strings.Clone(clock.epoch), strings.Clone(clock.policy)
	source, ok := providerHistoryCompileSourceSnapshot(ctx, frozen, clock, reader)
	if !ok {
		return nil, false
	}
	return &providerHistorySourcePlan{config: frozen, source: source}, true
}

// This constant-size necessary check uses the existing full32-per-stage SUM,
// not measured occupancy or the smaller request-count realization. The limit
// is shared with the actual ledger validator; no selected cap is increased.
// Passing qualifies neither this selection's other capacities nor any native
// resources, compiler-own funding, D or execution. The component's own numeric
// diagnostic remains available independently of this enclosing plan guard.
func providerHistorySourcePayloadDomainFits(config providerCustodyConfig) bool {
	b := config.bounds
	if b.history == nil || b.methods == nil || b.tls == nil {
		return false
	}
	storage, ok := providerHistoryCollectionUniformStorageUpper(b.history.collection, *b.methods, *b.tls,
		b.maximumObjectBytes, b.maximumMetadataPerCommit, b.fileOverheadBytes)
	return ok && storage.future.payload <= providerLedgerMaximumPayloadBytes
}

// Only the plan constructor passes its own detached selection here. Keep all
// original source-role joins and refusals; no second snapshot or construction
// can change the manifest identity between this calculation and the plan.
func providerHistoryCompileSourceSnapshot(ctx context.Context, config providerCustodyConfig, clock providerHistoryClockSelection, reader providerOwnedClockPolicyReader) (providerHistorySelectedSourceCost, bool) {
	var empty providerHistorySelectedSourceCost
	limits, limitsOK := providerHistorySourceLimitsFromConfig(config)
	if providerHistoryCompilerCanceled(ctx) || !limitsOK || !providerHistorySourceClockMatchesPolicy(config.policy, clock, reader) {
		return empty, false
	}
	bootstrap, bootstrapOK := providerHistoryCompileBootstrapSource(config.bounds, config.capture)
	if !bootstrapOK {
		return empty, false
	}
	b := config.bounds
	collection, collectionOK := providerHistoryCollectionUniformSourceUpperContext(ctx, b.history.collection, *b.methods, *b.tls, b.maximumObjectBytes, limits)
	if !collectionOK || providerHistoryCompilerCanceled(ctx) {
		return empty, false
	}
	catalog, catalogOK := providerHistoryFreshCatalogUpper(collection.storage, b.history.catalogEntries)
	if !catalogOK {
		return empty, false
	}
	recovery, recoveryOK := providerHistoryCompileReconstructionSource(config, true)
	finalRead, finalOK := providerHistoryCompileReconstructionSource(config, false)
	finalization, closingOK := providerHistoryFinalizationUpper(b.history.collection, uint64(len(clock.id)))
	if !recoveryOK || !finalOK || !closingOK || providerHistoryCompilerCanceled(ctx) {
		return empty, false
	}
	out := providerHistorySelectedSourceCost{
		limits: limits, clock: clock, bootstrap: bootstrap, collection: collection, catalog: catalog,
		recovery: recovery, finalRead: finalRead, finalization: finalization,
	}
	var a providerHistoryReadArithmetic
	for coordinate := range out.assigned.work {
		// Reconstruction stops at its first refusal. Closing is entered only
		// after reconstruction succeeds; its paired late-commit extras have
		// already SUMmed inside finalization. Only one phase can leave a new
		// outstanding incoming envelope in this successful-final invocation.
		out.finalWork[coordinate] = a.sum(finalRead.normal[coordinate], finalization.normal[coordinate],
			max(finalRead.incoming[coordinate], finalization.incoming[coordinate]))
		if coordinate == 10 {
			// The two clock events are not in either nonclock child above.
			out.finalWork[coordinate] = a.sum(out.finalWork[coordinate], finalization.clocks)
		}
		// These are distinct PRE-HELD roles. Failed inspection and successful
		// final inspection must SUM here, even though execution chooses one.
		// Keep the whole selected pre-I upper, its separate first P, and the
		// complete I+G*FULLstage future; no initial or future guard is refunded.
		out.assigned.work[coordinate] = a.sum(bootstrap.source.selectedPreI.work[coordinate], bootstrap.source.startup.work[coordinate],
			collection.future.work[coordinate], recovery.envelope[coordinate], out.finalWork[coordinate])
	}
	out.assigned.clocks = a.sum(bootstrap.source.selectedPreI.clocks, bootstrap.source.startup.clocks,
		collection.future.clocks, finalization.clocks)
	if a.failed || providerHistoryCompilerCanceled(ctx) {
		return empty, false
	}
	// Preserve ALL eleven dimensions, including historical proofs in R/F.
	// The live read domain permits separately funded historical verification,
	// but these source DATA still cannot install an executable recovery bound
	// without the complete production derivation and native qualification.
	return out, true
}

// Detach every selected pointer before compiling its costs. Reconstruct the
// collection through its original DATA constructor; copying its owner pointer
// would neither preserve its own seal nor detach it from the caller. All other
// selected pointees contain only value fields. Callers must not concurrently
// mutate input while this snapshot is taken.
//
// This is not an execution grant or a native qualification. The enclosing
// compiler must separately fund construction, copies and its own work before
// a production factory can use the same immutable selection.
func providerHistoryFreezeSourceConfig(config providerCustodyConfig) (providerCustodyConfig, bool) {
	var empty providerCustodyConfig
	if !providerHistorySourceInputTextFits(config) {
		return empty, false
	}
	if _, ok := providerHistorySourceLimitsFromConfig(config); !ok {
		return empty, false
	}
	b := config.bounds
	original := b.history.collection
	manifest, err := newProviderHistoryCollectionManifestCandidate(providerHistoryCloneCollectionText(original.input))
	if err != nil || manifest == nil || manifest.raw != original.raw || manifest.digest != original.digest ||
		manifest.input != original.input || manifest.counts != original.counts {
		return empty, false
	}
	policy, ledger := *config.policy, *b.continuation
	policy.raw, policy.policy = strings.Clone(policy.raw), strings.Clone(policy.policy)
	policy.instance, policy.provider = strings.Clone(policy.instance), strings.Clone(policy.provider)
	methods, certificates := *b.methods, *b.tls
	scan, first, history := *b.scan, *b.firstAdmission, *b.history
	history.program, history.costs = strings.Clone(history.program), strings.Clone(history.costs)
	history.collection = manifest
	config.custodyDirectory, config.evidenceDirectory = strings.Clone(config.custodyDirectory), strings.Clone(config.evidenceDirectory)
	config.anchor, config.expectedTip = strings.Clone(config.anchor), strings.Clone(config.expectedTip)
	config.expectedSetupTip, config.profile = strings.Clone(config.expectedSetupTip), strings.Clone(config.profile)
	config.expectedScanHead = strings.Clone(config.expectedScanHead)
	id := config.identity
	id.Policy, id.Instance, id.Provider = strings.Clone(id.Policy), strings.Clone(id.Instance), strings.Clone(id.Provider)
	id.Namespace, id.SourceID = strings.Clone(id.Namespace), strings.Clone(id.SourceID)
	config.identity = id
	config.policy = &policy
	b.continuation, b.methods, b.tls = &ledger, &methods, &certificates
	b.scan, b.firstAdmission, b.history = &scan, &first, &history
	config.bounds = b
	return config, true
}

// Bounded policy DATA matching only. No native sample or clock construction
// occurs. Parsing and transient copies here belong to the still-open compiler
// cost row, rather than being borrowed from an execution or final-read budget.
func providerHistorySourceClockMatchesPolicy(policy *ProviderPolicyCandidate, clock providerHistoryClockSelection, reader providerOwnedClockPolicyReader) bool {
	if policy == nil || policy.raw == "" || len(policy.raw) > 64<<10 || clock.id == "" || len(clock.id) > 256 ||
		clock.policy != policy.policy || !lowerHash(clock.policy) || !lowerHash(clock.epoch) || clock.origin.IsZero() ||
		clock.window <= 0 || clock.uncertainty <= 0 || clock.maximum == 0 || clock.maximum > 1<<20 {
		return false
	}
	if reader != nil {
		fields, err := reader.readOwnedClockPolicy(policy)
		return err == nil && fields.id == clock.id
	}
	root, ok := contentJSON([]byte(policy.raw)).(map[string]any)
	if !ok {
		return false
	}
	selected, ok := root["clock"].(map[string]any)
	return ok && selected["id"] == clock.id
}
