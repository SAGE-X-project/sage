package registry011

import (
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

func historySourceBackedText(text string) string {
	const padding = 4096
	whole := strings.Repeat("x", padding) + text + strings.Repeat("y", padding)
	return whole[padding : padding+len(text)]
}

// Read-only storage identity checks on creator-owned UNIT strings. No unsafe
// write, native resource promise, hostile traffic or operating input is used.
// The walk follows the original manifest's owner cycle only once.
func historyAssertDetachedText(t *testing.T, before, after reflect.Value, seen map[[2]uintptr]bool) {
	t.Helper()
	if before.Type() != after.Type() {
		t.Fatal("input type changed while detaching text")
	}
	switch before.Kind() {
	case reflect.Ptr:
		if before.IsNil() || after.IsNil() {
			if before.IsNil() != after.IsNil() {
				t.Fatal("input pointer shape changed")
			}
			return
		}
		pair := [2]uintptr{before.Pointer(), after.Pointer()}
		if seen[pair] {
			return
		}
		seen[pair] = true
		historyAssertDetachedText(t, before.Elem(), after.Elem(), seen)
	case reflect.Struct:
		for i := range before.NumField() {
			historyAssertDetachedText(t, before.Field(i), after.Field(i), seen)
		}
	case reflect.Array:
		for i := range before.Len() {
			historyAssertDetachedText(t, before.Index(i), after.Index(i), seen)
		}
	case reflect.String:
		old, next := before.String(), after.String()
		if old != next || old != "" && unsafe.StringData(old) == unsafe.StringData(next) {
			t.Fatal("copied input changed text or retained caller backing", before.Type())
		}
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		// Existing whole-value tests check numeric equality. This walk checks
		// the complete retained text graph, not a native heap allocation bound.
	default:
		t.Fatal("unreviewed input storage kind", before.Kind())
	}
}

func TestProviderHistorySourceInputCopyDoesNotKeepCallerStringBacking(t *testing.T) {
	config := historySelectedSourceConfig(t)
	config.custodyDirectory = historySourceBackedText(config.custodyDirectory)
	config.evidenceDirectory = historySourceBackedText(config.evidenceDirectory)
	digest := historySourceBackedText(admissionTestDigest("a"))
	config.anchor, config.expectedTip, config.expectedSetupTip, config.profile, config.expectedScanHead = digest, digest, digest, digest, digest
	config.identity = ProviderCandidateIdentity{Policy: digest, Instance: digest, Provider: digest, Namespace: digest, SourceID: digest}
	config.bounds.history.program, config.bounds.history.costs = digest, digest
	config.policy.raw = historySourceBackedText(config.policy.raw)
	// Equal string values keep the original seal valid. Only the backing
	// allocation changes; no new owner, seal value or authority is invented.
	manifest := config.bounds.history.collection
	manifest.input.journal.Registry = historySourceBackedText(manifest.input.journal.Registry)
	if !manifest.validSnapshot() || !providerHistorySourceInputTextFits(config) {
		t.Fatal("equal backed text lost the original positive prerequisite")
	}
	before, policy, history, original := config, *config.policy, *config.bounds.history, *manifest
	frozen, ok := providerHistoryFreezeSourceConfig(config)
	if !ok || !frozen.bounds.history.collection.validSnapshot() || frozen.bounds.history.derivation != nil || frozen.bounds.valid() {
		t.Fatal("bounded copy lost its genuine manifest or created execution authority")
	}
	historyAssertDetachedText(t, reflect.ValueOf(config), reflect.ValueOf(frozen), make(map[[2]uintptr]bool))
	if config != before || *config.policy != policy || *config.bounds.history != history || *manifest != original {
		t.Fatal("copy changed the caller's original values")
	}
}

func TestProviderHistorySourcePlanRetainsDetachedClockTextAndExactOrigin(t *testing.T) {
	config := historySelectedSourceConfig(t)
	clock := historySelectedSourceClock(config)
	clock.id, clock.epoch, clock.policy = historySourceBackedText(clock.id), historySourceBackedText(clock.epoch), historySourceBackedText(clock.policy)
	plan, ok := providerHistoryCompileSourcePlan(config, clock)
	if !ok || plan == nil || plan.source.clock != clock || plan.config.bounds.history.derivation != nil {
		t.Fatal("owned numeric plan changed the exact clock selection or created authority")
	}
	for i, text := range [...]string{clock.id, clock.epoch, clock.policy} {
		next := [...]string{plan.source.clock.id, plan.source.clock.epoch, plan.source.clock.policy}[i]
		if text != next || unsafe.StringData(text) == unsafe.StringData(next) {
			t.Fatal("numeric plan kept caller clock text backing", i)
		}
	}
	for _, field := range []string{"id", "epoch", "policy"} {
		t.Run(field, func(t *testing.T) {
			bad := clock
			oversized := strings.Repeat("x", providerHistorySourceFixedTextLimit+1)
			switch field {
			case "id":
				bad.id = oversized
			case "epoch":
				bad.epoch = oversized
			case "policy":
				bad.policy = oversized
			}
			if plan, ok := providerHistoryCompileSourcePlan(config, bad); ok || plan != nil {
				t.Fatal("oversized clock text returned a partial plan")
			}
		})
	}
}

func TestProviderHistorySourceInputPreflightRefusesUnboundedCopyInputs(t *testing.T) {
	for _, field := range []string{"policy-raw", "manifest-raw", "seal-raw", "identity", "program", "policy-digest", "journal-clock", "code-name", "sealed-code", "directories", "manifest-digest"} {
		t.Run(field, func(t *testing.T) {
			config := historySelectedSourceConfig(t)
			oversized := strings.Repeat("x", providerHistorySourceFixedTextLimit+1)
			manifest := config.bounds.history.collection
			switch field {
			case "policy-raw":
				config.policy.raw = strings.Repeat("x", providerHistorySourceRawTextLimit+1)
			case "manifest-raw":
				manifest.raw = strings.Repeat("x", providerHistorySourceRawTextLimit+1)
			case "seal-raw":
				manifest.sealedRaw = strings.Repeat("x", providerHistorySourceRawTextLimit+1)
			case "identity":
				config.identity.SourceID = oversized
			case "program":
				config.bounds.history.program = oversized
			case "policy-digest":
				config.policy.provider = oversized
			case "journal-clock":
				manifest.input.journal.Observation.ClockID = oversized
			case "code-name":
				manifest.input.codes[0].Name = oversized
			case "sealed-code":
				manifest.sealed.codes[len(manifest.sealed.codes)-1].SHA256 = oversized
			case "directories":
				config.custodyDirectory, config.evidenceDirectory = strings.Repeat("x", 2<<20), "x"
			case "manifest-digest":
				manifest.digest = oversized
			}
			if providerHistorySourceInputTextFits(config) {
				t.Fatal("unbounded input reached the allocating copy")
			}
			if frozen, ok := providerHistoryFreezeSourceConfig(config); ok || frozen != (providerCustodyConfig{}) {
				t.Fatal("refused copy exposed an earlier detached value")
			}
		})
	}
}
