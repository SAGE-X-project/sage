package registry011

import "context"

// One explicitly grown output belongs to the whole semantic walk. Recursive
// child results never allocate another full log prefix or trigger a hidden
// append growth/copy. Frame pointers retain the legacy borrowed DATA meaning.
func replayFrameObserved(ctx context.Context, frame *ReplayFrame, parent string, static, committed bool, depth int, b *replayBudget) ([]replayLog, error) {
	if ctx == nil || frame == nil || depth < 0 || b == nil || b.observer == nil || b.limits.MaxLogs < 0 || b.limits.MaxLogs > 65536 || b.traceLogs < 0 || b.traceLogs > b.limits.MaxLogs {
		return nil, ErrReplayUnavailable
	}
	out := replayObservedLogs{rows: []replayLog{}, maximum: b.limits.MaxLogs - b.traceLogs, meter: b.observer}
	if err := out.frame(ctx, frame, parent, static, committed, depth, b); err != nil {
		return nil, err
	}
	if ctx.Err() != nil || !b.observer.preflightWork(candidateWorkVector{}) {
		return nil, ErrReplayUnavailable
	}
	return out.rows, nil
}

func (out *replayObservedLogs) frame(ctx context.Context, f *ReplayFrame, parent string, static, committed bool, depth int, b *replayBudget) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m := out.meter
	if f == nil || depth > 32 || b.frames >= b.limits.MaxFrames || !b.charge(len(f.Type)+len(f.From)+len(f.To)+len(f.Value)+len(f.Input)+len(f.Error)) || !b.frameObserved() {
		return ErrReplayUnavailable
	}
	b.frames++
	if len(f.Calls) > b.limits.MaxFrames-b.frames || len(f.Logs) > b.limits.MaxLogs-b.traceLogs {
		return ErrReplayUnavailable
	}
	b.traceLogs += len(f.Logs)
	if !replayContentAddressObserved(f.From, m) || !replayContentAddressObserved(f.To, m) {
		return ErrReplay
	}
	value, ok := nodeValueObserved(f.Value, m)
	if !ok || parent != "" && !providerReplayTextEqual(f.From, parent, m) {
		return ErrReplay
	}
	contextAddress := f.To
	if !m.event(providerCompareEvent, uint64(len(f.Type))) {
		return ErrReplayUnavailable
	}
	switch f.Type {
	case "CALL", "CREATE", "CREATE2":
	case "STATICCALL":
		if value.Sign() != 0 {
			return ErrReplay
		}
		static = true
	case "DELEGATECALL", "CALLCODE":
		if parent == "" {
			return ErrReplay
		}
		contextAddress = parent
	default:
		return ErrReplayUnavailable
	}
	committed = committed && f.Error == ""
	eligible := committed && !static && f.Type == "CALL" && providerReplayTextEqual(contextAddress, SepoliaRegistryAddress, m) && value.Sign() == 0
	if !m.preflightWork(candidateWorkVector{}) {
		return ErrReplayUnavailable
	}
	previous, child := 0, 0
	children := func(until int) error {
		for child < until {
			if err := out.frame(ctx, &f.Calls[child], contextAddress, static, committed, depth+1, b); err != nil {
				return err
			}
			child++
		}
		return nil
	}
	for i := range f.Logs {
		l := &f.Logs[i]
		if !b.logsObserved(1) {
			return ErrReplayUnavailable
		}
		if l.AfterChildren < previous || l.AfterChildren > len(f.Calls) || !providerReplayTextEqual(l.Address, contextAddress, m) || len(l.Topics) > 4 {
			return ErrReplay
		}
		if !b.charge(len(l.Address) + 32*len(l.Topics) + len(l.Data)) {
			return ErrReplayUnavailable
		}
		if err := children(l.AfterChildren); err != nil {
			return err
		}
		previous = l.AfterChildren
		if committed {
			log, err := replayFrameLogObserved(l, m)
			if err != nil || !out.append(replayLog{log, f, eligible}) {
				return ErrReplayUnavailable
			}
		}
	}
	return children(len(f.Calls))
}

func replayContentAddressObserved(address string, meter *providerWorkMeter) bool {
	return meter.event(providerCompareEvent, uint64(len(address))) && contentAddress(address)
}
