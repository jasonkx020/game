package bot

import (
	"testing"

	"github.com/example/game/internal/game/engine"
	"github.com/example/game/internal/game/liuzichong"
)

func TestPickLiuzichongPrefersCapture(t *testing.T) {
	st := &liuzichong.State{
		Phase: 1, CurrentSeat: 0,
		Players: []engine.Player{{UserID: 1, Seat: 0}, {UserID: 2, Seat: 1}},
	}
	st.Board[1][0] = 1
	st.Board[1][2] = 2
	st.Board[2][1] = 1
	action, ok := PickLiuzichongAction(st, 0)
	if !ok {
		t.Fatal("expected a move")
	}
	if action.ToRow != 1 || action.ToCol != 1 {
		t.Fatalf("expected capture move to (1,1), got (%d,%d)", action.ToRow, action.ToCol)
	}
	cap := liuzichong.PreviewCaptureCount(st, action)
	if cap < 1 {
		t.Fatal("expected capture")
	}
}

func TestPickLiuzichongOpening(t *testing.T) {
	st := &liuzichong.State{Phase: 1, CurrentSeat: 0}
	// init via NewState board layout
	eng := liuzichong.New()
	state, _, err := eng.NewState(engine.GameConfig{GameID: liuzichong.GameID, Players: 2}, []engine.Player{
		{UserID: 1, Seat: 0}, {UserID: 2, Seat: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	st = state.(*liuzichong.State)
	action, ok := PickLiuzichongAction(st, 0)
	if !ok {
		t.Fatal("black should have opening moves")
	}
	if action.Kind != engine.ActionMove {
		t.Fatal("expected move")
	}
}
