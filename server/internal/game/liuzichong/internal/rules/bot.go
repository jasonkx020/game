package rules

import (
	"math/rand"

	"github.com/example/game/internal/game/engine"
)

type botPicker struct{}

// NewBotPicker returns a BotPicker for liuzichong.
func NewBotPicker() engine.BotPicker { return botPicker{} }

func (botPicker) Pick(state engine.GameState, seat uint32) (engine.Action, bool) {
	st, ok := state.(*State)
	if !ok {
		return engine.Action{}, false
	}
	return pickAction(st, seat)
}

func (botPicker) ActionRoute(engine.Action) string { return "game.liuzichong.move" }

func (botPicker) SettlementRoute() string { return "game.liuzichong.settlement" }

// pickAction prefers capturing moves; ties broken randomly.
func pickAction(st *State, seat uint32) (engine.Action, bool) {
	moves := ListLegalMoves(st, seat)
	if len(moves) == 0 {
		return engine.Action{}, false
	}
	bestCap := -1
	var best []engine.Action
	for _, m := range moves {
		cap := PreviewCaptureCount(st, m)
		if cap > bestCap {
			bestCap = cap
			best = []engine.Action{m}
		} else if cap == bestCap {
			best = append(best, m)
		}
	}
	return best[rand.Intn(len(best))], true
}
