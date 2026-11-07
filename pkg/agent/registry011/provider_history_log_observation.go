package registry011

import (
	"bytes"
	"encoding/hex"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type replayObservedLogs struct {
	rows    []replayLog
	maximum int
	meter   *providerWorkMeter
}

func (out *replayObservedLogs) append(value replayLog) bool {
	if out == nil || out.meter == nil || out.maximum < 1 || out.maximum > 65536 || len(out.rows) >= out.maximum {
		return false
	}
	m := out.meter
	size := uint64(reflect.TypeFor[replayLog]().Size())
	if len(out.rows) == cap(out.rows) {
		capacity := min(out.maximum, max(16, 2*cap(out.rows)))
		if !m.event(providerCopyEvent, uint64(capacity)*size) {
			return false
		}
		next := make([]replayLog, len(out.rows), capacity)
		if len(out.rows) != 0 {
			if !m.event(providerCopyEvent, uint64(len(out.rows))*size) {
				return false
			}
			copy(next, out.rows)
		}
		out.rows = next
	}
	if !m.event(providerCopyEvent, size) {
		return false
	}
	n := len(out.rows)
	out.rows = out.rows[:n+1]
	out.rows[n] = value
	return true
}

func replayFrameLogObserved(l *ReplayFrameLog, meter *providerWorkMeter) (*types.Log, error) {
	if l == nil || meter == nil || !replayContentAddressObserved(l.Address, meter) || len(l.Topics) > 4 || len(l.Data) > 64<<20 {
		return nil, ErrReplayUnavailable
	}
	if !meter.event(providerCodecEvent, 40) {
		return nil, ErrReplayUnavailable
	}
	address, err := hex.DecodeString(l.Address[2:])
	if err != nil || !meter.event(providerCopyEvent, 20) {
		return nil, ErrReplayUnavailable
	}
	a := common.BytesToAddress(address)
	if !meter.event(providerCopyEvent, uint64(32*len(l.Topics))) {
		return nil, ErrReplayUnavailable
	}
	topics := make([]common.Hash, len(l.Topics))
	if !meter.event(providerCopyEvent, uint64(32*len(l.Topics))) {
		return nil, ErrReplayUnavailable
	}
	copy(topics, l.Topics)
	if !meter.event(providerCopyEvent, uint64(len(l.Data))) {
		return nil, ErrReplayUnavailable
	}
	data := bytes.Clone(l.Data)
	if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[types.Log]().Size())) {
		return nil, ErrReplayUnavailable
	}
	return &types.Log{Address: a, Topics: topics, Data: data}, nil
}
