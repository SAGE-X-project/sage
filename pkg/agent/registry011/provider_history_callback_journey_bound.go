package registry011

import "reflect"

// One monotone semantic callback journey only. ABI/window setup, boundaries,
// history comparison, mechanical content, role/cursor work and native physical
// qualification are separate. Numeric DATA installs no owner or allowance.
type providerHistoryCallbackJourneyCost struct {
	abi, command, outer, state, queue                   candidateWorkVector
	failedABI, failedCommand                            candidateWorkVector
	normal, incoming, envelope                          candidateWorkVector
	windows, eventSlots, selectedAttempts, maximumQueue uint64
	queueMoves, oldSlots, queueBytes                    uint64
}

func providerHistoryCallbackJourneyUpper(windows, versions, logs, evidence uint64) (providerHistoryCallbackJourneyCost, bool) {
	var empty providerHistoryCallbackJourneyCost
	if windows == 0 || windows > 1<<20 || versions > 1<<20 || logs == 0 || logs > 65536 || evidence == 0 || evidence > 64<<20 {
		return empty, false
	}
	binary, binaryOK := providerHistoryABIWindowUpper(logs, evidence)
	command, commandOK := providerHistoryCommandWindowUpper(logs, evidence)
	failedABI, abiOK := providerHistoryABIRefusalUpper()
	failedCommand, commandRefusalOK := providerHistoryCommandRefusalUpper()
	state, stateOK := providerHistoryStateApplyUpper()
	if !binaryOK || !commandOK || !abiOK || !commandRefusalOK || !stateOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryCallbackJourneyCost
	out.windows = windows
	out.eventSlots = a.product(windows, logs)
	// Original prepareSubject guards global items<V before state work. A last
	// unretained attempt aborts once, so completed+failed<=V, not V+1.
	out.selectedAttempts = min(versions, out.eventSlots)
	out.maximumQueue = min(out.selectedAttempts, logs, uint64(4096))
	for c := range out.normal {
		out.abi[c] = a.product(windows, binary.window[c])
		out.command[c] = a.product(windows, command.actualWindow[c])
		out.state[c] = a.product(out.selectedAttempts, state[c])
		out.failedABI[c] = failedABI.normal[c]
		out.failedCommand[c] = failedCommand.normal[c]
	}
	claimID := uint64(reflect.TypeFor[replayClaimID]().Size())
	claimState := uint64(reflect.TypeFor[replayClaimState]().Size())
	text := uint64(reflect.TypeFor[string]().Size())
	flag := uint64(reflect.TypeFor[bool]().Size())
	item := uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size())
	array := uint64(reflect.TypeFor[[]providerHistoryExpectedItem]().Size())
	// Outer RecordChanged and optional create consumption only. The parsed
	// command raw hash uses the same charged original frame-byte pool.
	out.outer[2] = a.product(40, out.eventSlots)
	out.outer[10] = a.sum(a.product(out.eventSlots, a.sum(a.product(2, claimID), a.product(2, claimState), text, flag, 3104)), a.product(windows, command.rawPool))
	if out.selectedAttempts > 0 {
		// replayLimits fixes History.MaxItems<=4096. prepareSubject rejects
		// q>=History.MaxItems BEFORE pending/state work; every entered append
		// therefore has q<4096. Each full/partial append at q<u owns (2q+2)
		// item copies; the later retain guard cannot add another queue slot.
		out.queueMoves = a.product(out.selectedAttempts, a.sum(out.maximumQueue, 1))
		out.oldSlots = a.product(out.selectedAttempts, out.maximumQueue-1) / 2
		// Completed retain charges R+X+Y+Z to its original window E. One
		// first-aborted tail can copy/hash BEFORE that charge; fund it too.
		out.queueBytes = min(a.sum(a.product(5, a.product(windows, evidence)), 1900544), a.product(1900544, out.selectedAttempts))
		out.queue[10] = a.sum(a.product(item, out.queueMoves), out.oldSlots, out.queueBytes, a.product(out.selectedAttempts, a.sum(174, a.product(2, text), a.product(2, flag), array)))
	}
	for c := range out.normal {
		out.normal[c] = a.sum(out.abi[c], out.command[c], out.outer[c], out.state[c], out.queue[c], out.failedABI[c], out.failedCommand[c])
		if failedABI.envelope[c] < failedABI.normal[c] || failedCommand.envelope[c] < failedCommand.normal[c] {
			return empty, false
		}
		out.incoming[c] = max(command.headroom[c], failedABI.envelope[c]-failedABI.normal[c], failedCommand.envelope[c]-failedCommand.normal[c])
	}
	if out.selectedAttempts > 0 {
		// State's record producer may encode BEFORE the late65536-byte cap.
		// Closed record nodes=13+17*128+7*16; original walker bytes<=262144.
		const maximum uint64 = 6*(13+17*128+7*16+262144) + 64
		out.incoming[10] = max(out.incoming[10], a.sum(a.product(maximum, maximum), 1))
	}
	for c := range out.envelope {
		out.envelope[c] = a.sum(out.normal[c], out.incoming[c])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}
