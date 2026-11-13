//go:build darwin

package registry011

import (
	"errors"
	"reflect"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

// Preserve the original builder's format/capture rejection before policy
// semantic validation. Native policy grammar is checked before this boundary,
// just as it was by the original native input decoder.
func nativeHistorySelectionHeader(root compilermemory.JSONValue) error {
	format, found, err := root.Member("format")
	if err != nil || !found {
		return nativeHistoryDataError(err)
	}
	valid, err := format.StringEqual(historyPreparationInputFormat)
	if err != nil || !valid {
		return nativeHistoryDataError(err)
	}
	value, found, err := root.Member("capture")
	if err != nil || !found {
		return nativeHistoryDataError(err)
	}
	var capture ProviderCaptureLimits
	if err := decodeNativeHistoryValue(value, reflect.ValueOf(&capture).Elem(), 0); err != nil {
		if errors.Is(err, ErrConfiguration) {
			return err
		}
		return nativeHistoryDataError(err)
	}
	if !capture.valid() {
		return ErrConfiguration
	}
	return nil
}

// A section is a temporary input to the existing semantic/numeric constructors.
// No complete historyPreparationInput is built and no section is retained by
// the native plan. These Go working copies still require resource funding.
func nativeHistorySelectionPart[T any](selection *nativeHistoryOwnedSelection, name string) (T, error) {
	var output T
	value, err := selection.field(name)
	if err != nil {
		return output, err
	}
	if err := decodeNativeHistoryValue(value, reflect.ValueOf(&output).Elem(), 0); err != nil {
		var empty T
		if !errors.Is(err, ErrConfiguration) {
			err = nativeHistoryDataError(err)
		}
		return empty, err
	}
	return output, nil
}

func (v *nativeHistoryOwnedSelection) bindingText(name string, maximum int) (string, error) {
	value, err := v.field(name)
	if err != nil {
		return "", err
	}
	length, err := value.DecodedStringLength()
	if err != nil {
		return "", err
	}
	if length == 0 || length > maximum {
		return "", ErrProviderPolicyCandidate
	}
	data := make([]byte, length)
	if err := value.CopyString(data); err != nil {
		return "", err
	}
	return string(data), nil
}

func nativeHistorySpanBytes(region *compilermemory.Region, span compilermemory.Span, maximum int) ([]byte, error) {
	length, err := region.SpanLength(span)
	if err != nil {
		return nil, err
	}
	if length <= 0 || length > maximum {
		return nil, ErrConfiguration
	}
	data := make([]byte, length)
	if err := region.Read(span, data); err != nil {
		return nil, err
	}
	return data, nil
}

// The strict shared native validator already accepted the original policy
// bytes. Hash the actual canonical native subtrees, without reparsing a Go
// object tree or reserializing any signed/digested input.
func (v *nativeHistoryOwnedSelection) transientPolicy() (*ProviderPolicyCandidate, error) {
	if err := v.check(); err != nil {
		return nil, err
	}
	if v.policy == nil || v.policy.source != v.preparer.policy.region {
		return nil, ErrConfiguration
	}
	raw, err := nativeHistorySpanBytes(v.policy.source, v.policy.input, providerHistorySourceRawTextLimit)
	if err != nil {
		return nil, err
	}
	var digests [2]string
	for index, name := range [...]string{"instance", "provider"} {
		value, found, err := v.policy.root.Member(name)
		if err != nil || !found {
			return nil, nativeHistoryDataError(err)
		}
		span, err := value.Raw()
		if err != nil {
			return nil, err
		}
		data, err := nativeHistorySpanBytes(v.policy.source, span, providerHistorySourceRawTextLimit)
		if err != nil {
			return nil, err
		}
		domain := "sage-registry-instance-v1"
		if name == "provider" {
			domain = "sage-provider-origin-v1"
		}
		digests[index] = providerCandidateHash(domain, data)
	}
	if err := v.check(); err != nil {
		return nil, err
	}
	return &ProviderPolicyCandidate{
		raw: string(raw), policy: providerCandidateHash("sage-provider-policy-v1", raw),
		instance: digests[0], provider: digests[1],
	}, nil
}

func (v *nativeHistoryOwnedSelection) transientConfig() (providerCustodyConfig, historyPreparationClock, error) {
	var empty providerCustodyConfig
	var noClock historyPreparationClock
	policy, err := v.transientPolicy()
	if err != nil {
		return empty, noClock, err
	}
	namespace, err := v.bindingText("namespace", nativeHistoryRegistryDigestBytes)
	if err != nil {
		return empty, noClock, err
	}
	sourceID, err := v.bindingText("source_id", nativeHistoryRegistryDigestBytes)
	if err != nil {
		return empty, noClock, err
	}
	generation, err := v.bindingText("generation", 20)
	if err != nil {
		return empty, noClock, err
	}
	store, err := nativeHistorySelectionPart[historyPreparationStore](v, "store")
	if err != nil {
		return empty, noClock, err
	}
	capture, err := nativeHistorySelectionPart[ProviderCaptureLimits](v, "capture")
	if err != nil {
		return empty, noClock, err
	}
	collection, err := nativeHistorySelectionPart[historyPreparationCollection](v, "collection")
	if err != nil {
		return empty, noClock, err
	}
	bounds, err := nativeHistorySelectionPart[historyPreparationBounds](v, "bounds")
	if err != nil {
		return empty, noClock, err
	}
	clock, err := nativeHistorySelectionPart[historyPreparationClock](v, "clock")
	if err != nil {
		return empty, noClock, err
	}
	config, err := buildHistoryPreparationConfigParts(policy, namespace, sourceID, generation, store, capture, collection, bounds)
	if err != nil {
		return empty, noClock, err
	}
	return config, clock, nil
}
