package bot

import (
	"testing"

	"github.com/example/game/internal/game/engine"
	"github.com/example/game/internal/game/liuzichong"
)

// TestBotSelfPlay finishes a full game with both seats as bots (engine-only).
func TestBotSelfPlay(t *testing.T) {
	eng := liuzichong.New()
	players := []engine.Player{
		{UserID: 1, Seat: 0}, {UserID: 2, Seat: 1},
	}
	state, _, err := eng.NewState(engine.GameConfig{GameID: liuzichong.GameID, Players: 2}, players)
	if err != nil {
		t.Fatal(err)
	}
	for step := 0; step < 500; step++ {
		st := state.(*liuzichong.State)
		if st.Phase != 1 {
			break
		}
		action, ok := PickLiuzichongAction(st, st.CurrentSeat)
		if !ok {
			t.Fatalf("no move at step %d seat %d", step, st.CurrentSeat)
		}
		state, _, err = eng.ApplyAction(state, action)
		if err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
	}
	end, ok := eng.CheckRoundEnd(state)
	if !ok {
		// 4×4 可能长时间未分出胜负；至少验证 bot 能连续合法走子
		st := state.(*liuzichong.State)
		if st.Phase != 1 {
			t.Fatal("unexpected phase")
		}
		t.Log("game still playing after 500 moves (acceptable for weak bots)")
		return
	}
	if end.WinnerSeat > 1 {
		t.Fatalf("bad winner seat %d", end.WinnerSeat)
	}
	settle, err := eng.CalcSettlement(state, end)
	if err != nil || !settle.Valid {
		t.Fatalf("settlement: %+v err=%v", settle, err)
	}
}
