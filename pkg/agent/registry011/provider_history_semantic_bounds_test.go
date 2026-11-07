package registry011

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestProviderHistorySemanticFoldFundsHistoryAndProofsBeforeDerivation(t *testing.T) {
	for _, mode := range []string{"whole-items", "window-items", "queue-headroom", "proofs", "window-blocks", "window-count"} {
		t.Run(mode, func(t *testing.T) {
			_, in := historySemanticFixture(t, 11)
			c := journalTestConfig(t, in)
			maximumItems := uint64(11)
			switch mode {
			case "whole-items":
				maximumItems = 1
			case "window-items":
				c.Replay.History.MaxItems = 1
			case "queue-headroom":
				c.Replay.History.MaxBytes = 262144 + 65536 + 262144 + len(in.History[0].Command) - 1
			case "proofs":
				c.Replay.MaxProofs = 1 // Normal create carries two required proofs.
			case "window-blocks":
				c.Replay.MaxBlocks = 1
			case "window-count":
				c.MaxBatches = 1
			}
			f, err := newProviderHistorySemanticFold("0x"+strings.Repeat("a", 64), c, in.Anchor, uint64(len(in.Blocks)), maximumItems)
			if err != nil {
				t.Fatal("separate before-derivation bounded DATA configuration", err)
			}
			if mode == "whole-items" || mode == "window-count" {
				end := 1
				if mode == "window-count" {
					end = 0
				}
				historySemanticApply(t, f, in, 0, end)
				historySemanticCompare(t, f, in, 0, end)
				confirmedVersion, claims := f.confirmed.state.version, len(f.confirmed.claims)
				err = f.beginWindowCandidate(context.Background())
				if mode == "whole-items" {
					if err != nil {
						t.Fatal(err)
					}
					err = f.applyBlockCandidate(context.Background(), in.Blocks[2], journalTestHeader(t, in.Blocks[2]))
				}
				if err == nil || !f.failed || f.confirmed.state.version != confirmedVersion || len(f.confirmed.claims) != claims {
					t.Fatal("window boundary renewed lifetime history/index or window allowance")
				}
				return
			}
			historySemanticApply(t, f, in, 0, 0)
			if mode == "window-items" {
				if err := f.applyBlockCandidate(context.Background(), in.Blocks[1], journalTestHeader(t, in.Blocks[1])); err != nil {
					t.Fatal("one allowed subject transition", err)
				}
				err = f.applyBlockCandidate(context.Background(), in.Blocks[2], journalTestHeader(t, in.Blocks[2]))
			} else {
				err = f.applyBlockCandidate(context.Background(), in.Blocks[1], journalTestHeader(t, in.Blocks[1]))
			}
			if err == nil || !f.failed || f.confirmed.state.version != 0 {
				t.Fatal("insufficient scalar budget admitted a semantic state")
			}
			if (mode == "queue-headroom" || mode == "proofs" || mode == "window-blocks") && f.window.run.state.version != 0 {
				t.Fatal("capacity refusal occurred after the subject state/crypto step")
			}
			if (mode == "queue-headroom" || mode == "window-blocks") && f.window.budget.proofs != 0 {
				t.Fatal("known byte/block insufficiency consumed crypto proof work")
			}
		})
	}
}

func TestProviderHistorySemanticFoldKeepsSharedClaimLimitsAcrossWindows(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		name := "distinct-over-limit"
		if duplicate {
			name = "duplicate-prior-claim"
		}
		t.Run(name, func(t *testing.T) {
			normal := replayTestInput(t)
			in := ReplayInput{Registry: normal.Registry, Agent: normal.Agent, DeploymentBlock: 0,
				Blocks: []ReplayBlock{normal.Blocks[0]}, Anchor: journalTestHeader(t, normal.Blocks[0])}
			first := normal.Blocks[0].Executions[0].Root.Calls[0]
			var digest [32]byte
			copy(digest[:], first.Input[4:])
			if !duplicate {
				digest[0] ^= 1 // A second ordinary opaque commitment, not a key or exploit.
			}
			a, out, err := replayABIs()
			if err != nil {
				t.Fatal(err)
			}
			call, _ := a.Pack("commitClaim", digest)
			event := out.Events["ClaimCommitted"]
			data, _ := event.Inputs.NonIndexed().Pack(big.NewInt(1))
			log := &types.Log{Address: common.HexToAddress(SepoliaRegistryAddress),
				Topics: []common.Hash{event.ID, common.BytesToHash(common.HexToAddress(replayTestWallet).Bytes()), common.Hash(digest)}, Data: data}
			replayTestAppendBlock(t, &in, call, log, replayTestWallet, 1)
			c := journalTestConfig(t, in)
			c.MaxClaims = 1
			f, err := newProviderHistorySemanticFold("0x"+strings.Repeat("a", 64), c, in.Anchor, 2, 1)
			if err != nil {
				t.Fatal(err)
			}
			historySemanticApply(t, f, in, 0, 0)
			historySemanticCompare(t, f, in, 0, 0)
			if f.beginWindowCandidate(context.Background()) != nil {
				t.Fatal("second window")
			}
			if f.applyBlockCandidate(context.Background(), in.Blocks[1], journalTestHeader(t, in.Blocks[1])) == nil || !f.failed || len(f.confirmed.claims) != 1 {
				t.Fatal("new window erased shared claim bound or duplicate tombstone")
			}
		})
	}
}

func TestProviderHistorySemanticFoldRefusesInvalidSelectionAndCallOrdering(t *testing.T) {
	for _, mode := range []string{"program", "span", "blocks-zero", "items-large", "journal", "selected-before-base", "selected-too-far"} {
		t.Run(mode, func(t *testing.T) {
			in := replayTestInput(t)
			c, selected := journalTestConfig(t, in), in.Anchor
			program, blocks, items := "0x"+strings.Repeat("a", 64), uint64(len(in.Blocks)), uint64(11)
			switch mode {
			case "program":
				program = "unknown"
			case "span":
				blocks--
			case "blocks-zero":
				blocks = 0
			case "items-large":
				items = 1<<20 + 1
			case "journal":
				c.MaxClaims = 0
			case "selected-before-base":
				c.Deployment = selected
				selected.Number--
			case "selected-too-far":
				c.Observation.MaxAncestryHeaders = 1
			}
			if f, err := newProviderHistorySemanticFold(program, c, selected, blocks, items); err == nil || f != nil {
				t.Fatal("invalid semantic DATA selection accepted")
			}
		})
	}
	for _, mode := range []string{"nil-context", "empty-window", "repeat-window", "wrong-first", "skip-height", "wrong-final", "oversized-record", "oversized-history"} {
		t.Run(mode, func(t *testing.T) {
			f, in := historySemanticFixture(t, 11)
			var err error
			if mode == "nil-context" {
				var absentContext context.Context
				err = f.beginWindowCandidate(absentContext)
			} else {
				if f.beginWindowCandidate(context.Background()) != nil {
					t.Fatal("normal window")
				}
				switch mode {
				case "empty-window":
					err = f.finishWindowCandidate(context.Background())
				case "repeat-window":
					err = f.beginWindowCandidate(context.Background())
				case "wrong-first":
					err = f.applyBlockCandidate(context.Background(), in.Blocks[1], journalTestHeader(t, in.Blocks[1]))
				default:
					if f.applyBlockCandidate(context.Background(), in.Blocks[0], journalTestHeader(t, in.Blocks[0])) != nil {
						t.Fatal("normal first block")
					}
					switch mode {
					case "skip-height":
						err = f.applyBlockCandidate(context.Background(), in.Blocks[2], journalTestHeader(t, in.Blocks[2]))
					case "wrong-final":
						want := f.selection.selected
						want.Hash = "0x" + strings.Repeat("f", 64)
						err = f.applyBlockCandidate(context.Background(), in.Blocks[len(in.Blocks)-1], want)
					case "oversized-record":
						err = f.compareStateCandidate(context.Background(), false, make([]byte, 65537), nil)
					case "oversized-history":
						if f.compareStateCandidate(context.Background(), false, nil, nil) != nil {
							t.Fatal("normal no-create stop")
						}
						err = f.compareHistoryItemCandidate(context.Background(), 1, make([]byte, 262145), []byte(`{}`))
					}
				}
			}
			if err == nil || !f.failed || f.atEnd() {
				t.Fatal("invalid ordering/selection produced complete reusable DATA", mode)
			}
		})
	}
	var absent *providerHistorySemanticFold
	if absent.beginWindowCandidate(context.Background()) == nil || absent.atEnd() {
		t.Fatal("nil fold accepted work")
	}
}
