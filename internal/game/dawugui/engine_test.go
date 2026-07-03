package dawugui

import (
	"testing"

	"github.com/example/game/internal/game/engine"
)

func TestDiamond3FirstPlay(t *testing.T) {
	e := New()
	players := []struct {
		uid, seat uint64
	}{
		{1, 0}, {2, 1}, {3, 2},
	}
	var ps []engine.Player
	for _, p := range players {
		ps = append(ps, engine.Player{UserID: p.uid, Seat: uint32(p.seat)})
	}
	st, _, err := e.NewState(engine.GameConfig{GameID: GameID, Players: 3}, ps)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = e.ApplyAction(st, engine.Action{Kind: engine.ActionPass, Seat: 0})
	if err == nil {
		t.Fatal("expected pass error on first play")
	}
}

func TestPlayRankBeats(t *testing.T) {
	if cardRank(3*13+0) >= cardRank(3*13+1) {
		t.Fatal("diamond 3 should rank lower than diamond 4")
	}
}
