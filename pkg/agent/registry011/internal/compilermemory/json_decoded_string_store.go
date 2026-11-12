package compilermemory

import (
	"sync"
	"unicode/utf8"
)

// Cross-region copies take this gate before either region lock. Other region
// operations take only one region lock, so opposite-direction copies cannot
// each hold the other's source read lock while waiting for a destination.
// This control object remains a separate Go resource obligation.
var decodedStringTransfers sync.Mutex

// StoreDecodedString appends the decoded UTF-8 bytes directly into another
// original, writable region. No native slice or string escapes either lock.
// Source and destination must be distinct; a parsed source is already sealed.
// The original contexts of both regions remain in force throughout the copy.
// An entered copy spends its destination range even if it later fails. Failure
// clears that range and returns no span. This backs only the actual copied
// bytes, not the caller's Go state, compiler resources or execution authority.
func (v JSONValue) StoreDecodedString(destination *Region) (span Span, err error) {
	source := v.document.owner
	if source == nil || source.self != source || destination == nil || destination.self != destination || source == destination {
		return Span{}, ErrConfiguration
	}
	decodedStringTransfers.Lock()
	defer decodedStringTransfers.Unlock()
	source.mu.RLock()
	defer source.mu.RUnlock()
	destination.mu.Lock()
	defer destination.mu.Unlock()

	raw, err := v.decodedStringLocked()
	if err != nil {
		return Span{}, err
	}
	if err := destination.accessError(); err != nil {
		return Span{}, err
	}
	if destination.sealed {
		return Span{}, ErrConfiguration
	}
	length, err := source.decodedStringSize(raw)
	if err != nil {
		return Span{}, err
	}
	if destination.used > destination.capacity || length > destination.capacity-destination.used {
		return Span{}, ErrRange
	}
	if err := source.actionError(); err != nil {
		return Span{}, err
	}
	if err := destination.actionError(); err != nil {
		return Span{}, err
	}
	start, end := destination.used, destination.used+length
	destination.used = end
	defer func() {
		if err != nil {
			clear(destination.data[start:end])
			span = Span{}
		}
	}()

	written := start
	for at, limit := 1, len(raw)-1; at < limit; {
		if err := source.actionError(); err != nil {
			return Span{}, err
		}
		if err := destination.actionError(); err != nil {
			return Span{}, err
		}
		value, next, ok := jsonStringRune(raw, at, limit)
		if !ok {
			return Span{}, ErrJSON
		}
		width := utf8.RuneLen(value)
		if width <= 0 || width > end-written {
			return Span{}, ErrJSON
		}
		written += utf8.EncodeRune(destination.data[written:end], value)
		at = next
	}
	if written != end {
		return Span{}, ErrJSON
	}
	if err := source.actionError(); err != nil {
		return Span{}, err
	}
	if err := destination.actionError(); err != nil {
		return Span{}, err
	}
	return Span{owner: destination, offset: start, length: length}, nil
}
