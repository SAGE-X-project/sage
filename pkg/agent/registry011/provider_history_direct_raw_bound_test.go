package registry011

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderHistoryDirectRawUpperMatchesIndependentWholeRows(t *testing.T) {
	for _, tc := range []struct {
		input          providerHistoryDirectRawInput
		counts         [8]uint64
		logical        [20]uint64
		decoded, total uint64
	}{
		{providerHistoryDirectRawInput{1, 1, 4096, 65536, 8192, 4096}, [8]uint64{12, 0, 0, 5, 3, 0, 4, 26}, [20]uint64{494772, 0, 903850, 574413, 344604, 49156, 172044, 512041, 0, 147596, 221190, 738630, 0, 399, 0, 17180131336, 348337, 4164, 3458, 49156}, 262144, 17184695146},
		{providerHistoryDirectRawInput{1, 32, 4096, 65536, 8192, 4096}, [8]uint64{12, 0, 0, 2144, 2080, 31, 66, 150}, [20]uint64{494772, 0, 903850, 574413, 344604, 49156, 3727620, 512041, 0, 659778, 7078080, 316724544, 0, 276640, 0, 283472167044, 348337, 6241, 19950, 1572992}, 4325376, 283805460062},
		{providerHistoryDirectRawInput{2, 3, 4096, 65536, 8192, 4096}, [8]uint64{26, 14, 28, 27, 21, 2, 25, 58}, [20]uint64{1072216, 5346488, 1807700, 1148826, 689208, 98312, 458784, 1024082, 24578, 311714, 663570, 3988602, 8265152, 2793, 3724, 107375820850, 680220, 12492, 7714, 147468}, 1638400, 107401574493},
		{providerHistoryDirectRawInput{2, 33, 4096, 65536, 8192, 4096}, [8]uint64{26, 14, 448, 2149, 2083, 31, 84, 177}, [20]uint64{1072216, 5346488, 1807700, 1148826, 689208, 98312, 3899664, 1024082, 24578, 807374, 7299270, 317463174, 132242432, 277039, 59584, 360782758056, 680220, 14502, 23541, 1622148}, 5505024, 361258358414},
		{providerHistoryDirectRawInput{3, 96, 1048576, 33554432, 16777216, 8388608}, [8]uint64{40, 42, 1344, 6432, 6240, 93, 240, 452}, [20]uint64{1649870, 17587752, 2711550, 1723239, 402900564, 147468, 11182860, 202506363, 49156, 3422565318, 4850713152, 216666898368, 90547000320, 829920, 178752, 1030807880160, 3211655, 26917, 60116, 4718976}, 15728640, 1346944542476},
		{providerHistoryDirectRawInput{10000, 320000, 1048576, 33554432, 16777216, 8388608}, [8]uint64{139998, 699930000, 22397760000, 21440000, 20800000, 310000, 700590000, 1509999}, [20]uint64{16271207538, 293099887080000, 9038500000, 5744130000, 1343001880000, 491560000, 37276200000, 675021210000, 245755422, 11408551060000, 16169043840000, 722222994560000, 1508965760332800000, 2766400000, 2978902080000, 3009057053172060000, 3689575414, 103375903, 200829867, 15729920000}, 45913866240000, 4519070802464024144},
	} {
		got, ok := providerHistoryDirectRawUpper(tc.input)
		want := providerHistoryDirectRawCost{
			healthy: tc.counts[0], priorStages: tc.counts[1], archivedMembers: tc.counts[2], currentMembers: tc.counts[3],
			currentResults: tc.counts[4], nonfirst: tc.counts[5], policyCalls: tc.counts[6], wrapperHashes: tc.counts[7],
			total: candidateWorkVector{2: tc.decoded, 10: tc.total},
		}
		for row, n := range tc.logical {
			want.rows[row][10] = n
		}
		want.rows[providerHistoryRawPolicy][2] = tc.decoded
		if !ok || got != want {
			t.Fatal("independent whole direct raw source row", tc.input, got, want, ok)
		}
	}
}

func TestProviderHistoryDirectRawObservesGenuineOriginalsWithoutRenewal(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 2)
	historyRetainNormalPrefix(t, o, d, 1)
	s, m := o.store, o.meter
	original := d.originals[0]
	sealed := o.parent.recovery.members[0]
	if original == nil || sealed == nil || sealed.capture == nil || sealed.capture.intent == nil || d.retained != 1 {
		t.Fatal("genuine retained ordinary member and sealed filenames required")
	}
	head, err := os.ReadFile(filepath.Join(s.config.custodyDirectory, "HEAD"))
	if err != nil || string(head) != s.frontierFiles.pinRaw {
		t.Fatal("ordinary original HEAD precursor", err)
	}
	var raw [2][]byte
	for i, name := range []string{sealed.capture.intent.responseName, sealed.capture.intent.tlsName} {
		raw[i], err = os.ReadFile(filepath.Join(s.config.evidenceDirectory, name))
		if err != nil || len(raw[i]) == 0 {
			t.Fatal("ordinary retained response/TLS precursor", err)
		}
	}
	// Oracle file reads above are UNIT setup, not observed program I/O. Enter
	// only the existing original read-only DATA phase; no send/sample occurs.
	historyMemberReplayEnter(t, o, d)
	pin, clock, held, upper, window, sent, next := o.pin, s.clock.read, s.stored, m.upper, o.windowStart, o.order.sent, o.order.next
	in := providerHistoryDirectRawInput{1, 2, s.config.bounds.maximumMetadataPerCommit, s.config.bounds.maximumObjectBytes,
		uint64(s.config.capture.WireBytes), uint64(s.config.capture.BodyBytes)}
	quote, ok := providerHistoryDirectRawUpper(in)
	if !ok {
		t.Fatal("numeric DATA component for the original source ceilings")
	}
	start := m.actual
	if !providerReplayEqual(head, providerReplayBytes(s.frontierFiles.pinRaw, m), m) {
		t.Fatal("exact original HEAD byte copy/comparison")
	}
	want := start
	want[10] += 3*uint64(len(head)) + 2
	if m.actual != want {
		t.Fatal("readback clone and comparison are separate events", m.actual, want)
	}
	var current, archived uint64
	for i, expected := range []string{original.response, original.tls} {
		start = m.actual
		got := providerReplayDigest("", raw[i], m)
		if got != expected || !providerReplayTextEqual(got, expected, m) || providerReplayString(raw[i], m) != string(raw[i]) {
			t.Fatal("direct current raw hash/expected digest/text copy")
		}
		want = start
		units := 2*uint64(len(raw[i])) + 135
		want[10] += units
		current += units
		if m.actual != want {
			t.Fatal("current extra raw events differ from independent byte lengths", m.actual, want)
		}
		// Calibrate the archived reader's primitive subgraph using the same
		// ordinary originals. This is not another archived-owner invocation.
		start = m.actual
		got = providerReplayDigest("", raw[i], m)
		if got != expected || !providerReplayTextEqual(got, expected, m) ||
			!providerReplayEqual(raw[i], providerReplayBytes(string(raw[i]), m), m) {
			t.Fatal("direct archived raw hash/copy/equality primitives")
		}
		want = start
		units = 4*uint64(len(raw[i])) + 136
		want[10] += units
		archived += units
		if m.actual != want {
			t.Fatal("archive extra raw events differ from independent byte lengths", m.actual, want)
		}
	}
	l := in.wire + 65536
	if current > quote.rows[providerHistoryRawCurrentSpool][10]/quote.currentMembers || archived > 4*l+272 {
		t.Fatal("ordinary spool primitives escaped their per-member source envelope", current, archived)
	}
	checkpoint := o.parent.recovery.confirmed
	start = m.actual
	var copied uint64
	for _, value := range []string{checkpoint.head, checkpoint.frame.intent, checkpoint.frame.commit, checkpoint.frame.frontier, checkpoint.inventory.raw} {
		clone := providerHistoryRecoveryCloneObserved(value, m)
		if clone == "" || !providerReplayTextEqual(clone, value, m) {
			t.Fatal("ordinary retained checkpoint clone/equality")
		}
		copied += 3*uint64(len(value)) + 2
	}
	want = start
	want[10] += copied
	if m.actual != want || copied > 12*8192+3*min(in.metadata, 8192)+10 {
		t.Fatal("checkpoint copies or final comparisons escaped their closed envelope", m.actual, want)
	}
	var policy struct {
		Provider struct {
			Roots string `json:"tls_roots_artifact"`
		} `json:"provider"`
	}
	policyRaw := o.parent.policy.raw
	if err := json.Unmarshal([]byte(policyRaw), &policy); err != nil || len(policy.Provider.Roots) != 66 || len(policyRaw) > 65536 {
		t.Fatal("independent ordinary policy JSON precursor", err)
	}
	start = m.actual
	got := d.rootsObservedLocked()
	n := uint64(len(policyRaw))
	want = start
	want[2] += n
	want[10] += n*n + n + 2
	if got != policy.Provider.Roots || m.actual != want || n*n+n+2 > quote.rows[providerHistoryRawPolicy][10]/quote.policyCalls ||
		o.pin != pin || s.clock.read != clock || s.stored != held || m.upper != upper || o.windowStart != window ||
		o.order.sent != sent || o.order.next != next || d.retained != 1 || d.attempt == nil || d.attempt.used ||
		m.failed || m.closed || s.activeMeter.Load() != m {
		t.Fatal("original policy copy/decode changed source, allowance, clock or authority", got, m.actual, want)
	}
}

// Count actual direct caller classes chronologically, without the quote's
// triangular, weighted, packed-square or minimum-group closed expressions.
func historyDirectRawSequence(sizes []uint64) (out [8]uint64) {
	var prior uint64
	for group, members := range sizes {
		for range 12 {
			out[0]++
			for before := range group {
				out[1]++
				out[2] += sizes[before]
				out[6]++ // Archived S's policy getter.
			}
		}
		out[6] += 3  // Coordinator and two current S readers.
		out[7] += 23 // Planning4/T6/C10/dispatch1/C+S publication2.
		for ordinal := uint64(0); ordinal < members; ordinal++ {
			out[6]++    // Current member's actual retention policy getter.
			out[7] += 3 // Model pair plus retained pack/result pair.
			if ordinal > 0 {
				out[5]++
				out[6]++ // Earlier-prefix preparation's policy getter.
				out[7]++ // Earlier pack pair.
			}
			for range 2 {
				for previous := uint64(0); previous < ordinal; previous++ {
					out[3]++
					out[4]++
				}
			}
			for range 2 {
				for previous := uint64(0); previous <= ordinal; previous++ {
					out[3]++
					// The pending new member is not yet d.retained.
					if previous < ordinal {
						out[4]++
					}
				}
			}
		}
		for range 3 {
			for range members {
				out[3]++
				out[4]++
			}
		}
		prior += members
		if group+1 < len(sizes) {
			for range 2 {
				out[0]++
				out[1] += uint64(group + 1)
				out[2] += prior
				out[6] += uint64(group + 1)
			}
			out[7]++ // Nonfinal advance model pair.
		}
	}
	return
}

func TestProviderHistoryDirectRawUpperCoversChronologicalOptionalGroups(t *testing.T) {
	limits := providerHistoryDirectRawInput{3, 96, 1 << 20, 32 << 20, 16 << 20, 8 << 20}
	maximum, ok := providerHistoryDirectRawUpper(limits)
	if !ok {
		t.Fatal("finite prospective source maxima")
	}
	counts := func(q providerHistoryDirectRawCost) [8]uint64 {
		return [8]uint64{q.healthy, q.priorStages, q.archivedMembers, q.currentMembers, q.currentResults, q.nonfirst, q.policyCalls, q.wrapperHashes}
	}
	var sizes [3]uint64
	var visit func(int, int)
	visit = func(index, groups int) {
		if index < groups {
			for _, count := range []uint64{1, 2, 31, 32} {
				sizes[index] = count
				visit(index+1, groups)
			}
			return
		}
		in := limits
		in.groups, in.requests = uint64(groups), 0
		for _, n := range sizes[:groups] {
			in.requests += n
		}
		got, ok := providerHistoryDirectRawUpper(in)
		want, ceiling, whole := historyDirectRawSequence(sizes[:groups]), counts(got), counts(maximum)
		if !ok || want[0] != ceiling[0] || want[1] != ceiling[1] {
			t.Fatal("original full healthy chronology", sizes[:groups], want, ceiling)
		}
		for i, n := range want {
			if n > ceiling[i] || n > whole[i] {
				t.Fatal("skewed/optional direct raw calls exceed original maxima", sizes[:groups], i, want, ceiling)
			}
		}
	}
	for groups := 1; groups <= len(sizes); groups++ {
		visit(0, groups)
	}
	// Prospective G3/R4 can realize only one four-member group. In
	// particular N is3, so subtracting R-G would incorrectly produce1.
	limits.requests = 4
	partial, ok := providerHistoryDirectRawUpper(limits)
	want, ceiling := historyDirectRawSequence([]uint64{4}), counts(partial)
	if !ok || want[5] != 3 || partial.nonfirst != 3 {
		t.Fatal("optional group count must not reduce nonfirst member callers", partial)
	}
	for i, n := range want {
		if n > ceiling[i] {
			t.Fatal("partial realization direct calls exceed prospective maximum", i, want, ceiling)
		}
	}
}

func TestProviderHistoryDirectRawLateDataRefusalsKeepEnteredPrefixes(t *testing.T) {
	t.Run("last-copy", func(t *testing.T) {
		raw := []byte("ordinary retained original")
		positive := providerReplayUnitMeter()
		digest := providerReplayDigest("", raw, positive)
		if len(digest) != 66 || positive.failed {
			t.Fatal("ordinary raw hash precursor")
		}
		n := uint64(len(raw))
		m := providerHistoryContentUnitMeter(candidateWorkVector{10: 2*n + 134})
		got := providerReplayDigest("", raw, m)
		if got != digest || !providerReplayTextEqual(got, digest, m) || m.actual != (candidateWorkVector{10: n + 134}) {
			t.Fatal("actual raw hash and digest comparison prefix", m.actual)
		}
		before := m.actual
		if providerReplayString(raw, m) != "" || !m.failed || m.actual != before || providerReplayString(raw, m) != "" || m.actual != before {
			t.Fatal("one-short final raw copy refunded or retried earlier work", m.actual)
		}
	})
	t.Run("policy-decode", func(t *testing.T) {
		raw := `{"provider":{}}`
		n := uint64(len(raw))
		positive := providerHistoryContentUnitMeter(candidateWorkVector{2: n, 10: n*n + n + 2})
		if providerReplayJSON(providerReplayBytes(raw, positive), positive) == nil || positive.failed ||
			positive.actual != (candidateWorkVector{2: n, 10: n*n + n + 2}) {
			t.Fatal("ordinary copied policy JSON precursor", positive.actual)
		}
		m := providerHistoryContentUnitMeter(candidateWorkVector{2: n, 10: n*n + n + 1})
		copied := providerReplayBytes(raw, m)
		if string(copied) != raw || m.actual != (candidateWorkVector{10: n + 1}) || m.failed {
			t.Fatal("actual copied policy prefix must fit", m.actual)
		}
		if providerReplayJSON(copied, m) != nil || !m.failed || m.actual != (candidateWorkVector{10: n + 1}) ||
			providerReplayJSON(copied, m) != nil || m.actual != (candidateWorkVector{10: n + 1}) {
			t.Fatal("one-short decode charged a partial coordinate, refunded or retried", m.actual)
		}
	})
}

func TestProviderHistoryDirectRawPriorOperationKeepsItsSeparateDebit(t *testing.T) {
	o := historyMultistageOwner(t, 2)
	d := historyMultistageDispatch(t, o, 1)
	historyRetainNormalPrefix(t, o, d, 1)
	ticket, consumption := d.coordinator.ticket, d.coordinator.consumption
	previous, inventory, result := d.coordinator.head, o.inventory, d.originals[0].result
	completed, err := o.retainSettlementCandidate(context.Background(), d)
	if err != nil || completed == nil || completed.ack == nil {
		t.Fatal("ordinary completed earlier S prerequisite", err)
	}
	// This is UNIT setup from the actual immutable stored reference, not
	// observed product I/O or a producer's in-memory raw field. References
	// contain kind/digest/length only; the original file supplies the bytes.
	settlementRef := completed.ack.claim.settlement
	settlementName := providerHistoryStoredName(settlementRef, nil)
	if settlementRef.kind != "settlement" || settlementName == "" || settlementRef.bytes == 0 {
		t.Fatal("ordinary selected settlement reference prerequisite")
	}
	settlementRaw, err := os.ReadFile(filepath.Join(o.store.config.evidenceDirectory, settlementName))
	if err != nil || uint64(len(settlementRaw)) != settlementRef.bytes {
		t.Fatal("actual original settlement file length", err)
	}
	sum := sha256.Sum256(append([]byte(providerHistoryGroupSettlementDomain+"\x00"), settlementRaw...))
	if "0x"+hex.EncodeToString(sum[:]) != settlementRef.digest {
		t.Fatal("independent original settlement content digest")
	}
	settlementText := string(settlementRaw)
	checkpoint := o.parent.recovery.confirmed
	if checkpoint == nil || checkpoint.claim != completed.ack.claim {
		t.Fatal("original completed S checkpoint prerequisite")
	}
	if err := o.advanceStageCandidate(context.Background(), completed.ack); err != nil {
		t.Fatal("ordinary acknowledged earlier S advance", err)
	}
	next := historyMultistageDispatch(t, o, 2)
	historyMemberReplayEnter(t, o, next)
	s, m := o.store, o.meter
	pin, clock, held, upper, window, sent, ordinal := o.pin, s.clock.read, s.stored, m.upper, o.windowStart, o.order.sent, o.order.next
	parsed, err := parseProviderHistoryGroupSettlementObserved(providerReplayBytes(settlementText, m),
		ticket, consumption, previous, inventory, "complete", "", []string{result}, m)
	if err != nil || parsed == nil || len(parsed.digest) != 66 || len(checkpoint.frame.operation) != 66 {
		t.Fatal("ordinary typed earlier S and original frame operation", err)
	}
	// Parsing and ledger predecessor/result comparisons have other rows.
	// This is the additional OUTER invocation on those actual typed values.
	start := m.actual
	if !providerReplayTextEqual(parsed.digest, checkpoint.frame.operation, m) {
		t.Fatal("original prior S operation binding")
	}
	want := start
	want[10] += 133
	quote, ok := providerHistoryDirectRawUpper(providerHistoryDirectRawInput{2, 2, 4096, 65536, 8192, 4096})
	if !ok || quote.priorStages != 14 || quote.rows[providerHistoryRawPrior] != (candidateWorkVector{10: 5346488}) || m.actual != want {
		t.Fatal("fourteen prior S bindings must add1862 beyond the checkpoint row", quote, m.actual, want)
	}
	// A fresh DATA observer cannot consume half of this late comparison or
	// retry it after its real digest-copy prefix. The live owner is unchanged.
	short := providerHistoryContentUnitMeter(candidateWorkVector{10: 266})
	if providerReplayString(providerReplayBytes(parsed.digest, short), short) != parsed.digest {
		t.Fatal("ordinary two-copy digest prefix must fit")
	}
	// Both copies cost134, leaving132 rather than the required133.
	prefix := short.actual
	if prefix != (candidateWorkVector{10: 134}) || providerReplayTextEqual(parsed.digest, checkpoint.frame.operation, short) ||
		!short.failed || short.actual != prefix || providerReplayTextEqual(parsed.digest, checkpoint.frame.operation, short) || short.actual != prefix {
		t.Fatal("late separate operation comparison refunded or revived its prefix", short.actual)
	}
	if o.pin != pin || s.clock.read != clock || s.stored != held || m.upper != upper || o.windowStart != window ||
		o.order.sent != sent || o.order.next != ordinal || next.retained != 0 || m.failed || m.closed || s.activeMeter.Load() != m {
		t.Fatal("prior S calibration changed original ownership, clock, order or allowance")
	}
}

func TestProviderHistoryDirectRawOutputCopiesKeepBothParsedBodies(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 3)
	historyRetainNormalPrefix(t, o, d, 1)
	member := d.coordinator.ticket.table.members[1]
	const begin, end = `{"jsonrpc":"2.0","id":1,"result":"`, `"}`
	if member.request.method != "web3_clientVersion" || member.capture.BodyBytes < 512 {
		t.Fatal("ordinary bounded string-result member prerequisite")
	}
	body := begin + strings.Repeat("n", member.capture.BodyBytes-len(begin)-len(end)-1) + end
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil || len(body) <= 66 || len(envelope.Result) <= 66 {
		t.Fatal("ordinary independent JSON body/result precursor", err)
	}
	conn := providerIntegrationOwnedTLS(t, member.request.body, body)
	original, err := o.retainMemberCandidateWithConnection(context.Background(), d, 1, conn)
	if err != nil || original == nil {
		t.Fatal("genuine bounded ordinary string-result capture", err)
	}
	sealed := o.parent.recovery.members[1]
	if sealed == nil || sealed.capture == nil || sealed.capture.intent == nil {
		t.Fatal("original retained filenames required")
	}
	response, err := os.ReadFile(filepath.Join(o.store.config.evidenceDirectory, sealed.capture.intent.responseName))
	if err != nil {
		t.Fatal("ordinary original response precursor", err)
	}
	tlsRaw, err := os.ReadFile(filepath.Join(o.store.config.evidenceDirectory, sealed.capture.intent.tlsName))
	if err != nil {
		t.Fatal("ordinary original TLS precursor", err)
	}
	roots := historyFailureRoots(t, o)
	historyMemberReplayEnter(t, o, d)
	s, m := o.store, o.meter
	pin, clock, held, upper, window, sent, ordinal := o.pin, s.clock.read, s.stored, m.upper, o.windowStart, o.order.sent, o.order.next
	parsed, err := parseProviderHistoryMemberOriginalsObserved(string(response), string(tlsRaw),
		d.coordinator.ticket, d.coordinator.consumption, 1, roots, o.parent.tls, m)
	if err != nil || parsed.digest != original.result || parsed.response != body || parsed.result != string(envelope.Result) {
		t.Fatal("actual parser returns HTTP body and JSON result, rather than two hashes", err)
	}
	// Calibrate only the actual six-copy publication subgraph after parsing;
	// the parser's separate charged prefix is never removed or renewed.
	start := m.actual
	copy := providerHistoryMemberResult{
		providerReplayString(providerReplayBytes(parsed.digest, m), m),
		providerReplayString(providerReplayBytes(parsed.response, m), m),
		providerReplayString(providerReplayBytes(parsed.result, m), m),
	}
	units := 2*uint64(66+len(body)+len(envelope.Result)) + 6
	want := start
	want[10] += units
	b := uint64(member.capture.BodyBytes)
	p := s.config.bounds.maximumObjectBytes
	// This quote isolates this selected member's original capture ceiling.
	// It is numeric DATA, not a bound for the containing three-member owner.
	quote, ok := providerHistoryDirectRawUpper(providerHistoryDirectRawInput{1, 1, s.config.bounds.maximumMetadataPerCommit, p,
		uint64(member.capture.WireBytes), b})
	if !ok || copy != parsed || m.actual != want || units <= 2*b+270 || units > 4*b+138 ||
		quote.rows[providerHistoryRawOutput] != (candidateWorkVector{10: 2*p + 2 + 4*b + 138}) {
		t.Fatal("two body-sized fields require two separate pairs of copies", units, quote, m.actual, want)
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{10: units - 1})
	for _, value := range []string{parsed.digest, parsed.response} {
		if providerReplayString(providerReplayBytes(value, short), short) != value {
			t.Fatal("ordinary digest/body copy prefix must fit")
		}
	}
	last := providerReplayBytes(parsed.result, short)
	prefix := short.actual
	if string(last) != parsed.result || short.failed || prefix != (candidateWorkVector{10: units - uint64(len(parsed.result)) - 1}) ||
		providerReplayString(last, short) != "" || !short.failed || short.actual != prefix ||
		providerReplayString(last, short) != "" || short.actual != prefix {
		t.Fatal("one-short last result copy refunded or revived its five-copy prefix", short.actual)
	}
	if o.pin != pin || s.clock.read != clock || s.stored != held || m.upper != upper || o.windowStart != window ||
		o.order.sent != sent || o.order.next != ordinal || d.retained != 2 || m.failed || m.closed || s.activeMeter.Load() != m {
		t.Fatal("output calibration changed original ownership, clock, order or allowance")
	}
}

func TestProviderHistoryDirectRawUpperRejectsInvalidOriginalDomains(t *testing.T) {
	base := providerHistoryDirectRawInput{2, 3, 1 << 20, 32 << 20, 16 << 20, 8 << 20}
	for _, mode := range []string{"zero-groups", "wide-groups", "multiply-overflow", "few-requests", "many-requests", "request-overflow",
		"zero-metadata", "wide-metadata", "zero-object", "wide-object", "zero-wire", "wide-wire", "zero-body", "wide-body"} {
		t.Run(mode, func(t *testing.T) {
			in := base
			switch mode {
			case "zero-groups":
				in.groups = 0
			case "wide-groups":
				in.groups = (1<<20)/3 + 1
			case "multiply-overflow":
				in.groups = ^uint64(0)
			case "few-requests":
				in.requests = 1
			case "many-requests":
				in.requests = 65
			case "request-overflow":
				in.requests = ^uint64(0)
			case "zero-metadata":
				in.metadata = 0
			case "wide-metadata":
				in.metadata = 1<<20 + 1
			case "zero-object":
				in.object = 0
			case "wide-object":
				in.object = 32<<20 + 1
			case "zero-wire":
				in.wire = 0
			case "wide-wire":
				in.wire = 16<<20 + 1
			case "zero-body":
				in.body = 0
			case "wide-body":
				in.body = 8<<20 + 1
			}
			if got, ok := providerHistoryDirectRawUpper(in); ok || got != (providerHistoryDirectRawCost{}) {
				t.Fatal("invalid original raw source limit returned a partial quote", mode, got, ok)
			}
		})
	}
}

func TestProviderHistoryDirectRawCompositionOverflowReturnsWholeZero(t *testing.T) {
	// Every individual row is finite at these valid original inputs, but
	// their logical sum21864026210626460144 exceeds uint64. Both the finite
	// decoded total and all count/row fields must be discarded together.
	if got, ok := providerHistoryDirectRawUpper(providerHistoryDirectRawInput{22000, 704000, 1 << 20, 32 << 20, 16 << 20, 8 << 20}); ok || got != (providerHistoryDirectRawCost{}) {
		t.Fatal("late composed overflow leaked finite prefix fields", got, ok)
	}
}

func TestProviderHistoryDirectRawRepeatedRowOverflowReturnsWholeZero(t *testing.T) {
	// The source-valid maximum reaches a later repeated archived-spool row
	// that overflows, after finite earlier reader rows. This is pure DATA,
	// not a program-sized fixture or a native resource/feasibility claim.
	if got, ok := providerHistoryDirectRawUpper(providerHistoryDirectRawInput{349525, 11184800, 1 << 20, 32 << 20, 16 << 20, 8 << 20}); ok || got != (providerHistoryDirectRawCost{}) {
		t.Fatal("late repeated raw row overflow leaked finite prefix fields", got, ok)
	}
}
