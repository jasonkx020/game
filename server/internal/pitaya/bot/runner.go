package bot

import (
	"context"
	"encoding/json"

	"github.com/example/game/internal/audit"
	"github.com/example/game/internal/game"
	"github.com/example/game/internal/game/dawugui"
	"github.com/example/game/internal/game/engine"
	pb "github.com/example/game/internal/gen/pitaya/pitaya"
	"github.com/example/game/internal/platform/actionlog"
	"github.com/example/game/internal/pitaya/commit"
	"github.com/example/game/internal/pitaya/runtime"
	"google.golang.org/protobuf/proto"
)

// RunDawuguiBots executes pending bot turns until a human is on clock or the round ends.
func RunDawuguiBots(
	ctx context.Context,
	room *runtime.RoomRuntime,
	committer *commit.Committer,
	gen *audit.Generator,
	log *actionlog.Repo,
) error {
	if room.GameID != dawugui.GameID || room.EngineState == nil {
		return nil
	}
	eng, err := game.Get(room.GameID)
	if err != nil {
		return err
	}
	for step := 0; step < 16; step++ {
		st, ok := room.EngineState.(*dawugui.State)
		if !ok || st.Phase != 1 { // phasePlaying
			return nil
		}
		uid, ok := UserIDForSeat(room, st.CurrentSeat)
		if !ok || !IsBot(uid) {
			return nil
		}
		action, ok := PickDawuguiAction(st, st.CurrentSeat)
		if !ok {
			return nil
		}
		newState, events, err := eng.ApplyAction(room.EngineState, action)
		if err != nil {
			return nil
		}
		room.EngineState = newState
		route := "game.dawugui.playcards"
		if action.Kind == engine.ActionPass {
			route = "game.dawugui.pass"
		}
		if err := committer.CommitEventsLocked(ctx, room, events, route); err != nil {
			return err
		}
		if end, ok := eng.CheckRoundEnd(newState); ok {
			settle, _ := eng.CalcSettlement(newState, end)
			settleEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Settlement{Settlement: &pb.SettlementEvent{
				IsValid: settle.Valid, WinnerId: settle.WinnerID,
			}}})
			_ = committer.CommitEventsLocked(ctx, room, []engine.GameEvent{
				{Type: engine.EventSettlement, PushRoute: "onSettlement", Payload: settleEv},
			}, "game.dawugui.settlement")
			payload, _ := json.Marshal(settle)
			sn := gen.Next()
			_ = log.InsertSettlement(ctx, room.RoomID, room.RoundID, room.GameID, sn, payload)
			_ = log.EndRound(ctx, room.RoundID, "ended", []int64{int64(settle.WinnerID)}, sn)
			for _, seat := range room.Seats {
				seat.Ready = false
			}
			return nil
		}
	}
	return nil
}
