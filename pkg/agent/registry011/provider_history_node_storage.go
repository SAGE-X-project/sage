package registry011

import "strconv"

// ReplayFrame has five strings and three slices; ReplayFrameLog has one
// string, two slices and one int. Their word-aligned concrete copy sizes are
// fixed for the supported32/64-bit Go representations and checked in UNITs.
const (
	nodeFrameCopyBytes    = uint64(19 * (strconv.IntSize / 8))
	nodeFrameLogCopyBytes = uint64(9 * (strconv.IntSize / 8))
)

func nodeFrameStorageObserved(frame *ReplayFrame, calls, logs int, meter *providerWorkMeter) bool {
	if frame == nil || meter == nil || calls < 0 || calls > 65536 || logs < 0 || logs > 65536 || len(frame.Calls) != 0 || len(frame.Logs) != 0 {
		return false
	}
	if calls != 0 {
		if !meter.event(providerCopyEvent, uint64(calls)*nodeFrameCopyBytes) {
			return false
		}
		frame.Calls = make([]ReplayFrame, calls)
	}
	if logs != 0 {
		if !meter.event(providerCopyEvent, uint64(logs)*nodeFrameLogCopyBytes) {
			return false
		}
		frame.Logs = make([]ReplayFrameLog, logs)
	}
	return true // Empty fields remain nil, preserving the existing DATA shape.
}

func nodeFrameCopyObserved(frame *ReplayFrame, index int, value ReplayFrame, meter *providerWorkMeter) bool {
	if frame == nil || meter == nil || index < 0 || index >= len(frame.Calls) || !meter.event(providerCopyEvent, nodeFrameCopyBytes) {
		return false
	}
	frame.Calls[index] = value
	return true
}

func nodeFrameLogCopyObserved(frame *ReplayFrame, index int, value ReplayFrameLog, meter *providerWorkMeter) bool {
	if frame == nil || meter == nil || index < 0 || index >= len(frame.Logs) || !meter.event(providerCopyEvent, nodeFrameLogCopyBytes) {
		return false
	}
	frame.Logs[index] = value
	return true
}
