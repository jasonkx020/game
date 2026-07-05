package liuzichong

import (
	"testing"

	"github.com/example/game/internal/game/engine"
)

func TestNewState(t *testing.T) {
	eng := New()
	players := []engine.Player{
		{UserID: 1, Seat: 0, Nickname: "black"},
		{UserID: 2, Seat: 1, Nickname: "white"},
	}
	st, events, err := eng.NewState(engine.GameConfig{GameID: GameID, Players: 2}, players)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	s := st.(*State)
	if s.Board[0][0] != 1 || s.Board[3][0] != 2 {
		t.Fatal("initial board mismatch")
	}
}

func TestMoveAndCapture(t *testing.T) {
	eng := New()
	st := &State{
		Phase: phasePlaying, CurrentSeat: 0,
		Players: []engine.Player{{UserID: 1, Seat: 0}, {UserID: 2, Seat: 1}},
	}
	// row1: B, empty, W — move (2,1)->(1,1) forms B-B-W and eats W at (1,2)
	st.Board[1][0] = 1
	st.Board[1][2] = 2
	st.Board[1][3] = 0
	st.Board[2][1] = 1
	newSt, events, err := eng.ApplyAction(st, engine.Action{
		Kind: engine.ActionMove, Seat: 0,
		FromRow: 2, FromCol: 1, ToRow: 1, ToCol: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	ns := newSt.(*State)
	if ns.Board[1][2] != 0 {
		t.Fatal("expected captured piece removed")
	}
	if len(events) < 1 {
		t.Fatal("expected move event")
	}
}

func TestWinByPieceCount(t *testing.T) {
	eng := New()
	st := &State{
		Phase: phasePlaying, CurrentSeat: 0,
		Players: []engine.Player{{UserID: 1, Seat: 0}, {UserID: 2, Seat: 1}},
	}
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			st.Board[r][c] = 0
		}
	}
	st.Board[0][0] = 1
	st.Board[0][1] = 1
	st.Board[3][3] = 2
	// black moves; white still has 1 piece while black has 2 -> black wins
	newSt, _, err := eng.ApplyAction(st, engine.Action{
		Kind: engine.ActionMove, Seat: 0,
		FromRow: 0, FromCol: 0, ToRow: 1, ToCol: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	ns := newSt.(*State)
	if ns.Phase != phaseEnded {
		t.Fatal("expected game ended")
	}
	end, ok := eng.CheckRoundEnd(ns)
	if !ok || end.WinnerSeat != 0 {
		t.Fatalf("expected black wins, got %+v", end)
	}
}

func TestInvalidMove(t *testing.T) {
	eng := New()
	st := &State{
		Phase: phasePlaying, CurrentSeat: 0,
		Players: []engine.Player{{UserID: 1, Seat: 0}, {UserID: 2, Seat: 1}},
	}
	initBoard(&st.Board)
	_, _, err := eng.ApplyAction(st, engine.Action{
		Kind: engine.ActionMove, Seat: 0,
		FromRow: 0, FromCol: 0, ToRow: 2, ToCol: 2,
	})
	if err == nil {
		t.Fatal("expected error for invalid move")
	}
}

// TestEndToEndFlow 引擎级验收：开局 Push 事件 → 吃子 → 终局结算
func TestEndToEndFlow(t *testing.T) {
	eng := New()
	players := []engine.Player{
		{UserID: 1, Seat: 0, Nickname: "black"},
		{UserID: 2, Seat: 1, Nickname: "white"},
	}
	st, initEvents, err := eng.NewState(engine.GameConfig{GameID: GameID, Players: 2}, players)
	if err != nil {
		t.Fatal(err)
	}
	if len(initEvents) != 2 || initEvents[0].PushRoute != "onBoardInit" || initEvents[1].PushRoute != "onTurnNotify" {
		t.Fatalf("unexpected init events: %+v", initEvents)
	}

	// 吃子步
	captureSt := &State{
		Phase: phasePlaying, CurrentSeat: 0,
		Players: players,
	}
	captureSt.Board[1][0] = 1
	captureSt.Board[1][2] = 2
	captureSt.Board[2][1] = 1
	st2, moveEvents, err := eng.ApplyAction(captureSt, engine.Action{
		Kind: engine.ActionMove, Seat: 0,
		FromRow: 2, FromCol: 1, ToRow: 1, ToCol: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if st2.(*State).Board[1][2] != 0 {
		t.Fatal("capture failed in flow")
	}
	if len(moveEvents) < 1 || moveEvents[0].PushRoute != "onMoveResult" {
		t.Fatalf("expected onMoveResult, got %+v", moveEvents)
	}

	// 终局 + 结算
	endSt := &State{
		Phase: phasePlaying, CurrentSeat: 0,
		Players: players,
	}
	endSt.Board[0][0] = 1
	endSt.Board[0][1] = 1
	endSt.Board[3][3] = 2
	final, _, err := eng.ApplyAction(endSt, engine.Action{
		Kind: engine.ActionMove, Seat: 0,
		FromRow: 0, FromCol: 0, ToRow: 1, ToCol: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	end, ok := eng.CheckRoundEnd(final)
	if !ok || end.WinnerSeat != 0 {
		t.Fatalf("expected black wins, got %+v", end)
	}
	settle, err := eng.CalcSettlement(final, end)
	if err != nil {
		t.Fatal(err)
	}
	if !settle.Valid || settle.WinnerID != 1 {
		t.Fatalf("bad settlement: %+v", settle)
	}
	_ = st // initial state used for NewState validation
}
