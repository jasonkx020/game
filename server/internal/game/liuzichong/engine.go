package liuzichong

import (
	"fmt"
	"time"

	"github.com/example/game/internal/game/engine"
	pb "github.com/example/game/internal/gen/pitaya/pitaya"
	"google.golang.org/protobuf/proto"
)

const GameID = "liuzichong"
const boardSize = 4

const (
	phaseIdle = iota
	phasePlaying
	phaseEnded
)

type State struct {
	Phase        int
	Players      []engine.Player
	Board        [boardSize][boardSize]int // 0=empty 1=black 2=white
	CurrentSeat  uint32
	WinnerSeat   uint32
	TurnDeadline time.Time
}

const turnTimeout = 60 * time.Second

func (s *State) GameID() string { return GameID }

func (s *State) Clone() engine.GameState {
	cp := *s
	return &cp
}

type Engine struct{}

func New() *Engine { return &Engine{} }

func (e *Engine) Meta() engine.GameMeta {
	return engine.GameMeta{GameID: GameID, MinPlayers: 2, MaxPlayers: 2}
}

func (e *Engine) NewState(cfg engine.GameConfig, players []engine.Player) (engine.GameState, []engine.GameEvent, error) {
	if len(players) != 2 {
		return nil, nil, fmt.Errorf("liuzichong requires 2 players, got %d", len(players))
	}
	st := &State{
		Phase:        phasePlaying,
		Players:      append([]engine.Player(nil), players...),
		CurrentSeat:  players[0].Seat,
		TurnDeadline: time.Now().Add(turnTimeout),
	}
	initBoard(&st.Board)

	cells := flattenBoard(st.Board)
	boardEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_BoardInit{BoardInit: &pb.BoardInitEvent{
		Cells: cells, FirstSeat: st.CurrentSeat,
	}}})
	turnEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Turn{Turn: &pb.TurnEvent{
		CurrentSeat: st.CurrentSeat, TimeoutMs: uint32(turnTimeout / time.Millisecond),
	}}})

	events := []engine.GameEvent{
		{Type: engine.EventBoardInit, PushRoute: "onBoardInit", Payload: boardEv},
		{Type: engine.EventTurn, PushRoute: "onTurnNotify", Payload: turnEv},
	}
	return st, events, nil
}

func (e *Engine) ApplyAction(state engine.GameState, action engine.Action) (engine.GameState, []engine.GameEvent, error) {
	st, ok := state.(*State)
	if !ok {
		return nil, nil, fmt.Errorf("invalid state")
	}
	if st.Phase != phasePlaying {
		return nil, nil, fmt.Errorf("not in playing phase")
	}
	if action.Kind != engine.ActionMove {
		return nil, nil, fmt.Errorf("unsupported action")
	}
	if action.Seat != st.CurrentSeat {
		return nil, nil, fmt.Errorf("not your turn")
	}

	color := seatColor(action.Seat)
	fr, fc, tr, tc := action.FromRow, action.FromCol, action.ToRow, action.ToCol
	if !inBounds(fr, fc) || !inBounds(tr, tc) {
		return nil, nil, fmt.Errorf("out of bounds")
	}
	if st.Board[fr][fc] != color {
		return nil, nil, fmt.Errorf("not your piece")
	}
	if st.Board[tr][tc] != 0 {
		return nil, nil, fmt.Errorf("target not empty")
	}
	dr, dc := abs(tr-fr), abs(tc-fc)
	if !((dr == 1 && dc == 0) || (dr == 0 && dc == 1)) {
		return nil, nil, fmt.Errorf("invalid move")
	}

	st.Board[fr][fc] = 0
	st.Board[tr][tc] = color

	captured := checkEat(&st.Board, tr, tc, color)
	for _, p := range captured {
		st.Board[p.Row][p.Col] = 0
	}

	var capProto []*pb.MoveCapturedCell
	for _, p := range captured {
		capProto = append(capProto, &pb.MoveCapturedCell{Row: uint32(p.Row), Col: uint32(p.Col)})
	}

	nextSeat := otherSeat(st.CurrentSeat)
	st.CurrentSeat = nextSeat
	st.TurnDeadline = time.Now().Add(turnTimeout)

	if winnerColor := checkWinner(&st.Board, nextSeat); winnerColor > 0 {
		st.Phase = phaseEnded
		st.WinnerSeat = colorToSeat(winnerColor)
		moveEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Move{Move: &pb.MoveEvent{
			Seat: action.Seat, FromRow: uint32(fr), FromCol: uint32(fc),
			ToRow: uint32(tr), ToCol: uint32(tc), Captured: capProto,
		}}})
		return st, []engine.GameEvent{
			{Type: engine.EventMove, PushRoute: "onMoveResult", Seat: action.Seat, Payload: moveEv},
		}, nil
	}

	moveEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Move{Move: &pb.MoveEvent{
		Seat: action.Seat, FromRow: uint32(fr), FromCol: uint32(fc),
		ToRow: uint32(tr), ToCol: uint32(tc), Captured: capProto, NextSeat: nextSeat,
	}}})
	turnEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Turn{Turn: &pb.TurnEvent{
		CurrentSeat: nextSeat, TimeoutMs: uint32(turnTimeout / time.Millisecond),
	}}})
	return st, []engine.GameEvent{
		{Type: engine.EventMove, PushRoute: "onMoveResult", Seat: action.Seat, Payload: moveEv},
		{Type: engine.EventTurn, PushRoute: "onTurnNotify", Payload: turnEv},
	}, nil
}

func (e *Engine) OnTick(state engine.GameState, now time.Time) (engine.GameState, []engine.GameEvent, error) {
	st, ok := state.(*State)
	if !ok || st.Phase != phasePlaying {
		return state, nil, nil
	}
	if st.TurnDeadline.IsZero() || now.Before(st.TurnDeadline) {
		return state, nil, nil
	}
	// Current player timed out → opponent wins.
	st.Phase = phaseEnded
	st.WinnerSeat = otherSeat(st.CurrentSeat)
	return st, nil, nil
}

func (e *Engine) VisibleState(state engine.GameState, seat uint32) (interface{}, error) {
	st := state.(*State)
	return map[string]interface{}{
		"seat": seat, "board": flattenBoard(st.Board), "current_seat": st.CurrentSeat,
	}, nil
}

func (e *Engine) CheckRoundEnd(state engine.GameState) (engine.RoundEnd, bool) {
	st := state.(*State)
	if st.Phase != phaseEnded {
		return engine.RoundEnd{}, false
	}
	for _, p := range st.Players {
		if p.Seat == st.WinnerSeat {
			return engine.RoundEnd{Valid: true, WinnerID: p.UserID, WinnerSeat: p.Seat}, true
		}
	}
	return engine.RoundEnd{}, false
}

func (e *Engine) CalcSettlement(state engine.GameState, end engine.RoundEnd) (engine.SettlementResult, error) {
	st := state.(*State)
	scores := make([]engine.PlayerScore, len(st.Players))
	for i, p := range st.Players {
		score := int32(-1)
		if p.Seat == end.WinnerSeat {
			score = 1
		}
		scores[i] = engine.PlayerScore{UserID: p.UserID, Seat: p.Seat, RuleScore: score}
	}
	return engine.SettlementResult{Valid: end.Valid, WinnerID: end.WinnerID, Scores: scores}, nil
}

func initBoard(b *[boardSize][boardSize]int) {
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			b[r][c] = 0
		}
	}
	for c := 0; c < boardSize; c++ {
		b[0][c] = 1
		b[3][c] = 2
	}
	b[1][0] = 1
	b[1][3] = 1
	b[2][0] = 2
	b[2][3] = 2
}

func flattenBoard(b [boardSize][boardSize]int) []uint32 {
	out := make([]uint32, 0, boardSize*boardSize)
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			out = append(out, uint32(b[r][c]))
		}
	}
	return out
}

func seatColor(seat uint32) int {
	return int(seat) + 1
}

func colorToSeat(color int) uint32 {
	return uint32(color - 1)
}

func otherSeat(seat uint32) uint32 {
	if seat == 0 {
		return 1
	}
	return 0
}

func inBounds(r, c int) bool {
	return r >= 0 && r < boardSize && c >= 0 && c < boardSize
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

type cell struct{ Row, Col int }

func checkEat(b *[boardSize][boardSize]int, row, col, color int) []cell {
	enemy := 3 - color
	var eaten []cell

	for start := max(0, col-2); start <= min(boardSize-3, col); start++ {
		cells := []cell{{row, start}, {row, start + 1}, {row, start + 2}}
		if !containsCell(cells, row, col) {
			continue
		}
		vals := []int{b[cells[0].Row][cells[0].Col], b[cells[1].Row][cells[1].Col], b[cells[2].Row][cells[2].Col]}
		colorIdx := indicesWhere(vals, color)
		if len(colorIdx) != 2 || abs(colorIdx[0]-colorIdx[1]) != 1 {
			continue
		}
		enemyIdx := -1
		if colorIdx[0] == 0 && colorIdx[1] == 1 && vals[2] == enemy {
			enemyIdx = 2
		} else if colorIdx[0] == 1 && colorIdx[1] == 2 && vals[0] == enemy {
			enemyIdx = 0
		} else {
			continue
		}
		leftEmpty := start-1 < 0 || b[row][start-1] == 0
		rightEmpty := start+3 >= boardSize || b[row][start+3] == 0
		if !leftEmpty || !rightEmpty {
			continue
		}
		eatPos := cells[enemyIdx]
		if !containsEaten(eaten, eatPos) {
			eaten = append(eaten, eatPos)
		}
	}

	for start := max(0, row-2); start <= min(boardSize-3, row); start++ {
		cells := []cell{{start, col}, {start + 1, col}, {start + 2, col}}
		if !containsCell(cells, row, col) {
			continue
		}
		vals := []int{b[cells[0].Row][cells[0].Col], b[cells[1].Row][cells[1].Col], b[cells[2].Row][cells[2].Col]}
		colorIdx := indicesWhere(vals, color)
		if len(colorIdx) != 2 || abs(colorIdx[0]-colorIdx[1]) != 1 {
			continue
		}
		enemyIdx := -1
		if colorIdx[0] == 0 && colorIdx[1] == 1 && vals[2] == enemy {
			enemyIdx = 2
		} else if colorIdx[0] == 1 && colorIdx[1] == 2 && vals[0] == enemy {
			enemyIdx = 0
		} else {
			continue
		}
		topEmpty := start-1 < 0 || b[start-1][col] == 0
		bottomEmpty := start+3 >= boardSize || b[start+3][col] == 0
		if !topEmpty || !bottomEmpty {
			continue
		}
		eatPos := cells[enemyIdx]
		if !containsEaten(eaten, eatPos) {
			eaten = append(eaten, eatPos)
		}
	}
	return eaten
}

func hasLegalMoves(b *[boardSize][boardSize]int, player int) bool {
	dirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if b[r][c] != player {
				continue
			}
			for _, d := range dirs {
				nr, nc := r+d[0], c+d[1]
				if inBounds(nr, nc) && b[nr][nc] == 0 {
					return true
				}
			}
		}
	}
	return false
}

// ListLegalMoves returns all orthogonal one-step moves for seat.
func ListLegalMoves(st *State, seat uint32) []engine.Action {
	color := seatColor(seat)
	dirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	var out []engine.Action
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if st.Board[r][c] != color {
				continue
			}
			for _, d := range dirs {
				nr, nc := r+d[0], c+d[1]
				if inBounds(nr, nc) && st.Board[nr][nc] == 0 {
					out = append(out, engine.Action{
						Kind: engine.ActionMove, Seat: seat,
						FromRow: r, FromCol: c, ToRow: nr, ToCol: nc,
					})
				}
			}
		}
	}
	return out
}

// PreviewCaptureCount simulates a move and returns how many pieces would be captured.
func PreviewCaptureCount(st *State, action engine.Action) int {
	color := seatColor(action.Seat)
	var board [boardSize][boardSize]int
	board = st.Board
	board[action.FromRow][action.FromCol] = 0
	board[action.ToRow][action.ToCol] = color
	return len(checkEat(&board, action.ToRow, action.ToCol, color))
}

func checkWinner(b *[boardSize][boardSize]int, nextSeat uint32) int {
	black, white := 0, 0
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if b[r][c] == 1 {
				black++
			} else if b[r][c] == 2 {
				white++
			}
		}
	}
	if black <= 1 {
		return 2
	}
	if white <= 1 {
		return 1
	}
	nextColor := seatColor(nextSeat)
	if !hasLegalMoves(b, nextColor) {
		return 3 - nextColor
	}
	return 0
}

func containsCell(cells []cell, row, col int) bool {
	for _, p := range cells {
		if p.Row == row && p.Col == col {
			return true
		}
	}
	return false
}

func containsEaten(eaten []cell, p cell) bool {
	for _, e := range eaten {
		if e.Row == p.Row && e.Col == p.Col {
			return true
		}
	}
	return false
}

func indicesWhere(vals []int, color int) []int {
	var idx []int
	for i, v := range vals {
		if v == color {
			idx = append(idx, i)
		}
	}
	return idx
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
