package bot

import (
	"context"

	"github.com/example/game/internal/game"
	"github.com/example/game/internal/game/engine"
	"github.com/example/game/internal/pitaya/runtime"
	"github.com/example/game/internal/pitaya/settleapply"
)

// RunPending executes pending bot turns for room.GameID until a human is on
// clock or the round ends. Uses only engine interfaces — no concrete game imports.
// Caller must hold room.Lock.
func RunPending(ctx context.Context, room *runtime.RoomRuntime, deps settleapply.Deps) error {
	if room.EngineState == nil || room.GameID == "" {
		return nil
	}
	eng, err := game.Get(room.GameID)
	if err != nil {
		return nil
	}
	picker, err := game.Bot(room.GameID)
	if err != nil {
		// No BotPicker — trusteeship/bots cannot auto-play; do not error.
		return nil
	}

	maxSteps := 64
	for step := 0; step < maxSteps; step++ {
		turn, ok := room.EngineState.(engine.TurnAware)
		if !ok || !turn.IsPlaying() {
			return nil
		}
		seat := turn.TurnSeat()
		uid, seatInfo, ok := SeatForIndex(room, seat)
		if !ok {
			return nil
		}
		if !IsBot(uid) && (seatInfo == nil || !seatInfo.Trusteeship) {
			return nil
		}
		action, ok := picker.Pick(room.EngineState, seat)
		if !ok {
			return nil
		}
		newState, events, err := eng.ApplyAction(room.EngineState, action)
		if err != nil {
			return nil
		}
		room.EngineState = newState
		if err := deps.Committer.CommitEventsLocked(ctx, room, events, picker.ActionRoute(action)); err != nil {
			return err
		}
		if end, ok := eng.CheckRoundEnd(newState); ok {
			settle, _ := eng.CalcSettlement(newState, end)
			return settleapply.FinishRound(ctx, deps, room, settle, picker.SettlementRoute())
		}
	}
	return nil
}
