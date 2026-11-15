package rules

import (
	"testing"

	"github.com/example/game/internal/game/engine"
)

func TestPickPrefersCapture(t *testing.T) {
	st := &State{
		Phase: phasePlaying, CurrentSeat: 0,
		Players: []engine.Player{{UserID: 1, Seat: 0}, {UserID: 2, Seat: 1}},
	}
	st.Board[1][0] = 1
	st.Board[1][2] = 2
	st.Board[2][1] = 1
	action, ok := pickAction(st, 0)
	if !ok {
		t.Fatal("expected a move")
	}
	if action.ToRow != 1 || action.ToCol != 1 {
		t.Fatalf("expected capture move to (1,1), got (%d,%d)", action.ToRow, action.ToCol)
	}
	cap := PreviewCaptureCount(st, action)
	if cap < 1 {
		t.Fatal("expected capture")
	}
}

func TestPickOpening(t *testing.T) {
	eng := New()
	state, _, err := eng.NewState(engine.GameConfig{GameID: GameID, Players: 2}, []engine.Player{
		{UserID: 1, Seat: 0}, {UserID: 2, Seat: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	st := state.(*State)
	action, ok := pickAction(st, 0)
	if !ok {
		t.Fatal("black should have opening moves")
	}
	if action.Kind != engine.ActionMove {
		t.Fatal("expected move")
	}
}

func TestBotSelfPlay(t *testing.T) {
	eng := New()
	players := []engine.Player{
		{UserID: 1, Seat: 0}, {UserID: 2, Seat: 1},
	}
	state, _, err := eng.NewState(engine.GameConfig{GameID: GameID, Players: 2}, players)
	if err != nil {
		t.Fatal(err)
	}
	for step := 0; step < 500; step++ {
		st := state.(*State)
		if st.Phase != phasePlaying {
			break
		}
		action, ok := pickAction(st, st.CurrentSeat)
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
		st := state.(*State)
		if st.Phase != phasePlaying {
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
