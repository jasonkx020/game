package rules

import (
	"github.com/example/game/internal/game/engine"
)

type botPicker struct{}

// NewBotPicker returns a BotPicker for dawugui.
func NewBotPicker() engine.BotPicker { return botPicker{} }

func (botPicker) Pick(state engine.GameState, seat uint32) (engine.Action, bool) {
	st, ok := state.(*State)
	if !ok {
		return engine.Action{}, false
	}
	return pickAction(st, seat)
}

func (botPicker) ActionRoute(action engine.Action) string {
	if action.Kind == engine.ActionPass {
		return "game.dawugui.pass"
	}
	return "game.dawugui.playcards"
}

func (botPicker) SettlementRoute() string { return "game.dawugui.settlement" }

// pickAction returns a simple non-LLM bot move (smallest valid play or pass).
func pickAction(st *State, seat uint32) (engine.Action, bool) {
	if st == nil || st.CurrentSeat != seat {
		return engine.Action{}, false
	}
	eng := New()
	hand := append([]uint32(nil), st.Hands[seat]...)
	if len(hand) == 0 {
		return engine.Action{}, false
	}

	canPass := !(st.MustPlay && st.PlayCount == 0) && len(st.LastPlayCards) > 0
	if canPass {
		if _, _, err := eng.ApplyAction(st.Clone(), engine.Action{Kind: engine.ActionPass, Seat: seat}); err == nil {
			return engine.Action{Kind: engine.ActionPass, Seat: seat}, true
		}
	}

	for _, card := range hand {
		action := engine.Action{Kind: engine.ActionPlay, Seat: seat, Cards: []uint32{card}}
		if _, _, err := eng.ApplyAction(st.Clone(), action); err == nil {
			return action, true
		}
	}
	if canPass {
		return engine.Action{Kind: engine.ActionPass, Seat: seat}, true
	}
	if len(hand) > 0 {
		return engine.Action{Kind: engine.ActionPlay, Seat: seat, Cards: []uint32{hand[0]}}, true
	}
	return engine.Action{}, false
}
