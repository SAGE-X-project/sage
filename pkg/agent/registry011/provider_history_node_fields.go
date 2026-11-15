package registry011

import (
	"encoding/json"
	"strings"
)

func nodeJSONFieldsObserved(raw json.RawMessage, allowed, required string, meter *providerWorkMeter) (map[string]json.RawMessage, bool) {
	if meter == nil {
		return nodeJSONFields(raw, allowed, required)
	}
	fields, ok := nodeObjectObserved(raw, allowed, meter)
	if !ok {
		return nil, false
	}
	for _, name := range strings.Fields(required) {
		value, present := fields[name]
		if !present || nodeLiteralObserved(value, "null", meter) {
			return nil, false
		}
	}
	// A literal comparison can fail because its original observer refused;
	// distinguish that from an ordinary non-null field before returning DATA.
	if !meter.preflightWork(candidateWorkVector{}) {
		return nil, false
	}
	return fields, true
}
