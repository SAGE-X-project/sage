package registry011

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

type sourceOwnedBlock struct {
	block                *types.Block
	header               FinalizedHeader
	blockRPC, receiptRPC []byte
	rootRPC              []byte
}

func sourceRecordedBlock(t *testing.T) *types.Block {
	t.Helper()
	var response struct{ Result string }
	if json.Unmarshal(nodeRecorded(t)["raw-block"], &response) != nil {
		t.Fatal("recorded block envelope")
	}
	raw, err := hex.DecodeString(response.Result[2:])
	var block types.Block
	if err != nil || rlp.DecodeBytes(raw, &block) != nil {
		t.Fatal("recorded block")
	}
	return &block
}

// These blocks are explicitly synthetic content fixtures. In particular, the
// repeated-transaction variant is only a cumulative-budget refusal scenario;
// no claim is made that it could be a consensus-valid chain extension.
func sourceNextOwnedBlock(t *testing.T, previous *types.Block, repeat bool) sourceOwnedBlock {
	t.Helper()
	header := previous.Header()
	header.ParentHash = previous.Hash()
	header.Number = new(big.Int).Add(previous.Number(), big.NewInt(1))
	header.Time += 12
	header.Extra = []byte("owned")
	body := *previous.Body()
	if !repeat {
		header.TxHash, header.ReceiptHash = types.EmptyTxsHash, types.EmptyReceiptsHash
		header.GasUsed, header.Bloom = 0, types.Bloom{}
		body.Transactions = nil
		if header.BlobGasUsed != nil {
			header.BlobGasUsed = new(uint64)
		}
	}
	block := types.NewBlockWithHeader(header).WithBody(body)
	return sourceOwnedBlockRPC(t, block, repeat)
}

func sourceOwnedBlockRPC(t *testing.T, block *types.Block, repeat bool) sourceOwnedBlock {
	t.Helper()
	raw, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal(err)
	}
	blockRPC, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "result": nodeHex(raw)})
	if err != nil {
		t.Fatal(err)
	}
	empty := []byte(`{"jsonrpc":"2.0","id":1,"result":[]}`)
	got := sourceOwnedBlock{block: block, header: replayHeader(block.Header()), blockRPC: blockRPC, receiptRPC: empty, rootRPC: empty}
	if repeat {
		got.receiptRPC = nodeRecorded(t)["raw-receipts"]
		got.rootRPC = rootGasWire(t)
	}
	return got
}

func TestSourceBatchCollectsConsecutiveBodiesAndResumesOnlyAfterCursor(t *testing.T) {
	for _, mode := range []string{"two-blocks", "resume-cursor"} {
		t.Run(mode, func(t *testing.T) {
			client, config, expected, limits, clock, calls := sourceBatchFixture(t, false, mode)
			got, err := client.CollectReplayBatch(context.Background(), config, expected, limits, clock)
			if err != nil || got == nil {
				t.Fatal(err)
			}
			want := 2
			if mode == "resume-cursor" {
				want = 1
			}
			if len(got.Batch.Blocks) != want || len(got.Batch.Parents) != 1 || got.Batch.Anchor != got.Batch.SelectedFinalized || got.Batch.Anchor.Number != config.Deployment.Number+1 || len(got.Exchanges) != *calls {
				t.Fatal("consecutive interval or aggregate evidence changed")
			}
			work := replayBudget{limits: config.Replay}
			for i, block := range got.Batch.Blocks {
				decoded, err := replayDecodeBlock(context.Background(), block, &work)
				if err != nil || decoded.header.Number.Uint64() != config.Deployment.Number+uint64(2-want+i) {
					t.Fatal("body order", err)
				}
			}
			if mode == "resume-cursor" {
				for _, exchange := range got.Exchanges {
					if exchange.Method == "debug_getRawBlock" {
						var hashes []string
						if json.Unmarshal(exchange.Params, &hashes) != nil || len(hashes) != 1 || hashes[0] == config.Deployment.Hash {
							t.Fatal("old accepted body was fetched again")
						}
					}
				}
			}
		})
	}
}

func TestSourceBatchRefusesBodyLineageAndCumulativeTransactionDrift(t *testing.T) {
	for _, mode := range []string{"block-lineage-drift", "aggregate-transactions", "caught-up-cursor", "cursor-overflow"} {
		t.Run(mode, func(t *testing.T) {
			client, config, expected, limits, clock, calls := sourceBatchFixture(t, false, mode)
			if got, err := client.CollectReplayBatch(context.Background(), config, expected, limits, clock); got != nil || err == nil || *calls > limits.MaxRequests {
				t.Fatal("partial interval or reset cumulative bound accepted", err)
			}
		})
	}
}
