//go:build darwin

package registry011

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

// This is the original session's owned, typed input selection. Its JSON nodes
// and bytes stay native; no historyPreparationInput or full Go config is built.
// It is still DATA. Compiler consumption, derived manifest, promotion and full
// resource qualification must be connected before it can support execution.
type nativeHistoryOwnedSelection struct {
	self      *nativeHistoryOwnedSelection
	session   *HistoryPreparationSession
	preparer  *nativeHistoryPreparation
	compiler  *providerHistorySourceCompiler
	document  compilermemory.JSONDocument
	input     compilermemory.Span
	inputHash string
	policy    *nativeProviderPolicyReader
	numbers   nativeHistoryOwner
	validated bool
	closed    bool
}

func (*nativeHistoryOwnedSelection) UnmarshalJSON([]byte) error { return ErrConfiguration }

// The actual compiler entry must call this inside its original one-use session
// operation. Retaining the selection before acquisition preserves partial-owner
// cleanup even when an allocation returns both a region and an error.
func (n *nativeHistoryPreparation) decodeOwnedSelection(s *HistoryPreparationSession, input []byte) (*nativeHistoryOwnedSelection, error) {
	if n == nil || s == nil || s.self != s || s.closed || s.compiler == nil || n.owner != s || n.compiler != s.compiler || n.selection != nil {
		return nil, ErrConfiguration
	}
	selection := &nativeHistoryOwnedSelection{session: s, preparer: n, compiler: s.compiler}
	selection.self = selection
	n.selection = selection
	if err := s.compiler.contextError(); err != nil {
		return selection, err
	}
	if len(input) == 0 || len(input) > s.maximumInput {
		return selection, ErrConfiguration
	}
	limits := compilermemory.JSONValueDepthLimits{MaximumValueDepth: 64, MaximumNodes: n.schemaNodes}
	capacity, err := compilermemory.JSONValueCapacity(len(input), limits)
	if err != nil {
		return selection, nativeHistoryDataError(err)
	}
	if err := n.acquireOwner(s, &n.root, capacity); err != nil {
		return selection, err
	}
	selection.input, err = n.root.region.Store(input)
	if err != nil {
		return selection, nativeHistoryDataError(err)
	}
	selection.document, err = n.root.region.ParseJSONAtValueDepth(selection.input, limits)
	if err != nil {
		return selection, nativeHistoryDataError(err)
	}
	root := selection.document.Root()
	maximum, found, err := root.Member("maximum_policy_bytes")
	if err != nil || !found {
		return selection, nativeHistoryDataError(err)
	}
	policyMaximum, err := maximum.JSONInt64()
	if err != nil || policyMaximum <= 0 || policyMaximum > 64<<10 {
		return selection, nativeHistoryDataError(err)
	}
	policy, found, err := root.Member("policy")
	if err != nil || !found {
		return selection, nativeHistoryDataError(err)
	}
	policyBytes, err := policy.DecodedStringLength()
	if err != nil || policyBytes <= 0 || int64(policyBytes) > policyMaximum {
		return selection, nativeHistoryDataError(err)
	}
	if err := validateNativeHistorySelectionValue(root, reflect.TypeFor[historyPreparationInput](), 0); err != nil {
		// Keep the original decoder's input-error class without duplicating an
		// existing configuration error or erasing native/lifetime causes.
		if !errors.Is(err, ErrConfiguration) {
			err = nativeHistoryDataError(err)
		}
		return selection, err
	}
	policyLimits := compilermemory.JSONValueDepthLimits{MaximumValueDepth: 64, MaximumNodes: policyBytes}
	policyCapacity, err := compilermemory.JSONValueCapacity(policyBytes, policyLimits)
	if err != nil {
		return selection, nativeHistoryDataError(err)
	}
	if err := n.acquireOwner(s, &n.policy, policyCapacity); err != nil {
		return selection, err
	}
	policyInput, err := policy.StoreDecodedString(n.policy.region)
	if err != nil {
		return selection, nativeHistoryDataError(err)
	}
	policyDocument, err := n.policy.region.ParseJSONAtValueDepth(policyInput, policyLimits)
	if err != nil {
		return selection, nativeHistoryDataError(err)
	}
	// One semantic pass visits each numeric policy string once. Their decoded
	// bytes are bounded by the unchanged original policy byte length. This is
	// representation capacity, not a selected operating budget or admission.
	if err := n.acquireOwner(s, &selection.numbers, policyBytes); err != nil {
		return selection, err
	}
	selection.policy, err = newNativeProviderPolicyReader(n.policy.region, selection.numbers.region, policyDocument, policyInput, policyBytes)
	if err != nil {
		return selection, err
	}
	if err := selection.policy.validate(); err != nil {
		return selection, err
	}
	hash := sha256.New()
	var scratch [4096]byte
	for offset := 0; offset < len(input); {
		length := min(len(scratch), len(input)-offset)
		part, err := n.root.region.Slice(selection.input, offset, length)
		if err != nil {
			return selection, err
		}
		if err := n.root.region.Read(part, scratch[:length]); err != nil {
			return selection, err
		}
		_, _ = hash.Write(scratch[:length])
		offset += length
	}
	if err := s.compiler.contextError(); err != nil {
		return selection, err
	}
	var digest [sha256.Size]byte
	selection.inputHash = hex.EncodeToString(hash.Sum(digest[:0]))
	if err := s.compiler.contextError(); err != nil {
		return selection, err
	}
	selection.validated = true
	return selection, nil
}

func (v *nativeHistoryOwnedSelection) check() error {
	if v == nil || v.self != v || !v.validated || v.closed || v.session == nil || v.preparer == nil || v.compiler == nil ||
		v.session.self != v.session || v.session.closed || v.session.compiler != v.compiler ||
		v.preparer.selection != v || v.preparer.owner != v.session || v.preparer.compiler != v.compiler {
		return ErrConfiguration
	}
	return v.compiler.contextError()
}

func (v *nativeHistoryOwnedSelection) field(name string) (compilermemory.JSONValue, error) {
	if err := v.check(); err != nil {
		return compilermemory.JSONValue{}, err
	}
	value, found, err := v.document.Root().Member(name)
	if err != nil || !found {
		return compilermemory.JSONValue{}, nativeHistoryDataError(err)
	}
	return value, nil
}

// The caller first validates the fixed type metadata with the existing schema
// constructor. This walk checks wire types and widths without setting Go values
// or materializing wire strings. Original data consumers still validate their
// semantic relations; a typed proposal is not a compiled or qualified plan.
func validateNativeHistorySelectionValue(value compilermemory.JSONValue, schema reflect.Type, depth int) error {
	if schema == nil || depth < 0 || depth > 64 {
		return ErrConfiguration
	}
	switch schema.Kind() {
	case reflect.Struct:
		kind, err := value.Kind()
		if err != nil || kind != compilermemory.JSONObject {
			return nativeHistoryDataError(err)
		}
		length, err := value.Length()
		if err != nil {
			return err
		}
		required := 0
		for index := range schema.NumField() {
			field := schema.Field(index)
			name, included, err := nativeHistoryFieldName(field)
			if err != nil {
				return err
			}
			if !included {
				continue
			}
			required++
			child, found, err := value.Member(name)
			if err != nil || !found {
				return nativeHistoryDataError(err)
			}
			if err := validateNativeHistorySelectionValue(child, field.Type, depth+1); err != nil {
				return err
			}
		}
		if length != required {
			return ErrConfiguration
		}
	case reflect.Array:
		kind, err := value.Kind()
		if err != nil || kind != compilermemory.JSONArray {
			return nativeHistoryDataError(err)
		}
		length, err := value.Length()
		if err != nil || length != schema.Len() {
			return nativeHistoryDataError(err)
		}
		for index := range schema.Len() {
			child, err := value.Index(index)
			if err != nil {
				return err
			}
			if err := validateNativeHistorySelectionValue(child, schema.Elem(), depth+1); err != nil {
				return err
			}
		}
	case reflect.String:
		_, err := value.DecodedStringLength()
		return err
	case reflect.Bool:
		_, err := value.Boolean()
		return err
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		number, err := value.JSONInt64()
		if err != nil {
			return err
		}
		bits := schema.Bits()
		if bits < 64 && (number < -(int64(1)<<(bits-1)) || number > (int64(1)<<(bits-1))-1) {
			return ErrConfiguration
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		number, err := value.JSONUint64()
		if err != nil {
			return err
		}
		if bits := schema.Bits(); bits < 64 && number>>bits != 0 {
			return ErrConfiguration
		}
	default:
		return ErrConfiguration
	}
	return nil
}
