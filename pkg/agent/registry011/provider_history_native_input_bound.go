package registry011

import (
	"math/big"
	"reflect"
	"strconv"
)

// These private sizes are DATA inside an already serialized original reader,
// not an input credential, reservation or permission to encode another object.
// A collection caller must check its actual source bytes before typed decode.
// No production cost compiler or operating selection is provided here.
type providerHistoryNativeInputBounds struct {
	body, transaction, signing, value uint64
	collection                        bool
}

func providerHistoryNativeInputBoundObserved(m *providerWorkMeter) (providerHistoryNativeInputBounds, bool) {
	const maximum = uint64(64 << 20)
	structure := uint64(reflect.TypeFor[big.Int]().Size())
	wide := providerHistoryNativeInputBounds{transaction: maximum, signing: maximum + 256, value: maximum + structure}
	if m == nil {
		return wide, true
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.owner != m || m.failed || m.closed || !m.hasOwner() {
		m.failed = true
		return providerHistoryNativeInputBounds{}, false
	}
	o := m.historyGroup
	if o == nil {
		// A genuine recovery/first/other old role has its own original
		// predicates. In particular recovery must not match a healthy parent.
		return wide, true
	}
	if !o.fitsWork(m, candidateWorkVector{}) {
		m.failed = true
		return providerHistoryNativeInputBounds{}, false
	}
	if o.parent.bounds.collection == nil {
		return wide, true
	}
	capture := o.parent.collectionInput.capture
	bound, ok := providerHistoryNativeCollectionInputBounds(capture, uint64(strconv.IntSize/8), structure)
	if !ok || capture != o.store.config.capture {
		m.failed = true
		return providerHistoryNativeInputBounds{}, false
	}
	return bound, true
}

// This pure derivation has no owner or meter. Its numeric DATA is not a native
// object credential: consumers must still enter the original selector above.
func providerHistoryNativeCollectionInputBounds(capture ProviderCaptureLimits, word, structure uint64) (providerHistoryNativeInputBounds, bool) {
	if !capture.valid() {
		return providerHistoryNativeInputBounds{}, false
	}
	// Before canonical binding, pinned closed JSON tx tuples contain at most
	// 14 fixed fields (33 RLP bytes each) and three disjoint dynamic source
	// spans. Their prefixes plus outer/type bytes add at most37: s+499.
	// Mandatory access/authorization object syntax pays nested prefixes.
	// 1024 includes spare; no signature/location/canonical success is assumed.
	body := uint64(capture.BodyBytes)
	// The validated original capture is at most8MiB, so both fixed additions
	// are below64MiB. Word padding and struct addition remain checked below.
	transaction, signing := body+1024, body+1024+256
	value, ok := providerHistoryNativeValueCopyUpper(transaction, word, structure)
	if !ok {
		return providerHistoryNativeInputBounds{}, false
	}
	return providerHistoryNativeInputBounds{body, transaction, signing, value, true}, true
}

// Numeric DATA only, never a native-object credential or a live allowance.
// Division uses a proved nonzero word; its product cannot exceed padded, so
// the product cannot overflow. Both potentially growing additions are checked.
func providerHistoryNativeValueCopyUpper(maximum, word, structure uint64) (uint64, bool) {
	if word != 4 && word != 8 {
		return 0, false
	}
	padded, ok := providerUpperSum(maximum, word-1)
	if !ok {
		return 0, false
	}
	return providerUpperSum((padded/word)*word, structure)
}
