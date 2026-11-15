package settlement

import (
	"testing"

	"github.com/example/game/internal/game/engine"
)

func TestComputeOnlyRuleTimesBaseAndRake(t *testing.T) {
	settle := engine.SettlementResult{
		Valid:    true,
		WinnerID: 1,
		Scores: []engine.PlayerScore{
			{UserID: 1, Seat: 0, RuleScore: 15},
			{UserID: 2, Seat: 1, RuleScore: -5},
			{UserID: 3, Seat: 2, RuleScore: -5},
			{UserID: 4, Seat: 3, RuleScore: -5},
		},
	}
	res := ComputeOnly(500, settle, nil)
	if res.RakeAmount != 375 { // 15*500 * 5%
		t.Fatalf("rake=%d want 375", res.RakeAmount)
	}
	byUser := map[uint64]int64{}
	for _, c := range res.CoinDeltas {
		byUser[c.UserID] = c.Delta
	}
	if byUser[1] != 7500-375 {
		t.Fatalf("winner delta=%d", byUser[1])
	}
	if byUser[2] != -2500 {
		t.Fatalf("loser delta=%d", byUser[2])
	}
}

func TestComputeOnlySkipsBots(t *testing.T) {
	settle := engine.SettlementResult{
		Valid:    true,
		WinnerID: 1,
		Scores: []engine.PlayerScore{
			{UserID: 1, Seat: 0, RuleScore: 3},
			{UserID: 9_000_000_000_000_001, Seat: 1, RuleScore: -1},
		},
	}
	res := ComputeOnly(100, settle, func(uid uint64) bool { return uid >= 9_000_000_000_000_001 })
	if len(res.CoinDeltas) != 1 || res.CoinDeltas[0].UserID != 1 {
		t.Fatalf("coins=%+v", res.CoinDeltas)
	}
}

func TestComputeOnlyRoomCardNoOpPathStillComputes(t *testing.T) {
	// room_card handlers skip ApplyScoreField; ComputeOnly is for unit math only.
	settle := engine.SettlementResult{Valid: true, WinnerID: 1, Scores: []engine.PlayerScore{{UserID: 1, RuleScore: 1}}}
	res := ComputeOnly(1, settle, nil)
	if res.RakeAmount != 0 { // 1*1*5%=0
		t.Fatalf("rake=%d", res.RakeAmount)
	}
}
