package bot

import (
	"context"
	"encoding/json"

	"github.com/example/game/internal/game"
	"github.com/example/game/internal/game/engine"
	pb "github.com/example/game/internal/gen/pitaya/pitaya"
	"github.com/example/game/internal/platform/actionlog"
	"github.com/example/game/internal/pitaya/commit"
	"github.com/example/game/internal/pitaya/runtime"
	"github.com/example/game/pkg/audit"
	"google.golang.org/protobuf/proto"
)

// RunPending executes pending bot turns for room.GameID until a human is on
// clock or the round ends. Uses only engine interfaces — no concrete game imports.
// Caller must hold room.Lock.
func RunPending(
	ctx context.Context,
	room *runtime.RoomRuntime,
	committer *commit.Committer,
	gen *audit.Generator,
	log *actionlog.Repo,
) error {
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
		if err := committer.CommitEventsLocked(ctx, room, events, picker.ActionRoute(action)); err != nil {
			return err
		}
		if end, ok := eng.CheckRoundEnd(newState); ok {
			settle, _ := eng.CalcSettlement(newState, end)
			scoreProto := make([]*pb.EventPlayerScore, len(settle.Scores))
			for i, sc := range settle.Scores {
				scoreProto[i] = &pb.EventPlayerScore{
					UserId: sc.UserID, Seat: sc.Seat, RuleScore: sc.RuleScore,
					Rank: sc.Rank, TeamId: sc.TeamID,
				}
			}
			settleEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Settlement{Settlement: &pb.SettlementEvent{
				IsValid: settle.Valid, WinnerId: settle.WinnerID, Scores: scoreProto, Draw: settle.Draw,
			}}})
			_ = committer.CommitEventsLocked(ctx, room, []engine.GameEvent{
				{Type: engine.EventSettlement, PushRoute: "onSettlement", Payload: settleEv},
			}, picker.SettlementRoute())
			payload, _ := json.Marshal(settle)
			sn := gen.Next()
			_ = log.InsertSettlement(ctx, room.RoomID, room.RoundID, room.GameID, sn, payload)
			_ = log.EndRound(ctx, room.RoundID, "ended", []int64{int64(settle.WinnerID)}, sn)
			for _, si := range room.Seats {
				si.Ready = false
				si.Trusteeship = false
			}
			return nil
		}
	}
	return nil
}
