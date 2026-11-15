//go:build darwin

package registry011

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"reflect"
	"strings"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

// The native reader consumes the same exact DTO type as the existing diagnostic
// reader. Its detached DTO/strings remain Go allocations, not native funding.
func (n *nativeHistoryPreparation) decodeInput(s *HistoryPreparationSession, input []byte) (historyPreparationInput, string, error) {
	var empty historyPreparationInput
	limits := compilermemory.JSONValueDepthLimits{MaximumValueDepth: 64, MaximumNodes: n.schemaNodes}
	capacity, err := compilermemory.JSONValueCapacity(len(input), limits)
	if err != nil {
		return empty, "", nativeHistoryDataError(err)
	}
	if err := n.acquireOwner(s, &n.root, capacity); err != nil {
		return empty, "", err
	}
	span, err := n.root.region.Store(input)
	if err != nil {
		return empty, "", nativeHistoryDataError(err)
	}
	document, err := n.root.region.ParseJSONAtValueDepth(span, limits)
	if err != nil {
		return empty, "", nativeHistoryDataError(err)
	}
	root := document.Root()
	// Apply the existing public policy ceiling before copying the transport
	// string or sizing its independent syntax document. This selects no policy.
	maximum, found, err := root.Member("maximum_policy_bytes")
	if err != nil || !found {
		return empty, "", nativeHistoryDataError(err)
	}
	policyLimit, err := maximum.JSONInt64()
	if err != nil || policyLimit <= 0 || policyLimit > 64<<10 {
		return empty, "", nativeHistoryDataError(err)
	}
	policy, found, err := root.Member("policy")
	if err != nil || !found {
		return empty, "", nativeHistoryDataError(err)
	}
	policyBytes, err := policy.DecodedStringLength()
	if err != nil || policyBytes <= 0 || int64(policyBytes) > policyLimit {
		return empty, "", nativeHistoryDataError(err)
	}
	var output historyPreparationInput
	if err := decodeNativeHistoryValue(root, reflect.ValueOf(&output).Elem(), 0); err != nil {
		return empty, "", nativeHistoryDataError(err)
	}
	policyLimits := compilermemory.JSONValueDepthLimits{MaximumValueDepth: 64, MaximumNodes: policyBytes}
	policyCapacity, err := compilermemory.JSONValueCapacity(policyBytes, policyLimits)
	if err != nil {
		return empty, "", nativeHistoryDataError(err)
	}
	if err := n.acquireOwner(s, &n.policy, policyCapacity); err != nil {
		return empty, "", err
	}
	policySpan, err := n.policy.region.Store([]byte(output.Policy))
	if err != nil {
		return empty, "", nativeHistoryDataError(err)
	}
	if _, err := n.policy.region.ParseJSONAtValueDepth(policySpan, policyLimits); err != nil {
		return empty, "", nativeHistoryDataError(err)
	}
	// Hash the stored snapshot actually consumed above. The borrowed input is
	// never retained, and no second full-size Go input copy is created.
	hash := sha256.New()
	var scratch [4096]byte
	for offset := 0; offset < len(input); {
		length := min(len(scratch), len(input)-offset)
		part, err := n.root.region.Slice(span, offset, length)
		if err != nil {
			return empty, "", nativeHistoryDataError(err)
		}
		if err := n.root.region.Read(part, scratch[:length]); err != nil {
			return empty, "", nativeHistoryDataError(err)
		}
		_, _ = hash.Write(scratch[:length])
		offset += length
	}
	if err := s.compiler.contextError(); err != nil {
		return empty, "", err
	}
	var digest [sha256.Size]byte
	sum := hash.Sum(digest[:0])
	return output, hex.EncodeToString(sum), nil
}

func nativeHistoryDataError(err error) error {
	if err == nil {
		return ErrConfiguration
	}
	return errors.Join(ErrConfiguration, err)
}

func nativeHistoryFieldName(field reflect.StructField) (string, bool, error) {
	if field.PkgPath != "" {
		return "", false, nil
	}
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if name == "-" {
		return "", false, nil
	}
	if field.Anonymous {
		return "", false, ErrConfiguration
	}
	if name == "" {
		name = field.Name
	}
	return name, true, nil
}

func nativeHistorySchemaNodes(schema reflect.Type, depth int) (int, error) {
	if schema == nil || depth < 0 || depth > 64 {
		return 0, ErrConfiguration
	}
	switch schema.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return 1, nil
	case reflect.Array:
		element, err := nativeHistorySchemaNodes(schema.Elem(), depth+1)
		if err != nil || element <= 0 || schema.Len() > (math.MaxInt-1)/element {
			return 0, ErrConfiguration
		}
		return 1 + schema.Len()*element, nil
	case reflect.Struct:
		nodes := 1
		for index := range schema.NumField() {
			name, included, err := nativeHistoryFieldName(schema.Field(index))
			if err != nil {
				return 0, err
			}
			if !included {
				continue
			}
			for previous := 0; previous < index; previous++ {
				other, present, err := nativeHistoryFieldName(schema.Field(previous))
				if err != nil || (present && other == name) {
					return 0, ErrConfiguration
				}
			}
			child, err := nativeHistorySchemaNodes(schema.Field(index).Type, depth+1)
			if err != nil || child >= math.MaxInt-nodes {
				return 0, ErrConfiguration
			}
			nodes += 1 + child
		}
		return nodes, nil
	default:
		return 0, ErrConfiguration
	}
}

func decodeNativeHistoryValue(input compilermemory.JSONValue, output reflect.Value, depth int) error {
	if !output.IsValid() || !output.CanSet() || depth < 0 || depth > 64 {
		return ErrConfiguration
	}
	switch output.Kind() {
	case reflect.Struct:
		kind, err := input.Kind()
		if err != nil || kind != compilermemory.JSONObject {
			return nativeHistoryDataError(err)
		}
		length, err := input.Length()
		if err != nil {
			return err
		}
		required := 0
		for index := range output.NumField() {
			name, included, err := nativeHistoryFieldName(output.Type().Field(index))
			if err != nil {
				return err
			}
			if !included {
				continue
			}
			required++
			child, found, err := input.Member(name)
			if err != nil || !found {
				return nativeHistoryDataError(err)
			}
			if err := decodeNativeHistoryValue(child, output.Field(index), depth+1); err != nil {
				return err
			}
		}
		if length != required {
			return ErrConfiguration
		}
		return nil
	case reflect.Array:
		kind, err := input.Kind()
		if err != nil || kind != compilermemory.JSONArray {
			return nativeHistoryDataError(err)
		}
		length, err := input.Length()
		if err != nil || length != output.Len() {
			return nativeHistoryDataError(err)
		}
		for index := range output.Len() {
			child, err := input.Index(index)
			if err != nil {
				return err
			}
			if err := decodeNativeHistoryValue(child, output.Index(index), depth+1); err != nil {
				return err
			}
		}
		return nil
	case reflect.String:
		length, err := input.DecodedStringLength()
		if err != nil {
			return err
		}
		text := make([]byte, length)
		if err := input.CopyString(text); err != nil {
			return err
		}
		output.SetString(string(text))
		return nil
	case reflect.Bool:
		value, err := input.Boolean()
		if err != nil {
			return err
		}
		output.SetBool(value)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := input.JSONInt64()
		if err != nil || output.OverflowInt(value) {
			return nativeHistoryDataError(err)
		}
		output.SetInt(value)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := input.JSONUint64()
		if err != nil || output.OverflowUint(value) {
			return nativeHistoryDataError(err)
		}
		output.SetUint(value)
		return nil
	default:
		return ErrConfiguration
	}
}
