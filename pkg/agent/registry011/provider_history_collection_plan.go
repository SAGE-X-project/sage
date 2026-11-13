package registry011

import (
	"reflect"
	"strconv"
)

// A cursor is reconstructed DATA, never a caller cursor, T acknowledgement or
// dispatch capability. Both original readers must seed their own instance from
// their independently reconstructed first P and replay every later original S.
// Each cursor now owns its semantic graph and keeps its original observer when
// copied. Content/effects/state/history transitions consume original results;
// whole-source costs and qualification remain unfinished. Caller-supplied T
// tables still refuse; the internal collection entry requires an original
// pre-H0 derivation, whose production cost factory does not exist yet.
type providerHistoryCollectionCursor struct {
	owner                    *providerHistoryCollectionCursor
	selection, sealed        providerHistoryCollectionSelection
	phase, mode              string
	low, high, next          uint64
	spine                    []FinalizedHeader // selected down to deployment
	effects                  []providerHistoryCollectionEffect
	effectNext               int
	traces, native           uint64
	presentStops             uint64
	pending                  []providerHistoryCollectionBlock
	pendingApplied           int
	traceResults             []string
	stopRecord               string
	historyNext, stopVersion uint64
	completed                uint64
	genesis                  FinalizedHeader
	client                   string
	semantic                 *providerHistorySemanticFold
}

type providerHistoryCollectionSelection struct {
	program, manifest string
	input             providerHistoryCollectionInput
	selected          FinalizedHeader
}

type providerHistoryCollectionEffect struct {
	block, transaction string
	number, index      uint64
}

func (*providerHistoryCollectionCursor) UnmarshalJSON([]byte) error { return ErrConfiguration }

func newProviderHistoryCollectionCursorObserved(program string, manifest *providerHistoryCollectionManifest, selected FinalizedHeader, meter *providerWorkMeter) (*providerHistoryCollectionCursor, error) {
	if !lowerHash(program) || manifest == nil || !manifest.validSnapshot() || !manifest.selectedObserved(selected, meter) ||
		!meter.event(providerCopyEvent, uint64(len(program)+len(manifest.digest)+len(selected.Hash)+len(selected.ParentHash))) ||
		!meter.event(providerCopyEvent, uint64(reflect.TypeFor[providerHistoryCollectionCursor]().Size()+reflect.TypeFor[FinalizedHeader]().Size())) {
		return nil, ErrConfiguration
	}
	s := providerHistoryCollectionSelection{program, manifest.digest, manifest.input, selected}
	c := &providerHistoryCollectionCursor{selection: s, sealed: s, phase: "open", mode: "content", spine: []FinalizedHeader{selected}}
	c.owner = c
	semantic, err := newProviderHistorySemanticFoldObserved(program, s.input.journal, selected, s.input.blocks, s.input.versions, meter)
	if err != nil {
		return nil, err
	}
	c.semantic = semantic
	if selected.Number > s.input.journal.Deployment.Number {
		c.mode, c.next = "ancestry", selected.Number-1
		c.high = c.next
		c.low = max(s.input.journal.Deployment.Number, c.high-min(c.high, s.input.windowBlocks-1))
	} else {
		c.low, c.high, c.next = selected.Number, selected.Number, selected.Number
	}
	return c, nil
}

func (c *providerHistoryCollectionCursor) valid() bool {
	return c != nil && c.owner == c && c.selection == c.sealed && lowerHash(c.selection.program) && lowerHash(c.selection.manifest) &&
		c.semantic != nil && c.semantic.valid() && c.semantic.selection.program == c.selection.program && c.semantic.selection.config == c.selection.input.journal &&
		c.semantic.selection.selected == c.selection.selected && c.semantic.selection.maximumBlocks == c.selection.input.blocks && c.semantic.selection.maximumItems == c.selection.input.versions &&
		(c.mode == "ancestry" || c.mode == "content") &&
		validHeader(c.selection.selected) && c.low >= c.selection.input.journal.Deployment.Number && c.low <= c.high &&
		c.high <= c.selection.selected.Number && c.high-c.low < c.selection.input.windowBlocks &&
		c.selection.input.windowBlocks > 0 && c.selection.input.contentBlocks > 0 && c.selection.input.contentBlocks <= 10 &&
		c.native <= c.selection.input.native && c.traces <= c.selection.input.effects && c.presentStops <= c.selection.input.presentStops && c.completed < 1<<20 &&
		len(c.spine) > 0 && uint64(len(c.spine)) <= c.selection.input.blocks && c.spine[0] == c.selection.selected
}

// Planning does not advance phase, counters, windows or the original ledger.
// Response-dependent facts must first come from each reader's own completed
// original S. In particular no returned list or public endpoint chooses these
// requests. All incomplete facts refuse instead of producing a shorter table.
func (c *providerHistoryCollectionCursor) nextTableObserved(stage uint64, methods providerMethodBounds, capture ProviderCaptureLimits, meter *providerWorkMeter) (*providerHistoryGroupTable, error) {
	if !c.valid() || stage != c.completed+1 || meter == nil || !c.selection.input.capture.valid() || c.selection.input.capture != capture {
		return nil, ErrConfiguration
	}
	var requests []*providerIntegrationRequest
	purpose := ""
	add := func(method string, params any) bool {
		if len(requests) >= providerHistoryGroupMembers {
			return false
		}
		request, err := newProviderIntegrationRequestObserved(method, params, meter)
		if err != nil || !meter.event(providerCopyEvent, 1) {
			return false
		}
		requests = append(requests, request)
		return true
	}
	selectedBlock := map[string]any{"blockHash": c.selection.selected.Hash, "requireCanonical": true}
	registryID := []any{map[string]string{"to": SepoliaRegistryAddress, "data": "0x04ef898d"}, selectedBlock}
	switch c.phase {
	case "open":
		purpose = "identity"
		if !add("eth_chainId", []any{}) || !add("eth_getBlockByNumber", []any{"0x0", false}) ||
			!add("eth_call", registryID) || !add("web3_clientVersion", []any{}) || !add("eth_getBlockByNumber", []any{"finalized", false}) {
			return nil, ErrConfiguration
		}
		for _, code := range c.selection.input.codes {
			if !add("eth_getCode", []any{code.Address, selectedBlock}) {
				return nil, ErrConfiguration
			}
		}
	case "ancestry":
		purpose = "ancestry"
		if c.mode != "ancestry" || c.next < c.low || c.next > c.high {
			return nil, ErrConfiguration
		}
		count := min(uint64(providerHistoryGroupMembers), c.next-c.low+1)
		for i := uint64(0); i < count; i++ {
			tag := providerHistoryNumberTagObserved(c.next-i, meter)
			if tag == "" || !add("eth_getBlockByNumber", []any{tag, false}) {
				return nil, ErrConfiguration
			}
		}
	case "content":
		purpose = "content"
		if c.mode != "content" || c.next < c.low || c.next > c.high {
			return nil, ErrConfiguration
		}
		count, native := min(c.selection.input.contentBlocks, c.high-c.next+1), uint64(0)
		for i := uint64(0); i < count; i++ {
			header, ok := c.spineHeader(c.next+i, meter)
			if !ok {
				return nil, ErrConfiguration
			}
			blockMethod, blockParams, receipts := "debug_getRawBlock", []any{header.Hash}, "debug_getRawReceipts"
			if c.selection.input.encoding == EthereumJSONBlocks {
				blockMethod, blockParams, receipts = "eth_getBlockByHash", []any{header.Hash, true}, "eth_getBlockReceipts"
			} else if c.selection.input.encoding != "debug-raw-rlp-v1" {
				return nil, ErrConfiguration
			}
			if !add(blockMethod, blockParams) || !add(receipts, []any{header.Hash}) {
				return nil, ErrConfiguration
			}
			if header.Timestamp >= sepoliaAmsterdamTime {
				native++
				options, ok := c.traceOptionsObserved(true, meter)
				if !ok || !candidateCapacity(0, c.native, native, c.selection.input.native) || !add("debug_traceBlockByHash", []any{header.Hash, options}) {
					return nil, ErrConfiguration
				}
			}
		}
	case "effects":
		purpose = "effects"
		if !c.effectFactsObserved(meter) || c.effectNext < 0 || c.effectNext >= len(c.effects) || c.traces < uint64(c.effectNext) {
			return nil, ErrConfiguration
		}
		count := min(providerHistoryGroupMembers, len(c.effects)-c.effectNext)
		if !candidateCapacity(0, c.traces, uint64(count), c.selection.input.effects) {
			return nil, ErrConfiguration
		}
		options, ok := c.traceOptionsObserved(false, meter)
		if !ok {
			return nil, ErrConfiguration
		}
		for i := range count {
			effect := c.effects[c.effectNext+i]
			if !lowerHash(effect.block) || !lowerHash(effect.transaction) || !add("debug_traceTransaction", []any{effect.transaction, options}) {
				return nil, ErrConfiguration
			}
		}
	case "state", "grants", "history":
		if c.mode != "content" {
			return nil, ErrConfiguration
		}
		stop, ok := c.spineHeader(c.high, meter)
		if !ok {
			return nil, ErrConfiguration
		}
		method, count := "record", uint64(1)
		purpose = "state"
		if c.phase == "grants" {
			method = "activeGrants"
		}
		if c.phase == "history" {
			method, purpose = "historyItem", "history"
			if c.historyNext == 0 || c.historyNext > c.stopVersion || c.stopVersion > c.selection.input.versions {
				return nil, ErrConfiguration
			}
			count = min(uint64(providerHistoryGroupMembers), c.stopVersion-c.historyNext+1)
		}
		for i := uint64(0); i < count; i++ {
			version := uint64(0)
			if method == "historyItem" {
				version = c.historyNext + i
			}
			params, err := providerHistoryRegistryParamsObserved(SepoliaRegistryAddress, c.selection.input.journal.Agent, method, version, stop.Hash, meter)
			if err != nil || !add("eth_call", params) {
				return nil, ErrConfiguration
			}
		}
	case "close":
		purpose = "identity"
		stopNumber := c.high
		if c.mode == "ancestry" {
			stopNumber = c.low
		}
		stop, selected := providerHistoryNumberTagObserved(stopNumber, meter), providerHistoryNumberTagObserved(c.selection.selected.Number, meter)
		if stop == "" || selected == "" || !add("eth_getBlockByNumber", []any{stop, false}) || !add("eth_getBlockByNumber", []any{selected, false}) ||
			!add("eth_getBlockByNumber", []any{"finalized", false}) || !add("eth_chainId", []any{}) || !add("eth_getBlockByNumber", []any{"0x0", false}) ||
			!add("eth_call", registryID) || !add("web3_clientVersion", []any{}) {
			return nil, ErrConfiguration
		}
	default:
		return nil, ErrConfiguration
	}
	return newProviderHistoryGroupTableObserved(c.selection.program, purpose, stage, requests, methods, capture, meter)
}

func (c *providerHistoryCollectionCursor) spineHeader(number uint64, meter *providerWorkMeter) (FinalizedHeader, bool) {
	if !c.valid() || meter == nil || number > c.selection.selected.Number {
		return FinalizedHeader{}, false
	}
	index := c.selection.selected.Number - number
	if index >= uint64(len(c.spine)) {
		return FinalizedHeader{}, false
	}
	header := c.spine[index]
	if header.Number != number || !validHeader(header) || !meter.event(providerCompareEvent, uint64(len(header.Hash)+len(header.ParentHash))) {
		return FinalizedHeader{}, false
	}
	return header, true
}

// Transaction order is block number then exact receipt/transaction index.
// Hash/number context comes from the known spine, not a supplied hash-only list.
// Full receipt/effect derivation must still precede selecting this phase.
func (c *providerHistoryCollectionCursor) effectFactsObserved(meter *providerWorkMeter) bool {
	if !c.valid() || meter == nil || c.mode != "content" || len(c.effects) == 0 || uint64(len(c.effects)) > c.selection.input.effects {
		return false
	}
	maximum, ok := providerUpperProduct(c.selection.input.contentBlocks, uint64(c.selection.input.journal.Replay.MaxTransactions))
	if !ok || uint64(len(c.effects)) > maximum {
		return false
	}
	for i, effect := range c.effects {
		if effect.number < c.low || effect.number > c.high || effect.index >= uint64(c.selection.input.journal.Replay.MaxTransactions) || !lowerHash(effect.transaction) {
			return false
		}
		header, ok := c.spineHeader(effect.number, meter)
		if !ok || !providerReplayTextEqual(header.Hash, effect.block, meter) {
			return false
		}
		if i > 0 {
			previous := c.effects[i-1]
			if effect.number < previous.number || effect.number == previous.number && effect.index <= previous.index {
				return false
			}
		}
	}
	return true
}

func (c *providerHistoryCollectionCursor) traceOptionsObserved(gas bool, meter *providerWorkMeter) (map[string]any, bool) {
	if !c.valid() || c.selection.input.traceTimeout <= 0 || c.selection.input.traceTimeout > c.selection.input.capture.Timeout || meter == nil || !meter.event(providerEncodeEvent, 8) {
		return nil, false
	}
	options := map[string]any{"tracer": "callTracer", "timeout": c.selection.input.traceTimeout.String(), "reexec": 0,
		"tracerConfig": map[string]bool{"onlyTopCall": false, "withLog": true}}
	if gas {
		switch c.selection.input.gas {
		case SepoliaAmsterdamGasProfile, SepoliaAmsterdamNethermindGasProfile:
			options["tracer"] = "stateGasTracer"
			delete(options, "tracerConfig")
		case SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamRootExecutionGasProfile:
			options["tracerConfig"] = map[string]bool{"onlyTopCall": true, "withLog": false, "includePrecompiles": true}
		default:
			return nil, false
		}
	}
	return options, true
}

func providerHistoryNumberTagObserved(number uint64, meter *providerWorkMeter) string {
	if meter == nil || !meter.event(providerEncodeEvent, 8) {
		return ""
	}
	raw := strconv.FormatUint(number, 16)
	if !meter.event(providerCopyEvent, uint64(2+len(raw))) {
		return ""
	}
	return "0x" + raw
}
