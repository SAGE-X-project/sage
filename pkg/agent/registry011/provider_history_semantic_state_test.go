package registry011

import "testing"

func TestProviderHistorySemanticCloneOwnsEveryMutableReplayField(t *testing.T) {
	expires := uint64(100)
	id := replayClaimID{actor: replayTestWallet}
	before := replayStream{state: replayState{version: 1,
		record: replayRecord{Keys: []replayKey{{Name: "a", Expires: &expires}}, Services: []replayService{{"api", "A2A", "https://example.invalid"}}},
		grants: []replayGrant{{replayTestOperator, "activate"}}},
		claims: map[replayClaimID]replayClaimState{id: {block: 1}}, created: map[string]bool{replayTestAgent: true}}
	copyState := providerHistoryCloneSemanticState(before)
	copyState.state.record.Keys[0].Name = "copy"
	*copyState.state.record.Keys[0].Expires = 200
	copyState.state.record.Services[0].Name = "copy"
	copyState.state.grants[0].Scope = "copy"
	copyState.claims[id] = replayClaimState{block: 2, consumed: true}
	copyState.created[replayTestAgent] = false
	if before.state.record.Keys[0].Name != "a" || *before.state.record.Keys[0].Expires != 100 || before.state.record.Services[0].Name != "api" ||
		before.state.grants[0].Scope != "activate" || before.claims[id].block != 1 || before.claims[id].consumed || !before.created[replayTestAgent] {
		t.Fatal("new semantic window aliases a confirmed record, expiry, grants or global index")
	}
}
