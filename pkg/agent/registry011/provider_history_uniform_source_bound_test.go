package registry011

import "testing"

func historyAssignedWorkParts() providerHistoryRoleControlParts {
	var p providerHistoryRoleControlParts
	for c := 2; c < len(p.member.work); c++ {
		p.before.work[c], p.preparation.work[c] = uint64(c+1), 2*uint64(c+1)
		p.member.work[c], p.closure.work[c], p.advance.work[c] = 3*uint64(c+1), 4*uint64(c+1), 5*uint64(c+1)
	}
	p.member.work[0], p.member.work[1] = 1, 7
	p.before.clocks, p.preparation.clocks, p.member.clocks, p.closure.clocks, p.advance.clocks = 2, 1, 3, 4, 5
	return p
}

func TestProviderHistoryAssignedWorkKeepsEnteredPreparationAndEveryCoordinate(t *testing.T) {
	parts := historyAssignedWorkParts()
	stage, ok := providerHistoryAssignedStageWorkUpper(parts, 2)
	// B + entered P + 2 full M + C + A: 18 independent units, not16.
	want := providerHistoryStageWorkCost{work: candidateWorkVector{2, 14, 54, 72, 90, 108, 126, 144, 162, 180, 198}, clocks: 18}
	if !ok || stage != want {
		t.Fatal("entered preparation, a work coordinate or a clock row was dropped", stage, ok)
	}
	initial := providerHistoryStageWorkCost{work: candidateWorkVector{2: 11, 10: 17}, clocks: 2}
	first := candidateWorkVector{2: 5, 10: 7}
	future, valid := providerHistoryAssignedFutureWorkUpper(initial, first, stage, 3)
	whole := providerHistoryStageWorkCost{work: candidateWorkVector{6, 42, 178, 216, 270, 324, 378, 432, 486, 540, 618}, clocks: 56}
	if !valid || future != whole {
		t.Fatal("future replaced its full stage SUM by MAX or repeated the initializer", future, valid)
	}
}

func TestProviderHistoryUniformSourceUsesOriginalUniformRoleMaxima(t *testing.T) {
	manifest, methods, tls, _, limits := historyRoleSourceInputs(t)
	limits.legacy.objectBytes = 512 << 10
	original, oldMethods, oldTLS, oldLimits := *manifest, methods, tls, limits
	got, ok := providerHistoryCollectionUniformSourceUpper(manifest, methods, tls, 512<<10, limits)
	if !ok || got.groups != 40 || manifest.counts.requests != 242 {
		t.Fatal("ordinary sealed source-graph precursor", ok)
	}
	if got.stage.work[0] != 32 || got.stage.work[1] != 8388640 || got.stage.clocks != 75 ||
		got.future.work[0] != 1280 || got.future.work[1] != 335545600 || got.future.clocks != 3002 {
		t.Fatal("actual request count, a global MAX, or omitted final-advance room replaced uniform admission", got.stage, got.future)
	}
	if got.storage.future != (providerHistoryStorageCharge{495290120, 480380680, 387973160, 92407520, 2800, 3640}) {
		t.Fatal("the six independent storage domains lost their full admitted SUM")
	}
	roles := [6]providerHistoryStageWorkCost{got.roles.initial, got.roles.before, got.roles.preparation, got.roles.member, got.roles.closure, got.roles.advance}
	for _, prior := range []uint64{0, 3, 39} {
		child, valid := providerHistoryCollectionTransportSourceUpper(manifest, methods, tls, 512<<10, limits, prior, 32, prior != 39)
		if !valid || child.source.files.firstInitialization != got.firstInitialization || child.joined.initial != got.roles.initial {
			t.Fatal("one original initializer changed at a different source cut", prior, valid)
		}
		for role, part := range [6]providerHistoryStageWorkCost{child.joined.initial, child.joined.before, child.joined.preparation, child.joined.member, child.joined.closure, child.joined.advance} {
			for c, units := range part.work {
				if units > roles[role].work[c] {
					t.Fatal("a current source cut exceeds the corresponding original uniform role", prior, role, c)
				}
			}
			if part.clocks > roles[role].clocks {
				t.Fatal("native sample count was inferred from another coordinate", prior, role)
			}
		}
	}
	for c, units := range got.stage.work {
		if got.future.work[c] != 40*units+got.initial.work[c] ||
			got.initial.work[c] < got.roles.initial.work[c]+got.firstInitialization[c] || got.initial.work[c] < got.initialGuard[c] {
			t.Fatal("a complete initializer or full admitted stage was omitted or repeated", c)
		}
	}
	if *manifest != original || methods != oldMethods || tls != oldTLS || limits != oldLimits {
		t.Fatal("numeric compilation changed the sealed source selection")
	}
}

func TestProviderHistoryAssignedInitializationPreservesEveryGuardCoordinate(t *testing.T) {
	// Independent inert rows exercise both sides of the initializer maximum in
	// every domain. These values grant no store, sample, source or reservation.
	assigned := providerHistoryStageWorkCost{work: candidateWorkVector{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, clocks: 2}
	first := candidateWorkVector{11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	guard := candidateWorkVector{0, 13, 11, 14, 10, 15, 9, 16, 8, 17, 7}
	want := providerHistoryStageWorkCost{work: candidateWorkVector{12, 13, 12, 14, 12, 15, 12, 16, 12, 17, 12}, clocks: 2}
	got, ok := providerHistoryAssignedInitializationUpper(assigned, first, guard)
	if !ok || got != want {
		t.Fatal("initializer dropped a guard coordinate, doubled the same work or invented clocks", got, ok)
	}
	stage := providerHistoryStageWorkCost{work: candidateWorkVector{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, clocks: 3}
	future, futureOK := providerHistoryAssignedFutureWorkUpper(got, candidateWorkVector{}, stage, 3)
	whole := providerHistoryStageWorkCost{work: candidateWorkVector{18, 22, 24, 29, 30, 36, 36, 43, 42, 50, 48}, clocks: 11}
	if !futureOK || future != whole {
		t.Fatal("full future SUM borrowed initializer room or added firstRead a second time", future, futureOK)
	}
	assigned.work[10], first[10] = ^uint64(0), 1
	if partial, valid := providerHistoryAssignedInitializationUpper(assigned, first, guard); valid || partial != (providerHistoryStageWorkCost{}) {
		t.Fatal("late initializer overflow returned earlier coordinates", partial, valid)
	}
}

func TestProviderHistoryUniformSourceFundsSelectedInitializerBeforeEveryStage(t *testing.T) {
	manifest, methods, tls, _, limits := historyRoleSourceInputs(t)
	limits.legacy.objectBytes = 512 << 10
	got, ok := providerHistoryCollectionUniformSourceUpper(manifest, methods, tls, 512<<10, limits)
	// The real64KiB legacy guard is131072 decoded bytes and34364198276
	// logical units. Add TWO SELECTED64MiB/2^52 firstRead sublimits. The
	// smaller derived firstRead component cannot replace that incoming guard.
	want := candidateWorkVector{2: 134348800, 10: 9007233618939268}
	if !ok || got.initialGuard != want || got.initial.work != want || got.initial.clocks != 2 {
		t.Fatal("selected firstRead allowance or the original constructor guard was narrowed", got.initialGuard, got.initial, ok)
	}
	actualGuard, valid := providerHistoryGroupInitializationForProfileUpper(providerIntegrationBounds{
		maximumMetadataPerCommit: 8192, history: &providerHistoryProgramBounds{collection: manifest},
	})
	if !valid || actualGuard != want {
		t.Fatal("independent initializer literals disagree with the actual unchanged constructor guard", actualGuard, valid)
	}
	for _, row := range []struct {
		name       string
		coordinate int
	}{{"decoded", 2}, {"logical", 10}} {
		t.Run(row.name, func(t *testing.T) {
			in := manifest.input
			in.firstRead.work[row.coordinate] = ^uint64(0)
			selected := historyCollectionManifest(t, in)
			if partial, valid := providerHistoryCollectionUniformSourceUpper(selected, methods, tls, 512<<10, limits); valid || partial != (providerHistoryUniformSourceCost{}) {
				t.Fatal("selected initializer guard overflow escaped as a partial future", partial, valid)
			}
		})
	}
}

func TestProviderHistoryUniformSourceKeepsOriginalGlobalWireAdmission(t *testing.T) {
	in := historyCollectionInput(t)
	in.native, in.effects, in.capture.WireBytes = 0, 0, 16<<20
	manifest := historyCollectionManifest(t, in)
	_, methods, tls, _, limits := historyRoleSourceInputs(t)
	limits.legacy.objectBytes = 512 << 10
	methods.gas.wire = 16 << 20 // Inactive in this original selection.
	active, activeOK := providerHistoryCollectionTransportRoleUpper(manifest, methods, tls, 512<<10, 8192, 4096)
	got, ok := providerHistoryCollectionUniformSourceUpper(manifest, methods, tls, 512<<10, limits)
	if !activeOK || !ok || active.member[1] != 131073 || got.roles.member.work[1] != 16777217 || got.stage.work[1] != 536870944 {
		t.Fatal("active capture narrowed the unchanged GLOBAL admitted wire row", activeOK, ok)
	}
	if got.future.work[1] != got.groups*536870944 || got.future.work[0] != got.groups*32 {
		t.Fatal("uniform future permission was replaced by observed/actual member use")
	}
}

func TestProviderHistoryAssignedWorkRefusesInvalidAndLateOverflowData(t *testing.T) {
	base := historyAssignedWorkParts()
	for _, name := range []string{"zero-count", "wide-count", "before-request", "before-wire", "preparation-request", "preparation-wire", "closure-request", "closure-wire", "advance-request", "advance-wire", "member-request", "member-wire", "preparation-work", "preparation-clock", "late-work", "late-clock"} {
		t.Run(name, func(t *testing.T) {
			p, count := base, 2
			switch name {
			case "zero-count":
				count = 0
			case "wide-count":
				count = 33
			case "before-request":
				p.before.work[0] = 1
			case "before-wire":
				p.before.work[1] = 1
			case "preparation-request":
				p.preparation.work[0] = 1
			case "preparation-wire":
				p.preparation.work[1] = 1
			case "closure-request":
				p.closure.work[0] = 1
			case "closure-wire":
				p.closure.work[1] = 1
			case "advance-request":
				p.advance.work[0] = 1
			case "advance-wire":
				p.advance.work[1] = 1
			case "member-request":
				p.member.work[0] = 0
			case "member-wire":
				p.member.work[1] = 0
			case "preparation-work":
				p.preparation.work[10] = p.member.work[10] + 1
			case "preparation-clock":
				p.preparation.clocks = p.member.clocks + 1
			case "late-work":
				p.advance.work[10] = ^uint64(0)
			case "late-clock":
				p.advance.clocks = ^uint64(0)
			}
			if got, ok := providerHistoryAssignedStageWorkUpper(p, count); ok || got != (providerHistoryStageWorkCost{}) {
				t.Fatal("partial stage escaped an invalid cut or late overflow", name, ok)
			}
		})
	}
	for _, name := range []string{"zero-groups", "wide-groups", "future-product", "future-initial", "future-clock"} {
		t.Run(name, func(t *testing.T) {
			initial, first, stage, groups := providerHistoryStageWorkCost{}, candidateWorkVector{}, providerHistoryStageWorkCost{work: candidateWorkVector{0: 1, 10: 1}, clocks: 1}, uint64(40)
			switch name {
			case "zero-groups":
				groups = 0
			case "wide-groups":
				groups = (1<<20)/3 + 1
			case "future-product":
				stage.work[10] = ^uint64(0)/40 + 1
			case "future-initial":
				first[10] = ^uint64(0)
			case "future-clock":
				initial.clocks = ^uint64(0)
			}
			if got, ok := providerHistoryAssignedFutureWorkUpper(initial, first, stage, groups); ok || got != (providerHistoryStageWorkCost{}) {
				t.Fatal("partial future escaped a late checked SUM", name, ok)
			}
		})
	}
}

func TestProviderHistoryUniformSourceRefusesDetachedAndIncompleteSelections(t *testing.T) {
	manifest, methods, tls, _, limits := historyRoleSourceInputs(t)
	limits.legacy.objectBytes = 512 << 10
	copy := *manifest
	for _, name := range []string{"nil", "copied", "missing-method", "missing-overhead", "short-failure", "short-catalog", "different-first-read", "late-storage-overflow"} {
		t.Run(name, func(t *testing.T) {
			m, b, l, maximum := manifest, methods, limits, uint64(512<<10)
			switch name {
			case "nil":
				m = nil
			case "copied":
				m = &copy
			case "missing-method":
				b.scalar.body = 0
			case "missing-overhead":
				l.fileOverhead = 0
			case "short-failure":
				maximum, l.legacy.objectBytes = 278572, 278572
			case "short-catalog":
				l.catalogEntries = 1
			case "different-first-read":
				l.legacy.firstRaw++
			case "late-storage-overflow":
				l.fileOverhead = ^uint64(0)
			}
			if got, ok := providerHistoryCollectionUniformSourceUpper(m, b, tls, maximum, l); ok || got != (providerHistoryUniformSourceCost{}) {
				t.Fatal("detached/partial child became a whole source quote", name, ok)
			}
		})
	}
}
