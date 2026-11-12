package bot

import (
	"github.com/example/game/internal/game/dawugui"
	"github.com/example/game/internal/game/engine"
)

// PickDawuguiAction returns a simple non-LLM bot move (smallest valid play or pass).
func PickDawuguiAction(st *dawugui.State, seat uint32) (engine.Action, bool) {
	if st == nil || st.CurrentSeat != seat {
		return engine.Action{}, false
	}
	eng := dawugui.New()
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
