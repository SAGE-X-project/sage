//go:build darwin

package registry011

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strings"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

const nativeHistoryManifestMagic = "SAGEHPM1"
const nativeHistoryManifestHeaderBytes = 8 + nativeHistoryRegistryDigestBytes + 8 + 6*8

// The original compiler validates the same temporary numeric input. Complete
// funding of those Go allocations remains a separate prerequisite. Only its
// exact returned plan is promoted to native DATA; no authority is granted.
func (c *providerHistorySourceCompiler) compileSelection(selection *nativeHistoryOwnedSelection) (*providerHistorySourcePlan, error) {
	if c == nil || c.owner != c || c.used == nil || c.used.Load() || selection == nil || selection.compiler != c ||
		selection.session == nil || !selection.session.used.Load() || selection.session.plan != nil || selection.plan != nil {
		return nil, ErrConfiguration
	}
	if err := selection.check(); err != nil {
		return nil, err
	}
	config, clock, err := selection.transientConfig()
	if err != nil {
		return nil, err
	}
	// Preserve all existing freeze/cost/clock guards on this same one-use
	// compiler. Its temporary allocation and overlap obligations remain open.
	plan, err := c.compile(config, clock.Maximum, clock.BaseUncertaintyMS, clock.ReadWindowMS)
	if err != nil {
		return nil, err
	}
	if err := selection.retainCompiledPlan(plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (v *nativeHistoryOwnedSelection) retainCompiledPlan(plan *providerHistorySourcePlan) error {
	if err := v.check(); err != nil {
		return err
	}
	n, s := v.preparer, v.session
	if plan == nil || plan.clock == nil || plan.retained != nil || v.plan != nil || s.plan != nil ||
		!v.compiler.used.Load() || plan.compilerDeadline != v.compiler.deadline || plan.clock.origin != v.compiler.origin ||
		plan.config.bounds.history == nil || !plan.config.bounds.history.collection.validSnapshot() {
		return ErrConfiguration
	}
	manifest := plan.config.bounds.history.collection
	if err := v.storeCompiledManifest(manifest); err != nil {
		return err
	}
	if err := n.storeResult(s, plan, v.inputHash); err != nil {
		return err
	}
	if err := n.root.region.Seal(); err != nil {
		return err
	}
	if err := n.policy.region.Seal(); err != nil {
		return err
	}
	if err := v.compiler.contextError(); err != nil {
		return err
	}
	// Keep the same original clock while detaching its small immutable text
	// leaves from any temporary parser backing. This adds no native backing
	// claim for the clock's Go control/storage or these bounded copies.
	plan.clock.mu.Lock()
	plan.clock.id = strings.Clone(plan.clock.id)
	plan.clock.epoch = strings.Clone(plan.clock.epoch)
	plan.clock.policy = strings.Clone(plan.clock.policy)
	plan.clock.mu.Unlock()
	if err := v.compiler.contextError(); err != nil {
		return err
	}
	// These are the original real owners, not a flag standing in for backing.
	// The session installs this same plan before temporary cleanup observes it.
	v.rootOwner, v.policyOwner = n.root.region, n.policy.region
	v.resultOwner, v.resultSpan = n.result.region, n.resultSpan
	v.plan, plan.retained = plan, v
	// Retain no complete Go configuration/policy/manifest tree in native mode.
	// Dropping references is not a claim of immediate physical memory release.
	plan.config = providerCustodyConfig{}
	return nil
}

func (v *nativeHistoryOwnedSelection) storeCompiledManifest(manifest *providerHistoryCollectionManifest) error {
	if !manifest.validSnapshot() || len(manifest.raw) > providerHistorySourceRawTextLimit {
		return ErrConfiguration
	}
	var header [nativeHistoryManifestHeaderBytes]byte
	copy(header[:8], nativeHistoryManifestMagic)
	copy(header[8:8+nativeHistoryRegistryDigestBytes], manifest.digest)
	offset := 8 + nativeHistoryRegistryDigestBytes
	binary.LittleEndian.PutUint64(header[offset:], uint64(len(manifest.raw)))
	offset += 8
	for _, count := range [...]uint64{
		manifest.counts.ancestryWindows, manifest.counts.contentWindows, manifest.counts.contentGroups,
		manifest.counts.requests, manifest.counts.groups, manifest.counts.groupClocks,
	} {
		binary.LittleEndian.PutUint64(header[offset:], count)
		offset += 8
	}
	n := v.preparer
	if err := n.acquireOwner(v.session, &n.manifest, len(header)+len(manifest.raw)); err != nil {
		return err
	}
	if _, err := n.manifest.region.Store(header[:]); err != nil {
		return err
	}
	if _, err := n.manifest.region.Store([]byte(manifest.raw)); err != nil {
		return err
	}
	snapshot, err := n.manifest.region.SealSnapshot()
	if err != nil {
		return err
	}
	v.manifestOwner, v.manifestSpan = n.manifest.region, snapshot
	return nil
}

func nativeHistorySpanDigest(region *compilermemory.Region, span compilermemory.Span, domain string) (string, error) {
	length, err := region.SpanLength(span)
	if err != nil {
		return "", err
	}
	if length <= 0 || length > providerHistorySourceRawTextLimit {
		return "", ErrConfiguration
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte(domain))
	_, _ = hash.Write([]byte{0})
	var scratch [4096]byte
	for offset := 0; offset < length; {
		size := min(len(scratch), length-offset)
		part, err := region.Slice(span, offset, size)
		if err != nil {
			return "", err
		}
		if err := region.Read(part, scratch[:size]); err != nil {
			return "", err
		}
		_, _ = hash.Write(scratch[:size])
		offset += size
	}
	var digest [sha256.Size]byte
	return "0x" + hex.EncodeToString(hash.Sum(digest[:0])), nil
}

// Read the actual retained manifest, not a regenerated Go object or a hash
// label. The original Region context bounds every native read and hash chunk.
func (v *nativeHistoryOwnedSelection) retainedManifest() (string, providerHistoryCollectionCounts, error) {
	var empty providerHistoryCollectionCounts
	region := v.manifestOwner
	length, err := region.SpanLength(v.manifestSpan)
	if err != nil {
		return "", empty, err
	}
	if length <= nativeHistoryManifestHeaderBytes || length-nativeHistoryManifestHeaderBytes > providerHistorySourceRawTextLimit {
		return "", empty, ErrConfiguration
	}
	span, err := region.Slice(v.manifestSpan, 0, nativeHistoryManifestHeaderBytes)
	if err != nil {
		return "", empty, err
	}
	var header [nativeHistoryManifestHeaderBytes]byte
	if err := region.Read(span, header[:]); err != nil {
		return "", empty, err
	}
	if string(header[:8]) != nativeHistoryManifestMagic {
		return "", empty, ErrConfiguration
	}
	digest := string(header[8 : 8+nativeHistoryRegistryDigestBytes])
	offset := 8 + nativeHistoryRegistryDigestBytes
	if !lowerHash(digest) || binary.LittleEndian.Uint64(header[offset:]) != uint64(length-nativeHistoryManifestHeaderBytes) {
		return "", empty, ErrConfiguration
	}
	offset += 8
	var counts providerHistoryCollectionCounts
	for _, destination := range [...]*uint64{
		&counts.ancestryWindows, &counts.contentWindows, &counts.contentGroups,
		&counts.requests, &counts.groups, &counts.groupClocks,
	} {
		*destination = binary.LittleEndian.Uint64(header[offset:])
		offset += 8
	}
	raw, err := region.Slice(v.manifestSpan, nativeHistoryManifestHeaderBytes, length-nativeHistoryManifestHeaderBytes)
	if err != nil {
		return "", empty, err
	}
	actual, err := nativeHistorySpanDigest(region, raw, "sage-provider-history-collection-manifest-v1")
	if err != nil {
		return "", empty, err
	}
	if actual != digest {
		return "", empty, ErrConfiguration
	}
	return digest, counts, nil
}

func (v *nativeHistoryOwnedSelection) checkRetainedPlan(plan *providerHistorySourcePlan) error {
	_, err := v.checkedRetainedManifest(plan)
	return err
}

func (v *nativeHistoryOwnedSelection) checkedRetainedManifest(plan *providerHistorySourcePlan) (string, error) {
	if err := v.check(); err != nil {
		return "", err
	}
	n := v.preparer
	if plan == nil || v.plan != plan || plan.retained != v || v.session.plan != plan || n.resultPlan != plan ||
		v.rootOwner == nil || v.rootOwner != n.root.region || n.root.closed ||
		v.policyOwner == nil || v.policyOwner != n.policy.region || n.policy.closed ||
		v.manifestOwner == nil || v.manifestOwner != n.manifest.region || n.manifest.closed ||
		v.resultOwner == nil || v.resultOwner != n.result.region || n.result.closed || v.resultSpan != n.resultSpan {
		return "", ErrConfiguration
	}
	if kind, err := v.document.Root().Kind(); err != nil {
		return "", err
	} else if kind != compilermemory.JSONObject {
		return "", ErrConfiguration
	}
	if v.policy == nil || v.policy.source != v.policyOwner {
		return "", ErrConfiguration
	}
	if kind, err := v.policy.root.Kind(); err != nil {
		return "", err
	} else if kind != compilermemory.JSONObject {
		return "", ErrConfiguration
	}
	if length, err := v.resultOwner.SpanLength(v.resultSpan); err != nil {
		return "", err
	} else if length != nativeHistoryResultBytes {
		return "", ErrConfiguration
	}
	digest, counts, err := v.retainedManifest()
	if err != nil {
		return "", err
	}
	if counts.groups != plan.source.collection.groups {
		return "", ErrConfiguration
	}
	if err := v.compiler.contextError(); err != nil {
		return "", err
	}
	return digest, nil
}
