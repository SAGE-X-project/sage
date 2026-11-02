package registry011

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestProviderHistoryJSONShapeUpperMatchesPortableWholeRows(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	for _, row := range []struct {
		n, elements, slots, preflight, text, data, dataDecoded, arrayConstant, arrayWord, fieldsBase uint64
	}{
		{4, 0, 0, 27, 22, 21, 4, 18, 0, 30},
		{5, 65536, 2, 38, 32, 32, 6, 29, 12, 42},
		{73, 10, 10, 5478, 5404, 10164, 142, 5341, 60, 5550},
		{65536, 16384, 16384, 4295098371, 4295032834, 8589475857, 131068, 4294983682, 98304, 4295163906},
		{67108864, 65536, 65536, 4503599761588227, 4503599694479362, 9007198784978961, 134217724, 4503599627436034, 393216, 4503599828697090},
	} {
		want := providerHistoryJSONShapeCost{
			maximum: row.n, slots: row.slots,
			jsonPreflight: candidateWorkVector{2: row.n, 10: row.preflight},
			stringValue:   candidateWorkVector{2: row.n, 10: row.text},
			hexData:       candidateWorkVector{2: row.dataDecoded, 10: row.data},
			quantity:      candidateWorkVector{2: 16, 10: 566}, value: candidateWorkVector{2: 64, 10: 4164},
			literal:      candidateWorkVector{10: 2*row.n + 7},
			array:        candidateWorkVector{2: row.n, 10: row.arrayConstant + row.arrayWord*w},
			quantityRoom: candidateWorkVector{10: 257}, valueRoom: candidateWorkVector{10: 4097},
		}
		for i, constant := range [7]uint64{18287, 10296, 88, 408, 296, 5891, 2094} {
			want.fields[i] = candidateWorkVector{2: row.n, 10: row.fieldsBase + constant}
		}
		if got, ok := providerHistoryJSONShapeUpper(row.n, row.elements); !ok || got != want {
			t.Fatal("independent portable rows or zero coordinates differ", row.n, got, want)
		}
	}
}

func TestProviderHistoryJSONShapeFieldsKeepTheFinalComparisonPrefix(t *testing.T) {
	const raw = `{"address":"0x0000000000000000000000000000000000000001","storageKeys":[]}`
	const exact = uint64(5565) // n73, key membership57, required-value comparisons104.
	for _, short := range []bool{false, true} {
		limit := exact
		if short {
			limit--
		}
		m := providerHistoryContentUnitMeter(candidateWorkVector{2: 73, 10: limit})
		fields, ok := nodeJSONFieldsObserved(json.RawMessage(raw), "address storageKeys", "address storageKeys", m)
		if short {
			if ok || fields != nil || !m.failed || m.actual != (candidateWorkVector{2: 73, 10: 5558}) {
				t.Fatal("final seven-unit null comparison lost its entered prefix", ok, m.actual)
			}
			before := m.actual
			if _, ok := nodeJSONFieldsObserved(json.RawMessage(raw), "address storageKeys", "address storageKeys", m); ok || m.actual != before {
				t.Fatal("same failed observer resumed")
			}
		} else {
			if !ok || m.failed || string(fields["storageKeys"]) != "[]" || m.actual != (candidateWorkVector{2: 73, 10: exact}) {
				t.Fatal("ordinary distinct mandatory fields changed", ok, m.actual)
			}
			upper, ok := providerHistoryJSONShapeUpper(73, 0)
			if !ok || upper.fields[2][2] < m.actual[2] || upper.fields[2][10] < m.actual[10] {
				t.Fatal("closed access schema quote missed actual work")
			}
		}
	}
}

func TestProviderHistoryJSONShapeArraysPriceSpareSlotsAndTheLastCopy(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	for _, row := range []struct {
		raw   string
		count int
		units uint64
	}{{"[0,1]", 2, 12*w + 29}, {"[]", 0, 6}, {"[ ]", 0, 3*w + 11}} {
		m := providerHistoryContentUnitMeter(candidateWorkVector{2: uint64(len(row.raw)), 10: row.units})
		got, ok := nodeRawArrayObserved(json.RawMessage(row.raw), 2, m)
		if !ok || len(got) != row.count || m.failed || m.actual != (candidateWorkVector{2: uint64(len(row.raw)), 10: row.units}) {
			t.Fatal("ordinary array headers or unused initial slots were omitted", row.raw, m.actual)
		}
	}
	const raw = "[0,1]"
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 5, 10: 12*w + 28})
	if rows, ok := nodeRawArrayObserved(json.RawMessage(raw), 2, m); ok || rows != nil || !m.failed || m.actual != (candidateWorkVector{2: 5, 10: 9*w + 28}) {
		t.Fatal("last element copy leaked partial array or lost prefix", m.actual)
	}
	before := m.actual
	if _, ok := nodeRawArrayObserved(json.RawMessage(raw), 2, m); ok || m.actual != before {
		t.Fatal("failed array observer resumed")
	}
}

func TestProviderHistoryJSONShapeJSONScansAndStringsHaveExactRows(t *testing.T) {
	for _, raw := range []string{`"0x"`, `"sage"`} {
		n := uint64(len(raw))
		m := providerHistoryContentUnitMeter(candidateWorkVector{2: n, 10: n*n + 2*n + 3})
		if !nodeJSONPreflight(context.Background(), json.RawMessage(raw), &replayBudget{limits: replayTestLimits(), observer: m}) || m.failed || m.actual != (candidateWorkVector{2: n, 10: n*n + 2*n + 3}) {
			t.Fatal("the two original scans or JSON codec changed", raw, m.actual)
		}
	}
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 6, 10: 44})
	if got, ok := nodeStringObserved(json.RawMessage(`"sage"`), m); !ok || got != "sage" || m.failed || m.actual != (candidateWorkVector{2: 6, 10: 44}) {
		t.Fatal("ordinary string scan/codec changed", got, m.actual)
	}
}

func TestProviderHistoryJSONShapeHexUsesOriginalJSONAndHexBytes(t *testing.T) {
	for _, row := range []struct {
		raw, result    string
		decoded, units uint64
	}{{`"0x"`, "", 4, 21}, {`"0x00ff"`, string([]byte{0, 255}), 12, 89}} {
		m := providerHistoryContentUnitMeter(candidateWorkVector{2: row.decoded, 10: row.units})
		got, err := rpcDataObserved(json.RawMessage(row.raw), m)
		if err != nil || string(got) != row.result || m.failed || m.actual != (candidateWorkVector{2: row.decoded, 10: row.units}) {
			t.Fatal("ordinary hex DATA changed its two codec inputs", err, m.actual)
		}
		upper, ok := providerHistoryJSONShapeUpper(uint64(len(row.raw)), 0)
		if !ok || upper.hexData != m.actual {
			t.Fatal("unescaped ordinary hex row does not reach its independent bound")
		}
	}
}

func TestProviderHistoryJSONShapeQuantityPreflightIsUnconsumed(t *testing.T) {
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 16, 10: 566})
	if got, ok := rpcQuantityObserved("0xffffffffffffffff", m); !ok || got != ^uint64(0) || m.failed || m.actual != (candidateWorkVector{2: 16, 10: 566}) {
		t.Fatal("maximum ordinary uint64 row", got, m.actual)
	}
	for _, limit := range []uint64{262, 263} {
		m := providerHistoryContentUnitMeter(candidateWorkVector{2: 1, 10: limit})
		got, ok := rpcQuantityObserved("0x1", m)
		if limit == 262 {
			if ok || got != 0 || !m.failed || m.actual != (candidateWorkVector{2: 1, 10: 6}) {
				t.Fatal("original 16-digit preflight accepted insufficient room", m.actual)
			}
			before := m.actual
			if _, ok := rpcQuantityObserved("0x1", m); ok || m.actual != before {
				t.Fatal("failed quantity observer resumed")
			}
		} else if !ok || got != 1 || m.failed || m.actual != (candidateWorkVector{2: 1, 10: 11}) {
			t.Fatal("preflight was counted as actual or the original room changed", m.actual)
		}
	}
}

func TestProviderHistoryJSONShapeValuesAndLiteralsKeepTheirInputRows(t *testing.T) {
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 64, 10: 4164})
	if got, ok := nodeValueObserved("0x"+strings.Repeat("f", 64), m); !ok || got == nil || got.BitLen() != 256 || m.failed || m.actual != (candidateWorkVector{2: 64, 10: 4164}) {
		t.Fatal("bounded ordinary value codec changed", m.actual)
	}
	for _, row := range []struct {
		raw, literal string
		units        uint64
	}{{"null", "null", 14}, {"false", "false", 17}, {" false ", "false", 19}} {
		m := providerHistoryContentUnitMeter(candidateWorkVector{10: row.units})
		if !nodeLiteralObserved(json.RawMessage(row.raw), row.literal, m) || m.failed || m.actual != (candidateWorkVector{10: row.units}) {
			t.Fatal("literal trimming lost original scan bytes", row.raw, m.actual)
		}
	}
}

func TestProviderHistoryJSONShapeOriginalObserverAccumulatesAllCalls(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	want := candidateWorkVector{2: 43, 10: 12*w + 755}
	m := providerHistoryContentUnitMeter(want)
	if !nodeJSONPreflight(context.Background(), json.RawMessage(`"0x"`), &replayBudget{limits: replayTestLimits(), observer: m}) {
		t.Fatal("normal preflight")
	}
	if _, ok := nodeStringObserved(json.RawMessage(`"sage"`), m); !ok {
		t.Fatal("normal string")
	}
	if _, err := rpcDataObserved(json.RawMessage(`"0x00ff"`), m); err != nil {
		t.Fatal("normal hex DATA", err)
	}
	if _, ok := nodeRawArrayObserved(json.RawMessage(`[0,1]`), 2, m); !ok {
		t.Fatal("normal array")
	}
	if _, ok := rpcQuantityObserved("0xffffffffffffffff", m); !ok || m.failed || m.actual != want {
		t.Fatal("original observer was replaced or actuals did not accumulate", m.actual, want)
	}
	if _, ok := nodeStringObserved(json.RawMessage(`"sage"`), m); ok || !m.failed || m.actual != want {
		t.Fatal("exhausted observer accepted another ordinary input")
	}
}

func TestProviderHistoryJSONShapeInvalidDomainsReturnWholeZero(t *testing.T) {
	for _, row := range [][2]uint64{{0, 0}, {3, 0}, {64<<20 + 1, 0}, {^uint64(0), 0}, {4, 65537}, {4, ^uint64(0)}} {
		if got, ok := providerHistoryJSONShapeUpper(row[0], row[1]); ok || got != (providerHistoryJSONShapeCost{}) {
			t.Fatal("invalid size supplied a partial JSON shape quote", row, got)
		}
	}
}

func historyJSONControlSchemas() [14][2]string {
	return [14][2]string{
		{nodeJSONHeaderFields + " hash size totalDifficulty transactions uncles withdrawals", nodeJSONHeaderFields + " hash transactions uncles withdrawals"},
		{"type chainId nonce to gas gasPrice maxPriorityFeePerGas maxFeePerGas maxFeePerBlobGas value input accessList blobVersionedHashes authorizationList v r s yParity hash from blockHash blockNumber blockTimestamp transactionIndex", "type nonce gas value input r s hash from blockHash blockNumber transactionIndex"},
		{"address storageKeys", "address storageKeys"},
		{"chainId address nonce yParity r s", "chainId address nonce yParity r s"},
		{"index validatorIndex address amount", "index validatorIndex address amount"},
		{"type root status cumulativeGasUsed logsBloom logs transactionHash contractAddress gasUsed effectiveGasPrice blockHash blockNumber transactionIndex from to blobGasUsed blobGasPrice", "type status cumulativeGasUsed logsBloom logs transactionHash gasUsed blockHash blockNumber transactionIndex"},
		{"address topics data blockNumber transactionHash transactionIndex blockHash logIndex removed blockTimestamp", "address topics data blockNumber transactionHash transactionIndex blockHash logIndex removed"},
		{"txHash result", "txHash result"},
		{"gasUsed executionGasUsed stateGasUsed gasRefund", "gasUsed executionGasUsed stateGasUsed gasRefund"},
		{"gasUsed regularGasUsed stateGasUsed gasRefund", "gasUsed regularGasUsed stateGasUsed gasRefund"},
		{"gasUsed executionGasUsed regularGasUsed stateGasUsed gasRefund", ""},
		{"type from to gas gasUsed input output error revertReason value regularGasUsed executionGasUsed stateGasUsed gasRefund stateGasReservoir", ""},
		{"type from to gas gasUsed input output error revertReason calls logs value executionGasUsed regularGasUsed stateGasUsed gasRefund stateGasReservoir", "type from to gas gasUsed input"},
		{"address topics data position index", ""},
	}
}

func TestProviderHistoryJSONControlFieldsMatchOriginalSchemaRows(t *testing.T) {
	constants := [14]uint64{18287, 10296, 88, 408, 296, 5891, 2094, 64, 392, 376, 605, 3855, 4745, 325}
	for _, n := range []uint64{4, 1024, 65536, 64 << 20} {
		parts, ok := providerHistoryJSONControlUpper(n, 17)
		old, oldOK := providerHistoryJSONShapeUpper(n, 17)
		if !ok || !oldOK {
			t.Fatal("original JSON CONTROL parts")
		}
		for i, schema := range historyJSONControlSchemas() {
			allowed, names := strings.Fields(schema[0]), uint64(0)
			seen := map[string]bool{}
			for _, name := range allowed {
				if seen[name] {
					t.Fatal("source schema contains repeated names")
				}
				seen[name], names = true, names+uint64(len(name))
			}
			linear := n
			if schema[1] != "" {
				linear = 3 * n
			}
			want := candidateWorkVector{2: n, 10: n*n + linear + 2 + constants[i]}
			e := want
			e[10] += uint64(len(allowed))*n + names + uint64(len(allowed))
			if parts.fields[i] != (providerHistoryControlPair{want, e}) || i < 7 && parts.fields[i].normal != old.fields[i] {
				t.Fatal("fixed field schema or preserved normal row", i, n, parts.fields[i])
			}
		}
	}
}

func TestProviderHistoryJSONControlFieldsFundFirstUnknownName(t *testing.T) {
	// Fixed bounded UNIT JSON only. No network, endpoint or owner is involved.
	key := strings.Repeat("z", 4096)
	raw := json.RawMessage(`{"` + key + `":0}`)
	n := uint64(len(raw))
	parts, ok := providerHistoryJSONControlUpper(n, 0)
	if !ok {
		t.Fatal("unknown-name envelope")
	}
	for i, schema := range historyJSONControlSchemas() {
		m := providerHistoryContentUnitMeter(parts.fields[i].envelope)
		if fields, ok := nodeJSONFieldsObserved(raw, schema[0], schema[1], m); ok || fields != nil || m.failed {
			t.Fatal("ordinary unknown name escaped refusal or its prefix was unfunded", i)
		}
		allowed, nameBytes := strings.Fields(schema[0]), uint64(0)
		for _, name := range allowed {
			nameBytes += uint64(len(name))
		}
		want := candidateWorkVector{2: n, 10: n*n + n + 2 + uint64(len(allowed))*uint64(len(key)) + nameBytes + uint64(len(allowed))}
		if m.actual != want {
			t.Fatal("first unknown key comparisons before mandatory fields", i, m.actual, want)
		}
	}
	short := providerHistoryContentUnitMeter(parts.fields[0].normal)
	schema := historyJSONControlSchemas()[0]
	if fields, ok := nodeJSONFieldsObserved(raw, schema[0], schema[1], short); ok || fields != nil || !short.failed {
		t.Fatal("successful fixed-name row silently funded a long unknown name")
	}
	before := short.actual
	if _, ok := nodeJSONFieldsObserved(raw, schema[0], schema[1], short); ok || short.actual != before {
		t.Fatal("failed original JSON observer resumed")
	}
}

func TestProviderHistoryJSONControlQuantityKeepsCodecBeforeUintRefusal(t *testing.T) {
	text := "0x" + strings.Repeat("f", 64)
	parts, ok := providerHistoryJSONControlUpper(uint64(len(text)), 0)
	if !ok {
		t.Fatal("quantity failure parts")
	}
	m := providerHistoryContentUnitMeter(parts.quantity.envelope)
	if value, ok := rpcQuantityObserved(text, m); ok || value != 0 || m.failed || m.actual != (candidateWorkVector{2: 64, 10: 4164}) {
		t.Fatal("uint64 refusal erased its earlier original input codec", m.actual)
	}
	short := providerHistoryContentUnitMeter(parts.quantity.normal)
	if _, ok := rpcQuantityObserved(text, short); ok || !short.failed || short.actual != (candidateWorkVector{10: 67}) {
		t.Fatal("successful16 decoded bytes admitted the long failed token", short.actual)
	}
}

func TestProviderHistoryJSONControlIndexRetainsFailedStringFallback(t *testing.T) {
	for _, row := range []struct {
		raw              string
		valid            bool
		decoded, logical uint64
	}{
		{"18446744073709551615", true, 20, 885},
		{`"0xffffffffffffffff"`, true, 36, 988},
		{`"` + strings.Repeat("a", 19), false, 40, 844},
		{"true", false, 4, 27},
	} {
		p, ok := providerHistoryJSONControlUpper(uint64(len(row.raw)), 0)
		if !ok {
			t.Fatal("bounded index parts")
		}
		m := providerHistoryContentUnitMeter(p.index.envelope)
		value, valid := nodeIndexObserved(json.RawMessage(row.raw), m)
		if valid != row.valid || valid && value != ^uint64(0) || m.failed || m.actual != (candidateWorkVector{2: row.decoded, 10: row.logical}) {
			t.Fatal("original string/fallback/decimal path", row.raw, valid, m.actual)
		}
	}
}

func TestProviderHistoryJSONControlSignedQuantityKeepsLateRangeRefusal(t *testing.T) {
	for _, text := range []string{"-0x8000000000000000", "-0x" + strings.Repeat("f", 64), "-0x0"} {
		raw, err := json.Marshal(text)
		if err != nil {
			t.Fatal(err)
		}
		p, ok := providerHistoryJSONControlUpper(uint64(len(raw)), 0)
		if !ok {
			t.Fatal("signed diagnostic parts")
		}
		m := providerHistoryContentUnitMeter(p.signedQuantity.envelope)
		valid := nodeSignedGasObserved(raw, m)
		if valid != (text == "-0x8000000000000000") || m.failed {
			t.Fatal("int64 semantic range or envelope changed", text, m.actual)
		}
		historyEventQueueWithin(t, m.actual, p.signedQuantity.envelope)
		if valid && m.actual != (candidateWorkVector{2: 37, 10: 1050}) {
			t.Fatal("ordinary negative int64 exact copy/codec row", m.actual)
		}
	}
}

// The seven original callees in nodeRootContextObserved's initial prefix.
// This is an inert leaf-call composition, not a native transaction check.
func historyJSONRootPrefixCalls(fields map[string]json.RawMessage, m *providerWorkMeter) bool {
	_, a := nodeStringObserved(fields["from"], m)
	_, b := nodeStringObserved(fields["type"], m)
	gas, c := nodeStringObserved(fields["gas"], m)
	_, q := rpcQuantityObserved(gas, m)
	_, d := nodeDataObserved(fields["input"], m)
	value, e := nodeStringObserved(fields["value"], m)
	_, v := nodeValueObserved(value, m)
	return a && b && c && q && d && e && v
}

func TestProviderHistoryJSONControlRootPrefixRetainsEveryContinuedSibling(t *testing.T) {
	quote := func(s string) json.RawMessage {
		raw, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	for _, mode := range []string{"normal", "from", "quantity", "input", "value"} {
		fields := map[string]json.RawMessage{"from": quote(replayTestWallet), "type": quote("CALL"),
			"gas": quote("0xffffffffffffffff"), "input": quote("0x" + strings.Repeat("ab", 256)), "value": quote("0x1")}
		switch mode {
		case "from":
			fields["from"] = json.RawMessage("true")
		case "quantity":
			fields["gas"] = quote("0x" + strings.Repeat("f", 64))
		case "input":
			fields["input"] = quote("0xzz")
		case "value":
			fields["value"] = quote("0x" + strings.Repeat("f", 65))
		}
		raw, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		p, ok := providerHistoryJSONControlUpper(uint64(len(raw)), 0)
		if !ok {
			t.Fatal("one original object's distinct raw-field pool")
		}
		composed := providerHistoryContentUnitMeter(p.rootContextPrefix.envelope)
		valid := historyJSONRootPrefixCalls(fields, composed)
		if valid != (mode == "normal") || composed.failed {
			t.Fatal("original seven-call prefix precursor", mode, composed.actual)
		}
		if mode == "normal" {
			historyEventQueueWithin(t, composed.actual, p.rootContextPrefix.normal)
			continue
		}
		m := providerHistoryContentUnitMeter(p.rootContextPrefix.envelope)
		// Each chosen false prefix returns before any native tx access. No
		// credential/owner/network subject is supplied by the nil UNIT value.
		if nodeRootContextObserved(fields, nil, 1, m) || m.failed || m.actual != composed.actual || m.clocks != 0 {
			t.Fatal("combined bool guard skipped an earlier sibling or entered native work", mode, m.actual, composed.actual)
		}
	}
}

func TestProviderHistoryJSONControlCommonAtomsCoverBoundedRefusals(t *testing.T) {
	for _, maximum := range []uint64{0, 1} {
		raw := json.RawMessage("[0,1]")
		p, ok := providerHistoryJSONControlUpper(uint64(len(raw)), maximum)
		if !ok {
			t.Fatal("array failure parts")
		}
		m := providerHistoryContentUnitMeter(p.array.envelope)
		if rows, ok := nodeRawArrayObserved(raw, int(maximum), m); ok || rows != nil || m.failed {
			t.Fatal("original array count refusal was not bounded")
		}
		historyEventQueueWithin(t, m.actual, p.array.normal)
	}
	for _, raw := range []json.RawMessage{json.RawMessage("[0,"), json.RawMessage("true")} {
		p, ok := providerHistoryJSONControlUpper(max(4, uint64(len(raw))), 0)
		if !ok {
			t.Fatal("bounded bad JSON/string")
		}
		m := providerHistoryContentUnitMeter(p.jsonPreflight.envelope)
		valid := nodeJSONPreflight(context.Background(), raw, &replayBudget{limits: replayTestLimits(), observer: m})
		if valid != (string(raw) == "true") || m.failed {
			t.Fatal("original full JSON preflight", string(raw), m.actual)
		}
		historyEventQueueWithin(t, m.actual, p.jsonPreflight.normal)
		m = providerHistoryContentUnitMeter(p.stringValue.envelope)
		if _, ok := nodeStringObserved(raw, m); ok || m.failed {
			t.Fatal("non-string JSON refusal")
		}
		historyEventQueueWithin(t, m.actual, p.stringValue.normal)
	}
}

func TestProviderHistoryJSONControlInvalidDomainsReturnWholeZero(t *testing.T) {
	for _, row := range [][2]uint64{{0, 0}, {3, 0}, {64<<20 + 1, 0}, {^uint64(0), 0}, {4, 65537}} {
		if got, ok := providerHistoryJSONControlUpper(row[0], row[1]); ok || got != (providerHistoryJSONControlCost{}) {
			t.Fatal("invalid raw/count bounds exposed partial failure parts", row, got)
		}
	}
}
