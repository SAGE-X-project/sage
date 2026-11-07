package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

// The numeric UNIT profile predates H0. The small manually entered reader
// window tests private DATA operations, not a production compiler or Source S.
// It keeps the original observer, actuals, held upper and native-clock prefix.
func historyNativeInputFixture(t *testing.T, collection bool, extra int) *providerHistoryGroupOwner {
	t.Helper()
	var o *providerHistoryGroupOwner
	var upper providerHistoryGroupUpper
	if collection {
		in := historyCollectionInput(t)
		in.maximumSelected, in.blocks = in.journal.Deployment.Number, 1
		in.native, in.effects, in.versions, in.presentStops = 0, 0, 0, 0
		in.windowBlocks, in.contentBlocks = 1, 1
		in.capture.BodyBytes += extra
		manifest := historyCollectionManifest(t, in)
		s := providerHistoryProvisionedInspectionFixture(t, providerScanFixtureSelection{capture: &in.capture, pin: true, pinLargeObject: true,
			historyProgram: true, historyGroupWrite: true, historyGroupConsume: true, historyGroupDispatch: true,
			historyRecoveryNamespace: true, historyRecoveryLegacy: true, historyCollection: manifest, historyCollectionAdmission: true, historyCollectionFirstP: true, historyNativeInputCounts: true})
		first, err := newProviderFirstOperationCandidate(s)
		if err != nil {
			t.Fatal("original first operation", err)
		}
		h := in.journal.Deployment
		response := string(replayTestJSON(t, map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]string{
			"hash": h.Hash, "parentHash": h.ParentHash, "number": "0x" + strconv.FormatUint(h.Number, 16), "timestamp": "0x" + strconv.FormatUint(h.Timestamp, 16)}}))
		conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, response)
		if _, err := first.executeCandidateWithConnection(context.Background(), conn); err != nil {
			t.Fatal("original first P", err)
		}
		o, err = newProviderHistoryGroupOwnerCandidate(s)
		if err != nil {
			t.Fatal("independent original models", err)
		}
		upper = o.parent.collectionAdmission.maximum
	} else {
		o, upper = historyAdmissionWriterFixture(t)
	}
	s, m := o.store, o.meter
	s.mu.Lock()
	t.Cleanup(func() { o.failLocked(); s.mu.Unlock() })
	work, ok := upper.work()
	if !ok {
		t.Fatal("original held UNIT stage")
	}
	if collection {
		for _, row := range [][2]uint64{{3, 4}, {4, 16}, {6, 64}} {
			if work[row[0]] != row[1] || m.upper[row[0]] != 4*row[1] {
				t.Fatal("pre-H0 decoder counts missing from original stage or whole future", row, work, m.upper)
			}
		}
	}
	o.window, o.windowStart, o.windowUpper = true, m.actual, work
	o.clockStart, o.clockUpper, o.phase = m.clocks, upper.clocks, "preparing-admit"
	s.activeMeter.Store(m)
	if m.historyInputGroup != o || !o.fitsWork(m, candidateWorkVector{}) {
		t.Fatal("original live UNIT reader role")
	}
	return o
}

func historyNativeInputDecode(t *testing.T, m *providerWorkMeter) (ReplayBlock, replayDecodedBlock) {
	t.Helper()
	raw := replayTestInput(t).Blocks[0]
	before := m.actual
	budget := replayBudget{limits: replayTestLimits(), observer: m}
	decoded, err := replayDecodeContent(context.Background(), raw, &budget)
	if err != nil || len(decoded.txs) == 0 {
		t.Fatal("fresh complete canonical UNIT input", err)
	}
	// This exact first fixture block has one transaction and one receipt log.
	// Check the actual joined decoder, not just a positive numeric allowance.
	for _, index := range []int{3, 4, 6} {
		if m.actual[index] != before[index]+1 {
			t.Fatal("original canonical precursor count was omitted or reset", index, before, m.actual)
		}
	}
	return raw, decoded
}

func TestProviderHistoryNativeInputPreservesLegacyCeilings(t *testing.T) {
	structure := uint64(reflect.TypeFor[big.Int]().Size())
	want := providerHistoryNativeInputBounds{transaction: 64 << 20, signing: (64 << 20) + 256, value: (64 << 20) + structure}
	for _, m := range []*providerWorkMeter{nil, providerReplayUnitMeter()} {
		if got, ok := providerHistoryNativeInputBoundObserved(m); !ok || got != want {
			t.Fatal("legacy DATA ceiling changed", got, ok)
		}
	}
	o := historyNativeInputFixture(t, false, 0)
	before, reads, held, upper := o.meter.actual, o.store.clock.read, o.store.stored, o.meter.upper
	if got, ok := providerHistoryNativeInputBoundObserved(o.meter); !ok || got != want || o.meter.actual != before ||
		o.store.clock.read != reads || o.store.stored != held || o.meter.upper != upper {
		t.Fatal("genuine noncollection group narrowed or renewed its original hold", got, ok)
	}
}

func TestProviderHistoryNativeInputBindsOriginalCaptureAndWordRoundup(t *testing.T) {
	for _, extra := range []int{0, 1, 3, 7} {
		t.Run(strconv.Itoa(extra), func(t *testing.T) {
			o := historyNativeInputFixture(t, true, extra)
			m := o.meter
			actual, clocks, reads, held, upper := m.actual, m.clocks, o.store.clock.read, o.store.stored, m.upper
			body := uint64(128<<10) + uint64(extra)
			word, structure := uint64(strconv.IntSize/8), uint64(reflect.TypeFor[big.Int]().Size())
			p := body + 1024
			want := providerHistoryNativeInputBounds{body, p, p + 256, ((p+word-1)/word)*word + structure, true}
			for range 3 {
				got, ok := providerHistoryNativeInputBoundObserved(m)
				if !ok || got != want || m.actual != actual || m.clocks != clocks || o.store.clock.read != reads ||
					o.store.stored != held || m.upper != upper || o.parent.bounds.future.work != upper {
					t.Fatal("original limit lookup became work, a sample or a fresh budget", got, ok)
				}
			}
		})
	}
}

func TestProviderHistoryNativeValueCopyChecksWordBoundariesAndOverflow(t *testing.T) {
	_, capture := providerFrontierBoundsFixture()
	for _, word := range []uint64{4, 8} {
		got, ok := providerHistoryNativeCollectionInputBounds(capture, word, 32)
		p := uint64(capture.BodyBytes) + 1024
		want := providerHistoryNativeInputBounds{uint64(capture.BodyBytes), p, p + 256, ((p+word-1)/word)*word + 32, true}
		if !ok || got != want {
			t.Fatal("pure capture derivation lost its checked word bound", word, got, ok)
		}
	}
	for _, mode := range []string{"body-zero", "body-large", "timeout-zero", "word-zero", "word-one", "word-large", "struct-overflow"} {
		c, word, structure := capture, uint64(8), uint64(32)
		switch mode {
		case "body-zero":
			c.BodyBytes = 0
		case "body-large":
			c.BodyBytes = (8 << 20) + 1
		case "timeout-zero":
			c.Timeout = 0
		case "word-zero":
			word = 0
		case "word-one":
			word = 1
		case "word-large":
			word = 16
		case "struct-overflow":
			structure = ^uint64(0)
		}
		if got, ok := providerHistoryNativeCollectionInputBounds(c, word, structure); ok || got != (providerHistoryNativeInputBounds{}) {
			t.Fatal("invalid capture/word/structure DATA produced a bound", mode, got, ok)
		}
	}
	for _, word := range []uint64{4, 8} {
		for _, n := range []uint64{0, 1, 3, 4, 7, 8, 9, 31, 32, 33, 65537, (8 << 20) + 1024} {
			got, ok := providerHistoryNativeValueCopyUpper(n, word, 32)
			if !ok || got < 32+n || (got-32)%word != 0 || got-(32+n) >= word {
				t.Fatal("native nat copy must cover bytes without losing a partial word", n, word, got, ok)
			}
		}
	}
	for _, row := range [][3]uint64{{1, 0, 32}, {1, 1, 32}, {1, 16, 32}, {^uint64(0), 8, 32}, {^uint64(0) - 7, 8, 32}} {
		if got, ok := providerHistoryNativeValueCopyUpper(row[0], row[1], row[2]); ok || got != 0 {
			t.Fatal("invalid word/padding/struct overflow produced an allowance", row, got, ok)
		}
	}
}

func TestProviderHistoryNativeInputClosedJSONSizePrecedesCanonicalBinding(t *testing.T) {
	seen := map[uint8]bool{}
	// Existing finite codec subjects, not signatures, authorizations or peers.
	for _, tx := range providerHistoryNodeJSONUnitTransactions()[:6] {
		for _, n := range []int{0, 1, 55, 56, 257} {
			fields, _ := providerHistoryNodeJSONUnitFields(t, tx)
			fields["input"] = replayTestJSON(t, nodeHex(bytes.Repeat([]byte{1}, n)))
			fields["value"] = replayTestJSON(t, "0x"+strings.Repeat("f", 64))
			// A later location comparison would refuse this field. Native size
			// does not rely on successful location/signature/canonical binding.
			fields["blockHash"] = replayTestJSON(t, admissionTestDigest("f"))
			raw := replayTestJSON(t, fields)
			before := bytes.Clone(raw)
			var decoded types.Transaction
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatal("closed pinned typed decode precursor", tx.Type(), n, err)
			}
			encoded, err := decoded.MarshalBinary()
			if err != nil || len(encoded) > len(raw)+499 || len(encoded) > len(raw)+1024 || !bytes.Equal(raw, before) {
				t.Fatal("before-binding source-size lemma", tx.Type(), n, len(encoded), len(raw), err)
			}
			seen[decoded.Type()] = true
		}
	}
	if len(seen) != 5 {
		t.Fatal("all five closed native types required", seen)
	}
}

func TestProviderHistoryNativeInputOriginalDecodeAndJSONKeepActuals(t *testing.T) {
	o := historyNativeInputFixture(t, true, 0)
	m := o.meter
	raw, decoded := historyNativeInputDecode(t, m)
	tx, h := decoded.txs[0], decoded.header
	fields, _ := providerHistoryNodeJSONUnitFields(t, tx)
	sender, err := types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), tx)
	if err != nil {
		t.Fatal("existing finite signature precursor", err)
	}
	fields["from"] = replayTestJSON(t, replayAddress(sender))
	fields["blockHash"] = replayTestJSON(t, h.Hash().Hex())
	fields["blockNumber"] = replayTestJSON(t, "0x"+strconv.FormatUint(h.Number.Uint64(), 16))
	fields["transactionIndex"] = replayTestJSON(t, "0x0")
	input := replayTestJSON(t, fields)
	before := bytes.Clone(input)
	wide := providerReplayUnitMeter()
	want, err := nodeJSONTransactionObserved(input, h, 0, wide)
	if err != nil {
		t.Fatal("wide original closed JSON precursor", err)
	}
	start := m.actual
	for pass := uint64(1); pass <= 2; pass++ {
		got, err := nodeJSONTransactionObserved(input, h, 0, m)
		if err != nil || got.Hash() != want.Hash() || !bytes.Equal(input, before) {
			t.Fatal("closed JSON route changed its canonical output", err)
		}
		for i, n := range wide.actual {
			if m.actual[i]-start[i] != pass*n {
				t.Fatal("new input guards hid/reset/invented actual work", i, m.actual, start, wide.actual)
			}
		}
	}
	fields["transactionIndex"] = replayTestJSON(t, "0x1")
	bad := replayTestJSON(t, fields)
	actual := m.actual
	if got, err := nodeJSONTransactionObserved(bad, h, 0, m); err == nil || got != nil || m.actual[2]-actual[2] < 97 {
		t.Fatal("later semantic refusal must retain its entered recovery work", err, m.actual, actual)
	}
	if !bytes.Equal(raw.Transactions[0], replayTestInput(t).Blocks[0].Transactions[0]) {
		t.Fatal("original binary input changed")
	}
}

func TestProviderHistoryNativeInputRejectsOversizedOriginalsBeforeNativeDecode(t *testing.T) {
	for _, mode := range []string{"json", "binary", "retained"} {
		t.Run(mode, func(t *testing.T) {
			o := historyNativeInputFixture(t, true, 0)
			m := o.meter
			raw, decoded := historyNativeInputDecode(t, m)
			bound, ok := providerHistoryNativeInputBoundObserved(m)
			if !ok {
				t.Fatal("original input binding")
			}
			oversized := bytes.Repeat([]byte{0}, int(bound.body)+1)
			if mode == "retained" {
				retained, ok := providerHistoryCollectionRetainBlockObserved(raw, replayHeader(decoded.header), m)
				if !ok {
					t.Fatal("normal immutable retention precursor")
				}
				retained.transactions[0] = string(oversized) // finite invalid DATA only
				raw, ok = retained.replayObserved(m)
				if !ok {
					t.Fatal("reconstructed bytes before native decode")
				}
			} else {
				raw.Transactions = append([][]byte(nil), raw.Transactions...)
				raw.Transactions[0] = oversized
			}
			actual, reads, upper := m.actual, o.store.clock.read, m.upper
			if mode == "json" {
				if got, err := nodeJSONTransactionObserved(oversized, decoded.header, 0, m); err == nil || got != nil {
					t.Fatal("oversized original JSON entered typed decode")
				}
			} else {
				limits := replayTestLimits()
				limits.MaxEvidenceBytes = 64 << 20 // old evidence cap is deliberately sufficient
				budget := replayBudget{limits: limits, observer: m}
				if _, err := replayDecodeContent(context.Background(), raw, &budget); err == nil {
					t.Fatal("oversized original binary/retained transaction entered native decode")
				}
			}
			if !m.failed || m.actual != actual || o.store.clock.read != reads || m.upper != upper {
				t.Fatal("before-native byte refusal sampled/decoded/renewed", m.actual, actual)
			}
			if _, ok := providerHistoryNativeInputBoundObserved(m); ok || m.actual != actual {
				t.Fatal("failed original input selection restarted")
			}
		})
	}
}

func TestProviderHistoryNativeInputOriginalPreflightBoundaries(t *testing.T) {
	for _, kind := range []string{"hash", "signing", "value"} {
		for _, shortage := range []uint64{0, 1} {
			t.Run(kind+strconv.FormatUint(shortage, 10), func(t *testing.T) {
				o := historyNativeInputFixture(t, true, 0)
				m := o.meter
				_, decoded := historyNativeInputDecode(t, m)
				tx := decoded.txs[0]
				bound, ok := providerHistoryNativeInputBoundObserved(m)
				if !ok {
					t.Fatal("fresh guarded transaction precursor")
				}
				var room uint64
				switch kind {
				case "hash":
					room = bound.transaction*bound.transaction + 1
				case "signing":
					room = bound.signing*bound.signing + 1
				case "value":
					room = bound.value + 1
				}
				// UNIT-only fault injection inside the already-held original
				// window. Global actual/upper/hold/epoch/clock never resets.
				original := o.windowUpper
				o.windowUpper[10] = m.actual[10] - o.windowStart[10] + room - shortage
				actual, upper, reads := m.actual, m.upper, o.store.clock.read
				var succeeded bool
				switch kind {
				case "hash":
					h, ok := nodeTransactionHashObserved(tx, m)
					succeeded = ok && h == tx.Hash()
				case "signing":
					h, err := nodeSigningHashObserved(tx, m)
					succeeded = err == nil && h == types.LatestSignerForChainID(big.NewInt(11155111)).Hash(tx)
				case "value":
					v, ok := nodeTransactionValueObserved(tx, m)
					succeeded = ok && v.Cmp(tx.Value()) == 0
				}
				if succeeded != (shortage == 0) || m.failed != (shortage == 1) || m.upper != upper || o.store.clock.read != reads {
					t.Fatal("exact/one-short native primitive envelope", succeeded, m.actual, actual)
				}
				if shortage == 1 {
					if m.actual != actual {
						t.Fatal("preflight shortage was misreported as a late actual shortage")
					}
					o.windowUpper = original
					if _, ok := nodeTransactionHashObserved(tx, m); ok || m.actual != actual {
						t.Fatal("restoring UNIT window revived a spent original observer")
					}
				}
			})
		}
	}
}

func TestProviderHistoryNativeInputRefusesBrokenOriginalLineage(t *testing.T) {
	for _, mode := range []string{"owner", "group-tag", "route-seal", "both-tags", "copied-group", "active-meter", "phase", "window", "window-prefix", "clock-prefix", "manifest-owner", "capture-snapshot", "closed", "failed"} {
		t.Run(mode, func(t *testing.T) {
			o := historyNativeInputFixture(t, true, 0)
			m, p, s := o.meter, o.parent, o.store
			actual, upper, reads, held := m.actual, m.upper, s.clock.read, s.stored
			var restore func()
			switch mode {
			case "owner":
				old := m.owner
				m.owner = nil
				restore = func() { m.owner = old }
			case "group-tag", "both-tags":
				old, seal := m.historyGroup, m.historyInputGroup
				m.historyGroup = nil
				if mode == "both-tags" {
					m.historyInputGroup = nil
				}
				restore = func() { m.historyGroup, m.historyInputGroup = old, seal }
			case "route-seal":
				old := m.historyInputGroup
				m.historyInputGroup = nil
				restore = func() { m.historyInputGroup = old }
			case "copied-group":
				copy := *o
				m.historyGroup, m.historyInputGroup = &copy, &copy
				restore = func() { m.historyGroup, m.historyInputGroup = o, o }
			case "active-meter":
				s.activeMeter.Store(nil)
				restore = func() { s.activeMeter.Store(m) }
			case "phase":
				old := o.phase
				o.phase = "prepared"
				restore = func() { o.phase = old }
			case "window":
				o.window = false
				restore = func() { o.window = true }
			case "window-prefix":
				old := o.windowStart
				o.windowStart[10] = m.actual[10] + 1
				restore = func() { o.windowStart = old }
			case "clock-prefix":
				old := o.clockStart
				o.clockStart = m.clocks + 1
				restore = func() { o.clockStart = old }
			case "manifest-owner":
				old := p.bounds.collection.owner
				p.bounds.collection.owner = nil
				restore = func() { p.bounds.collection.owner = old }
			case "capture-snapshot":
				old := p.collectionInput.capture
				p.collectionInput.capture.BodyBytes++
				restore = func() { p.collectionInput.capture = old }
			case "closed":
				m.closed = true
				restore = func() { m.closed = false }
			case "failed":
				m.failed = true
				restore = func() {}
			}
			defer restore()
			if got, ok := providerHistoryNativeInputBoundObserved(m); ok || got != (providerHistoryNativeInputBounds{}) ||
				!m.failed || m.actual != actual || m.upper != upper || s.clock.read != reads || s.stored != held {
				t.Fatal("invalid original selected narrow/wide fallback or performed work", mode, got, ok)
			}
			restore()
			if _, ok := providerHistoryNativeInputBoundObserved(m); ok || m.actual != actual {
				t.Fatal("restoring metadata revived the failed original")
			}
		})
	}
}

func TestProviderHistoryNativeInputRecoveryRetainsWideIndependentRole(t *testing.T) {
	o, _ := historyMemberDispatchFixture(t, 1)
	s, r, old := o.store, o.parent.recovery, o.meter
	s.mu.Lock()
	o.failLocked()
	s.mu.Unlock()
	if err := r.beginInspectionCandidate(context.Background(), o); err != nil {
		t.Fatal("genuine original terminal inspection lane", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	defer r.stopLocked(true)
	actual, producer, reads := r.meter.actual, old.actual, s.clock.read
	got, ok := providerHistoryNativeInputBoundObserved(r.meter)
	if !ok || got.collection || got.transaction != 64<<20 || got.signing != (64<<20)+256 ||
		r.meter.actual != actual || old.actual != producer || s.clock.read != reads || !o.parent.failed {
		t.Fatal("read-only recovery required a healthy parent or borrowed collection ceiling", got, ok)
	}
	// The route seal check precedes even the real valid recovery branch.
	r.meter.historyInputGroup = o
	if r.meter.hasOwner() {
		t.Fatal("another owner branch bypassed the original group route seal")
	}
	if _, ok := providerHistoryNativeInputBoundObserved(r.meter); ok || !r.meter.failed || r.meter.actual != actual || old.actual != producer {
		t.Fatal("invalid inspection tag retried a wide route")
	}
	r.meter.historyInputGroup = nil
}
