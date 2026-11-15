package registry011

import "testing"

func TestProviderHistorySourcePlanRejectsImpossiblePayloadBeforeGroupEnumeration(t *testing.T) {
	config := historySelectedSourceConfig(t)
	in := config.bounds.history.collection.input
	in.maximumSelected = 12067934
	in.blocks = in.maximumSelected - in.journal.Deployment.Number + 1
	in.native, in.effects, in.versions = 0, 16, 16
	in.windowBlocks, in.contentBlocks = 4096, 10
	in.presentStops = 1 + (in.blocks-1)/in.windowBlocks
	in.journal.Observation.MaxAncestryHeaders = in.blocks - 1
	manifest := historyCollectionManifest(t, in)
	config.bounds.history.collection, config.bounds.history.stages = manifest, manifest.counts.groups
	if in.blocks != 93265 || !config.bounds.validStructure() {
		t.Fatal("full historical interval lost its positive structural prerequisite")
	}
	if _, ok := providerHistoryFreezeSourceConfig(config); !ok {
		t.Fatal("input snapshot refused before the new payload-domain guard")
	}
	before, historyBefore, manifestBefore := config, *config.bounds.history, *manifest
	if providerHistorySourcePayloadDomainFits(config) {
		t.Fatal("full admitted storage must not fit any supported payload ceiling")
	}
	plan, ok := providerHistoryCompileSourcePlan(config, historySelectedSourceClock(config))
	if ok || plan != nil || config != before || *config.bounds.history != historyBefore || *manifest != manifestBefore ||
		config.bounds.history.derivation != nil || config.bounds.valid() {
		t.Fatal("early refusal changed input or returned partial authority")
	}
}

func TestProviderHistorySourcePayloadDomainRetainsSmallUnqualifiedPlan(t *testing.T) {
	config := historySelectedSourceConfig(t)
	if !providerHistorySourcePayloadDomainFits(config) {
		t.Fatal("small numeric selection failed the necessary payload check")
	}
	plan, ok := providerHistoryCompileSourcePlan(config, historySelectedSourceClock(config))
	if !ok || plan == nil || plan.config.bounds.history.derivation != nil || plan.config.bounds.valid() {
		t.Fatal("positive numeric selection was lost or converted into authority")
	}
}

func TestProviderHistorySourcePayloadDomainIncludesTheExactAbsoluteBoundary(t *testing.T) {
	for _, test := range []struct {
		name    string
		object  uint64
		payload uint64
		fits    bool
	}{
		{"below", 12451822, (16 << 30) - 1024, true},
		{"equal", 12451823, 16 << 30, true},
		{"above", 12451824, (16 << 30) + 1024, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := historySelectedSourceConfig(t)
			in := config.bounds.history.collection.input
			in.blocks, in.native, in.effects, in.versions = 3850, 0, 0, 0
			in.maximumSelected = in.journal.Deployment.Number + in.blocks - 1
			in.windowBlocks, in.contentBlocks, in.presentStops = 4096, 10, 1
			in.journal.Observation.MaxAncestryHeaders = in.blocks - 1
			in.capture.WireBytes = (256 << 10) + 1
			manifest := historyCollectionManifest(t, in)
			config.bounds.history.collection, config.bounds.history.stages = manifest, manifest.counts.groups
			config.bounds.maximumObjectBytes, config.capture = test.object, in.capture
			methods := *config.bounds.methods
			for _, limit := range []*providerMethodLimit{&methods.scalar, &methods.header, &methods.code, &methods.state,
				&methods.block, &methods.receipts, &methods.gas, &methods.execution} {
				limit.wire = in.capture.WireBytes
			}
			config.bounds.methods = &methods
			storage, ok := providerHistoryCollectionUniformStorageUpper(manifest, *config.bounds.methods, *config.bounds.tls,
				config.bounds.maximumObjectBytes, config.bounds.maximumMetadataPerCommit, config.bounds.fileOverheadBytes)
			// Independent literal vectors exercise the actual storage component
			// on both sides of the shared validator's unchanged16GiB boundary.
			// The arithmetic below invokes no group enumeration, store open,
			// RPC or budget-clock sample. Shared fixture preparation above
			// creates temporary originals and an unused native clock.
			if !ok || manifest.counts.groups != 512 || storage.future.payload != test.payload ||
				providerHistorySourcePayloadDomainFits(config) != test.fits {
				t.Fatal("payload-domain boundary differs from the independent literal", ok, manifest.counts.groups, storage.future.payload)
			}
		})
	}
}

func TestProviderHistorySourcePayloadDomainRejectsIncompleteStorageInputs(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*providerCustodyConfig)
	}{
		{"zero-config", func(c *providerCustodyConfig) { *c = providerCustodyConfig{} }},
		{"missing-manifest", func(c *providerCustodyConfig) { c.bounds.history.collection = nil }},
		{"missing-methods", func(c *providerCustodyConfig) { c.bounds.methods = nil }},
		{"missing-tls", func(c *providerCustodyConfig) { c.bounds.tls = nil }},
		{"foreign-manifest", func(c *providerCustodyConfig) {
			copied := *c.bounds.history.collection
			c.bounds.history.collection = &copied
		}},
		{"zero-object-bound", func(c *providerCustodyConfig) { c.bounds.maximumObjectBytes = 0 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := historySelectedSourceConfig(t)
			test.change(&config)
			if providerHistorySourcePayloadDomainFits(config) {
				t.Fatal("incomplete storage inputs passed the necessary domain check")
			}
		})
	}
}
