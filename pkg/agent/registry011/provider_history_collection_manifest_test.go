package registry011

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// Only explicit offline DATA is selected before the genuine owned fixture's H0.
// Matching the pinned deployment artifact is not runtime code qualification.
func historyCollectionInput(t *testing.T) providerHistoryCollectionInput {
	t.Helper()
	in := providerHistoryCollectionInput{journal: journalTestConfig(t, replayTestInput(t)),
		encoding: EthereumJSONBlocks, gas: SepoliaAmsterdamRootExecutionGasProfile,
		maximumSelected: sepoliaDeploymentBlock + 12, blocks: 13, native: 13,
		effects: 4, versions: 4, presentStops: 4, windowBlocks: 4, contentBlocks: 2,
		// Explicit UNIT-only read row, selected before H0. These numbers do
		// not certify sufficient whole-source costs or operating limits.
		firstRead: providerHistoryFirstPReadBounds{objects: 64, raw: 1 << 20, work: candidateWorkVector{2: 64 << 20, 10: 1 << 52}}, traceTimeout: time.Second}
	identity, _, err := nodeIdentityManifest(nodeIdentityJSON, nodeABIJSON)
	if err != nil {
		t.Fatal("pinned artifact DATA", err)
	}
	copy(in.codes[:], identity.Contracts)
	in.journal.Deployment = FinalizedHeader{identity.Deployment.BlockHash, admissionTestDigest("d"), sepoliaDeploymentBlock, historyFirstFinalizedHeader().Timestamp - 10}
	_, in.capture = providerFrontierBoundsFixture()
	// Finite pre-H0 UNIT allowance for the complete original-prefix replays.
	in.capture.Timeout = 2 * time.Minute
	return in
}

func historyCollectionManifest(t *testing.T, in providerHistoryCollectionInput) *providerHistoryCollectionManifest {
	t.Helper()
	m, err := newProviderHistoryCollectionManifestCandidate(in)
	if err != nil || m == nil || !m.validSnapshot() {
		t.Fatal("explicit collection manifest DATA", err)
	}
	return m
}

func TestProviderHistoryCollectionManifestUsesHandwrittenProspectiveGrammarCounts(t *testing.T) {
	for _, tc := range []struct {
		name                                      string
		blocks, window, content, native, t, v, gp uint64
		want                                      providerHistoryCollectionCounts
	}{
		{"one-absent", 1, 1, 1, 0, 0, 0, 0, providerHistoryCollectionCounts{0, 1, 1, 28, 4, 96}},
		{"two-native", 2, 1, 1, 2, 0, 0, 0, providerHistoryCollectionCounts{1, 2, 2, 84, 11, 278}},
		{"short-last-window", 13, 4, 2, 13, 4, 4, 4, providerHistoryCollectionCounts{3, 4, 7, 242, 40, 884}},
		{"whole-hypothetical-interval", 38420, 4096, 10, 38420, 4096, 4096, 10, providerHistoryCollectionCounts{10, 10, 3846, 162391, 9226, 417042}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := providerHistoryCollectionInput{blocks: tc.blocks, native: tc.native, effects: tc.t, versions: tc.v, presentStops: tc.gp, windowBlocks: tc.window, contentBlocks: tc.content}
			got, ok := providerHistoryCollectionMaximumCounts(in)
			if !ok || got != tc.want {
				t.Fatal("prospective grammar count differs from independent literal", got, ok)
			}
		})
	}
}

func TestProviderHistoryCollectionCountsRefuseInvalidPartitionsAndOverflow(t *testing.T) {
	for _, mode := range []string{"zero-blocks", "zero-window", "large-window", "zero-content", "large-content", "native", "present", "windows-overflow", "requests-overflow", "clocks-overflow"} {
		t.Run(mode, func(t *testing.T) {
			in := providerHistoryCollectionInput{blocks: 1, windowBlocks: 1, contentBlocks: 1}
			switch mode {
			case "zero-blocks":
				in.blocks = 0
			case "zero-window":
				in.windowBlocks = 0
			case "large-window":
				in.windowBlocks = 4097
			case "zero-content":
				in.contentBlocks = 0
			case "large-content":
				in.contentBlocks = 11
			case "native":
				in.native = 2
			case "present":
				in.presentStops = 2
			case "windows-overflow":
				in.blocks = ^uint64(0)
			case "requests-overflow":
				in.effects = ^uint64(0)
			case "clocks-overflow":
				in.effects = 1 << 63
			}
			if got, ok := providerHistoryCollectionMaximumCounts(in); ok || got != (providerHistoryCollectionCounts{}) {
				t.Fatal("invalid prospective count became funding", mode, got)
			}
		})
	}
}

func TestProviderHistoryCollectionManifestRejectsUnselectedIdentityAndImplicitFallback(t *testing.T) {
	in := historyCollectionInput(t)
	m := historyCollectionManifest(t, in)
	copyInput := in
	copyInput.codes[0].SHA256 = "changed returned DATA"
	if m.input != in || m.input == copyInput || json.Unmarshal([]byte(`{}`), &providerHistoryCollectionManifest{}) == nil {
		t.Fatal("caller DATA or JSON supplied a live original selection")
	}
	for _, mode := range []string{"code", "deployment", "height", "span", "encoding", "gas", "capture", "subject", "window", "versions", "ancestry", "batches", "effect-window", "version-window"} {
		t.Run(mode, func(t *testing.T) {
			changed := in
			switch mode {
			case "code":
				changed.codes[0].SHA256 = "wrong pinned code DATA"
			case "deployment":
				changed.journal.Deployment.Hash = admissionTestDigest("e")
			case "height":
				changed.maximumSelected = sepoliaDeploymentBlock - 1
			case "span":
				changed.blocks--
			case "encoding":
				changed.encoding = ""
			case "gas":
				changed.gas = "automatic"
			case "capture":
				changed.capture.JSONDepth = 0
			case "subject":
				changed.journal.Agent = ""
			case "window":
				changed.windowBlocks = uint64(changed.journal.Replay.MaxBlocks) + 1
			case "versions":
				changed.versions = 1<<20 + 1
			case "ancestry":
				changed.journal.Observation.MaxAncestryHeaders = changed.blocks - 2
			case "batches":
				changed.journal.MaxBatches = 3
			case "effect-window":
				changed.effects = 4*uint64(changed.journal.Replay.MaxTransactions) + 1
			case "version-window":
				changed.versions = 4*uint64(changed.journal.Replay.History.MaxItems) + 1
			}
			if out, err := newProviderHistoryCollectionManifestCandidate(changed); err == nil || out != nil {
				t.Fatal("invalid manifest gained a default or fallback", mode)
			}
		})
	}
	for _, mode := range []string{"copy", "input", "raw", "digest", "counts"} {
		t.Run(mode, func(t *testing.T) {
			changed := *m
			switch mode {
			case "input":
				changed.owner = &changed
				changed.input.maximumSelected++
			case "raw":
				changed.owner = &changed
				changed.raw += " "
			case "digest":
				changed.owner = &changed
				changed.digest = admissionTestDigest("f")
			case "counts":
				changed.owner = &changed
				changed.counts.requests--
			}
			if changed.validSnapshot() {
				t.Fatal("changed or copied original selection was accepted", mode)
			}
		})
	}
}

func TestProviderHistoryCollectionSelectionKeepsInclusiveAndExactDeploymentBoundaries(t *testing.T) {
	m := historyCollectionManifest(t, historyCollectionInput(t))
	for _, mode := range []string{"maximum", "deployment", "above", "below", "wrong-deployment-parent", "old-time", "nil-meter", "closed-meter", "no-budget"} {
		t.Run(mode, func(t *testing.T) {
			h, meter := historyFirstFinalizedHeader(), providerReplayUnitMeter()
			switch mode {
			case "deployment":
				h = m.input.journal.Deployment
			case "above":
				h.Number++
			case "below":
				h.Number = sepoliaDeploymentBlock - 1
			case "wrong-deployment-parent":
				h = m.input.journal.Deployment
				h.ParentHash = admissionTestDigest("e")
			case "old-time":
				h.Timestamp = m.input.journal.Deployment.Timestamp - 1
			case "nil-meter":
				meter = nil
			case "closed-meter":
				meter.closed = true
			case "no-budget":
				meter.upper = candidateWorkVector{}
			}
			got := m.selectedObserved(h, meter)
			if got != (mode == "maximum" || mode == "deployment") {
				t.Fatal("selected original moved outside the fixed inclusive interval", mode)
			}
		})
	}
}

func TestProviderHistoryCollectionManifestBindsOriginalPreH0AndBothFreshSelections(t *testing.T) {
	m := historyCollectionManifest(t, historyCollectionInput(t))
	o, r, refs := historyFirstFinalizedFixtureWithManifest(t, historyFirstFinalizedResponse(t), m)
	s, p := o.store, o.parent
	old, clocks, held, before := o.meter.actual, s.clock.read, s.stored, providerHistoryProvisionedOriginalBytes(t, s.config)
	if !p.collectionMatchesLocked() || p.collectionInput != m.input || p.bounds.collection != m || !r.observerShapeLocked(r.meter) {
		t.Fatal("original pre-H0 manifest was not copied into both roles")
	}
	for pass := range 2 {
		historyRecoveryModelPrefix(t, r, pass, refs)
		model, err := r.replayInitialModelCandidate(context.Background(), pass)
		if err != nil {
			t.Fatal("genuine original model", pass, err)
		}
		if selected, err := model.selectFirstFinalizedCandidate(context.Background()); err != nil || selected != historyFirstFinalizedHeader() {
			t.Fatal("original manifest-bound header", pass, err)
		}
	}
	if o.meter.actual != old || s.clock.read != clocks || s.stored != held || s.activeMeter.Load() != nil ||
		!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("manifest selection renewed execution, time or storage")
	}
	// Even coordinated edits of private manifest DATA cannot change the value
	// copied into the original pre-H0 program owner. No file or traffic changes.
	m.input.maximumSelected++
	m.sealed = m.input
	if p.collectionMatchesLocked() || r.observerShapeLocked(r.meter) {
		t.Fatal("mutable manifest replaced the original owner selection")
	}
}

func TestProviderHistoryCollectionRefusesOriginalHeaderAboveMaximumAfterOwnedFirstTransport(t *testing.T) {
	in := historyCollectionInput(t)
	in.maximumSelected--
	in.blocks--
	in.native--
	in.presentStops = 3
	m := historyCollectionManifest(t, in)
	s := historyFirstFinalizedStore(t, historyFirstFinalizedResponse(t), m)
	clocks, held, pin := s.clock.read, s.stored, s.frontierFiles.pin
	before := providerHistoryProvisionedOriginalBytes(t, s.config)
	if got, err := newProviderHistoryGroupOwnerCandidate(s); err == nil || got != nil || !s.failed || !s.history.failed ||
		s.history.group == nil || !s.history.group.failed || !s.history.group.meter.closed || s.history.group.last != nil ||
		s.clock.read != clocks+1 || s.stored != held || s.frontierFiles.pin != pin || s.activeMeter.Load() != nil ||
		!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("larger first header initialized a collection or changed originals", err)
	}
}

func TestProviderHistoryCollectionManifestRefusesUnconnectedNextTableBeforeEffects(t *testing.T) {
	m := historyCollectionManifest(t, historyCollectionInput(t))
	s := historyFirstFinalizedStore(t, historyFirstFinalizedResponse(t), m)
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil || !o.parent.collectionMatchesLocked() || !o.parent.matchesLocked() {
		t.Fatal("genuine healthy original selection", err)
	}
	work, clocks, held, pin := o.meter.actual, s.clock.read, s.stored, o.pin
	before := providerHistoryProvisionedOriginalBytes(t, s.config)
	upper := providerHistoryGroupUpper{physical: 8 << 20, retainedRaw: 4 << 20, payload: 128 << 10, metadata: 2 << 20, objects: 8, files: 64, clocks: 16}
	upper.execution[0], upper.execution[1], upper.execution[10] = 1, 1, 128
	upper.custody[2], upper.custody[9], upper.custody[10] = 16<<20, 16<<20, 1<<50
	if ack, err := o.retainAdmissionCandidate(context.Background(), "selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper); err == nil || ack != nil ||
		o.last != nil || o.admission != nil || o.order != nil || o.window || o.phase != "prepared" ||
		o.meter.actual != work || s.clock.read != clocks || s.stored != held || o.pin != pin || s.activeMeter.Load() != nil ||
		o.failed || o.meter.closed || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("manifest DATA admitted a caller table before original grammar connection", err)
	}
}

func TestProviderHistoryCollectionCopiesCountsIntoOriginalOwnerBeforeH0(t *testing.T) {
	m := historyCollectionManifest(t, historyCollectionInput(t))
	s := historyFirstFinalizedStore(t, historyFirstFinalizedResponse(t), m)
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil || !o.parent.collectionMatchesLocked() || !o.parent.matchesLocked() {
		t.Fatal("original selected healthy owner", err)
	}
	p, r := o.parent, o.parent.recovery
	counts, raw, digest := p.collectionCounts, m.raw, m.digest
	work, clocks, held := o.meter.actual, s.clock.read, s.stored
	before := providerHistoryProvisionedOriginalBytes(t, s.config)
	m.counts.requests--
	m.sealedCounts = m.counts // Coordinated private DATA fault only; no original file changes.
	if !m.validSnapshot() || p.collectionCounts != counts || m.raw != raw || m.digest != digest ||
		p.collectionMatchesLocked() || p.matchesLocked() || r.observerShapeLocked(r.meter) ||
		o.meter.actual != work || s.clock.read != clocks || s.stored != held ||
		!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("coordinated count edit replaced the original pre-H0 reservation inputs")
	}
}

func TestProviderHistoryCollectionSeparatesSharedClaimsFromWholeTransactionAndVersionLimits(t *testing.T) {
	for _, tc := range []struct {
		agent             string
		versions, present uint64
	}{{replayTestAgent, 11, 4}, {"other.selected.subject", 0, 0}} {
		t.Run(tc.agent, func(t *testing.T) {
			in := historyCollectionInput(t)
			in.journal.Agent, in.journal.MaxClaims = tc.agent, 1
			in.effects, in.versions, in.presentStops = 12, tc.versions, tc.present
			m := historyCollectionManifest(t, in)
			if m.input != in || m.input.effects <= uint64(m.input.journal.MaxClaims) || m.input.versions != tc.versions {
				t.Fatal("one shared claim incorrectly constrained whole transactions or subject versions")
			}
		})
	}
}
