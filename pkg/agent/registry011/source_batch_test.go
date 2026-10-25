package registry011

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Public recorded block/code/root responses are combined with explicit inert
// state scenarios. Synthetic metadata/state is never called a live observation.
// No server, operator key, transaction submission or VM is used.
func sourceBatchFixture(t *testing.T, backfill bool, mode string) (*ReadRPC, JournalConfig, JournalCheckpoint, SourceReadLimits, SourceClock, *int) {
	t.Helper()
	reader := rootGasFixture(t, nil)
	config := journalTestConfig(t, replayTestInput(t))
	config.Agent = replayTestAgent
	config.Deployment = journalTestHeader(t, gasRecorded(t))
	config.CodeIdentity, config.ABIIdentity = SepoliaEvidenceDigests()
	config.Replay.MaxBlocks = 1
	selected := config.Deployment
	if backfill {
		selected = FinalizedHeader{testBlock, config.Deployment.Hash, config.Deployment.Number + 1, config.Deployment.Timestamp + 12}
	}
	if mode == "ancestry-budget" {
		selected.Number++
		selected.Timestamp += 12
		config.Observation.MaxAncestryHeaders = 1
	}
	var extensions []sourceOwnedBlock
	if mode == "two-blocks" || mode == "resume-cursor" || mode == "resume-absent" || mode == "block-lineage-drift" || mode == "aggregate-transactions" {
		second := sourceNextOwnedBlock(t, sourceRecordedBlock(t), mode == "aggregate-transactions")
		extensions = append(extensions, second)
		selected = second.header
		config.Replay.MaxBlocks = 2
		if mode == "aggregate-transactions" {
			config.Replay.MaxTransactions = 120
		}
		if mode == "block-lineage-drift" {
			third := sourceNextOwnedBlock(t, second.block, false)
			selected = third.header
			header := second.block.Header()
			header.Extra = []byte("another")
			changed := second.block.WithSeal(header)
			extensions = append(extensions, sourceOwnedBlockRPC(t, changed, false))
		}
	}
	if mode == "cursor-overflow" {
		selected.Number = ^uint64(0)
	}
	stateHash := config.Deployment.Hash
	if mode == "two-blocks" || mode == "resume-cursor" || mode == "resume-absent" {
		stateHash = selected.Hash
	}
	encoded, _, err := journalConfiguration(config)
	if err != nil {
		t.Fatal(err)
	}
	expected := JournalCheckpoint{Digest: journalDigest(encoded), Anchor: config.Deployment}
	if mode == "resume-cursor" || mode == "resume-absent" || mode == "caught-up-cursor" || mode == "cursor-overflow" {
		expected.Sequence = 1
		if mode == "resume-absent" {
			expected.Version = 1
		}
		if mode == "cursor-overflow" {
			expected.Anchor = selected
		}
	}
	limits := acquisitionTestLimits()
	limits.MaxRequests, limits.MaxWireBytes, limits.AcquisitionTimeout = 128, 16<<20, 30*time.Second
	limits.Node.Replay = config.Replay
	clockCalls := 0
	clock := func() ClockSample {
		clockCalls++
		sample := journalTestClock(selected, int64(clockCalls))
		if mode == "completion-clock" && clockCalls == 2 {
			sample.Epoch = "another-lifetime"
		}
		return sample
	}
	calls, finalized, canonical, middle := 0, 0, 0, 0
	base := reader.rpc.client.Transport
	methods := readABI(t)
	steps := replayTestSteps(t)
	reader.rpc.client.Transport = rpcRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(raw))
		var request struct {
			Method string
			Params []json.RawMessage
		}
		if json.Unmarshal(raw, &request) != nil {
			t.Fatal("owned request")
		}
		switch request.Method {
		case "eth_getBlockByNumber":
			var tag string
			_ = json.Unmarshal(request.Params[0], &tag)
			if tag != "0x0" {
				header := config.Deployment
				if tag == "finalized" || tag == "0x"+strconv.FormatUint(selected.Number, 16) && selected != config.Deployment {
					header = selected
				}
				if len(extensions) != 0 && tag == "0x"+strconv.FormatUint(extensions[0].header.Number, 16) {
					middle++
					header = extensions[0].header
					if mode == "block-lineage-drift" && middle >= 2 {
						header = extensions[1].header
					}
				}
				if tag == "finalized" {
					finalized++
					if mode == "finality-regression" && finalized > 1 {
						header.Number--
					}
				}
				if tag == "0x"+strconv.FormatUint(config.Deployment.Number, 16) {
					canonical++
					if mode == "changed-canonical" && canonical >= 4 {
						header.Hash = testParent
					}
				}
				if mode == "parent-gap" && backfill && tag != "finalized" && header == config.Deployment {
					header.Number--
				}
				return fixtureResult(map[string]any{"hash": header.Hash, "parentHash": header.ParentHash, "number": "0x" + strconv.FormatUint(header.Number, 16), "timestamp": "0x" + strconv.FormatUint(header.Timestamp, 16)}), nil
			}
		case "eth_getCode":
			if mode == "code-unavailable" {
				return fixtureResult(nil), nil
			}
			if mode == "changed-code" {
				return fixtureResult("0x00"), nil
			}
		case "debug_getRawBlock", "debug_getRawReceipts", "debug_traceBlockByHash":
			var hash string
			_ = json.Unmarshal(request.Params[0], &hash)
			for _, block := range extensions {
				if hash != block.header.Hash {
					continue
				}
				switch request.Method {
				case "debug_getRawBlock":
					return jsonResponse(string(block.blockRPC)), nil
				case "debug_getRawReceipts":
					return jsonResponse(string(block.receiptRPC)), nil
				default:
					return jsonResponse(string(block.rootRPC)), nil
				}
			}
		case "eth_call":
			var call struct{ To, Data string }
			if json.Unmarshal(request.Params[0], &call) != nil {
				t.Fatal("call framing")
			}
			data, err := hex.DecodeString(strings.TrimPrefix(call.Data, "0x"))
			if err != nil || len(data) < 4 {
				t.Fatal("call data")
			}
			method, err := methods.MethodById(data[:4])
			if err != nil {
				t.Fatal(err)
			}
			if method.Name != "registryId" {
				var pin map[string]any
				_ = json.Unmarshal(request.Params[1], &pin)
				if pin["blockHash"] != stateHash || pin["requireCanonical"] != true {
					t.Fatal("state/history did not use the expected interval stop block")
				}
			}
			var result string
			switch method.Name {
			case "record":
				switch mode {
				case "history-candidate":
					result = returnABI(t, "record", true, steps[1].Record)
				case "nonempty-absence":
					result = returnABI(t, "record", false, []byte("{}"))
				default:
					result = returnABI(t, "record", false, []byte{})
				}
			case "activeGrants":
				if mode != "history-candidate" {
					t.Fatal("CM-08 absent record must not query activeGrants")
				}
				result = returnABI(t, "activeGrants", []byte("[]"))
			case "historyItem":
				args, err := method.Inputs.Unpack(data[4:])
				if err != nil || args[0] != config.Agent {
					t.Fatal("history identity", err)
				}
				version := args[1].(uint64)
				if version < 1 || version > 2 {
					t.Fatal("history range", version)
				}
				step := steps[version-1]
				result = returnABI(t, "historyItem", step.History.Entry, step.History.Command)
			}
			if result != "" {
				return fixtureResult(result), nil
			}
		}
		return base.RoundTrip(req)
	})
	return reader.rpc, config, expected, limits, clock, &calls
}

func TestSourceBatchCollectsOneStopAndRetainsEveryExchange(t *testing.T) {
	for _, backfill := range []bool{false, true} {
		t.Run(strconv.FormatBool(backfill), func(t *testing.T) {
			client, config, expected, limits, clock, calls := sourceBatchFixture(t, backfill, "")
			got, err := client.CollectReplayBatch(context.Background(), config, expected, limits, clock)
			if err != nil || got == nil {
				t.Fatal(err)
			}
			if got.Batch.Anchor != config.Deployment || len(got.Batch.Blocks) != 1 || len(got.Batch.Parents) != map[bool]int{false: 0, true: 1}[backfill] || len(got.Exchanges) != *calls || len(got.Batch.Record) != 0 || len(got.Batch.Grants) != 0 || len(got.Batch.History) != 0 {
				t.Fatal("partial, unpinned or lost evidence")
			}
			if client.acquisition != nil || got.Completed.MonotonicMS <= got.Started.MonotonicMS || (got.Batch.Anchor != got.Batch.SelectedFinalized) != backfill {
				t.Fatal("scope escaped or backfill refreshed itself")
			}
			for _, exchange := range got.Exchanges {
				if !json.Valid(exchange.Response) || !json.Valid(exchange.Params) {
					t.Fatal("original exchange missing")
				}
			}
		})
	}
}

func TestSourceHistoryCollectionDoesNotReplaceJournalReplay(t *testing.T) {
	client, config, expected, limits, clock, _ := sourceBatchFixture(t, false, "history-candidate")
	got, err := client.CollectReplayBatch(context.Background(), config, expected, limits, clock)
	if err != nil || got == nil || len(got.Batch.History) != 2 || got.Batch.History[0].Version != 1 || got.Batch.History[1].Version != 2 {
		t.Fatal("complete pinned interval", err)
	}
	journal, _ := journalTestCreate(t, config)
	// The inert scenario has invented state without corresponding block effects.
	// Full replay must refuse; successful collection never grants that authority.
	if _, err := journal.Append(context.Background(), got.Batch, got.Started, got.Completed); err == nil {
		t.Fatal("transport collection substituted for command replay")
	}
}

func TestSourceBatchRefusesDriftAndBudgetWithoutPartialOutput(t *testing.T) {
	for _, mode := range []string{"parent-gap", "changed-code", "code-unavailable", "changed-canonical", "finality-regression", "nonempty-absence", "completion-clock", "request-budget", "byte-budget", "ancestry-budget"} {
		t.Run(mode, func(t *testing.T) {
			client, config, expected, limits, clock, calls := sourceBatchFixture(t, mode == "parent-gap" || mode == "ancestry-budget", mode)
			switch mode {
			case "request-budget":
				limits.MaxRequests = 20
			case "byte-budget":
				limits.MaxWireBytes = 1
			}
			if got, err := client.CollectReplayBatch(context.Background(), config, expected, limits, clock); got != nil || err == nil {
				t.Fatal("partial or inconsistent acquisition accepted")
			}
			if *calls > limits.MaxRequests {
				t.Fatal("per-node counter reset the aggregate bound")
			}
		})
	}
}

func TestSourceRecordVersionUsesCanonicalStopBytes(t *testing.T) {
	config := journalTestConfig(t, replayTestInput(t))
	steps := replayTestSteps(t)
	if version, err := sourceRecordVersion(config, false, nil, nil); err != nil || version != 0 {
		t.Fatal("internal absent state", version, err)
	}
	if _, err := sourceRecordVersion(config, false, nil, []byte("[]")); !errors.Is(err, ErrRPCState) {
		t.Fatal("fabricated grants read for absent identity", err)
	}
	if version, err := sourceRecordVersion(config, true, steps[1].Record, steps[1].Grants); err != nil || version != 2 {
		t.Fatal(version, err)
	}
	for _, mode := range []string{"no-record", "empty", "oversize", "bad-json", "wrong-id", "numeric-version", "zero-version", "leading-zero", "overflow", "noncanonical", "bad-grants"} {
		t.Run(mode, func(t *testing.T) {
			record, grants := bytes.Clone(steps[1].Record), bytes.Clone(steps[1].Grants)
			var m map[string]any
			_ = json.Unmarshal(record, &m)
			switch mode {
			case "no-record":
				m = nil
			case "empty":
				record = nil
			case "oversize":
				record = bytes.Repeat([]byte(" "), 65537)
			case "bad-json":
				record = []byte("{")
			case "wrong-id":
				m["id"] = "did:example:another"
			case "numeric-version":
				m["version"] = 2
			case "zero-version":
				m["version"] = "0"
			case "leading-zero":
				m["version"] = "02"
			case "overflow":
				m["version"] = "18446744073709551616"
			case "noncanonical":
				record = append([]byte(" "), record...)
			case "bad-grants":
				grants = []byte("[ ]")
			}
			if mode != "bad-json" && mode != "noncanonical" && mode != "empty" && mode != "oversize" {
				record = replayTestJSON(t, m)
			}
			if _, err := sourceRecordVersion(config, true, record, grants); !errors.Is(err, ErrRPCState) {
				t.Fatal("ambiguous current version accepted", err)
			}
		})
	}
}

func TestSourceBatchRejectsUnboundConfigurationBeforeAnyRequest(t *testing.T) {
	client, config, expected, limits, clock, calls := sourceBatchFixture(t, false, "")
	for name, edit := range map[string]func(*JournalConfig, *JournalCheckpoint, *SourceReadLimits){
		"registry":          func(c *JournalConfig, _ *JournalCheckpoint, _ *SourceReadLimits) { c.Registry = "another" },
		"code-identity":     func(c *JournalConfig, _ *JournalCheckpoint, _ *SourceReadLimits) { c.CodeIdentity = testBlock },
		"abi-identity":      func(c *JournalConfig, _ *JournalCheckpoint, _ *SourceReadLimits) { c.ABIIdentity = testBlock },
		"deployment-hash":   func(c *JournalConfig, _ *JournalCheckpoint, _ *SourceReadLimits) { c.Deployment.Hash = testBlock },
		"deployment-number": func(c *JournalConfig, _ *JournalCheckpoint, _ *SourceReadLimits) { c.Deployment.Number++ },
		"different-budget":  func(_ *JournalConfig, _ *JournalCheckpoint, l *SourceReadLimits) { l.Node.Replay.MaxBlocks++ },
		"no-batches-left":   func(c *JournalConfig, p *JournalCheckpoint, _ *SourceReadLimits) { p.Sequence = uint64(c.MaxBatches) },
		"no-checkpoint":     func(_ *JournalConfig, p *JournalCheckpoint, _ *SourceReadLimits) { p.Digest = "" },
		"wrong-initial-hash": func(_ *JournalConfig, p *JournalCheckpoint, _ *SourceReadLimits) {
			p.Digest = testBlock
		},
		"initial-version": func(_ *JournalConfig, p *JournalCheckpoint, _ *SourceReadLimits) { p.Version = 1 },
		"before-baseline": func(c *JournalConfig, p *JournalCheckpoint, _ *SourceReadLimits) {
			p.Anchor.Number = c.Deployment.Number - 1
		},
		"invalid-node":  func(_ *JournalConfig, _ *JournalCheckpoint, l *SourceReadLimits) { l.Node.MaxRequests = 0 },
		"invalid-total": func(_ *JournalConfig, _ *JournalCheckpoint, l *SourceReadLimits) { l.MaxRequests = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			c, p, l := config, expected, limits
			edit(&c, &p, &l)
			if got, err := client.CollectReplayBatch(context.Background(), c, p, l, clock); got != nil || err == nil || *calls != 0 {
				t.Fatal("unbound configuration performed work", err)
			}
		})
	}
	if got, err := client.CollectReplayBatch(context.Background(), config, expected, limits, nil); got != nil || !errors.Is(err, ErrConfiguration) {
		t.Fatal("no clock", err)
	}
	if got, err := client.CollectReplayBatch(context.Background(), config, expected, limits, func() ClockSample { return ClockSample{} }); got != nil || !errors.Is(err, ErrClock) || *calls != 0 {
		t.Fatal("unbound initial clock", err)
	}
	var missing *ReadRPC
	if got, err := missing.CollectReplayBatch(context.Background(), config, expected, limits, clock); got != nil || !errors.Is(err, ErrConfiguration) {
		t.Fatal("missing reader", err)
	}
	//nolint:staticcheck // Verify refusal of an explicitly missing context.
	if got, err := client.CollectReplayBatch(nil, config, expected, limits, clock); got != nil || !errors.Is(err, ErrConfiguration) {
		t.Fatal("no context", err)
	}
}
