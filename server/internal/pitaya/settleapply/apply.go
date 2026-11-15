package settleapply

import (
	"context"
	"encoding/json"

	"google.golang.org/protobuf/proto"

	"github.com/example/game/internal/game/engine"
	pb "github.com/example/game/internal/gen/pitaya/pitaya"
	"github.com/example/game/internal/platform/actionlog"
	"github.com/example/game/internal/pitaya/commit"
	"github.com/example/game/internal/pitaya/runtime"
	"github.com/example/game/pkg/audit"
	"github.com/example/game/pkg/platform/settlement"
	"github.com/example/game/pkg/platform/wallet"
)

// Deps wires monetary settlement into round-end handlers.
type Deps struct {
	Wallet     *wallet.Service
	Settlement *settlement.Service
	Committer  *commit.Committer
	Audit      *audit.Generator
	ActionLog  *actionlog.Repo
	SkipUser   func(userID uint64) bool
}

func protoScores(settle engine.SettlementResult) []*pb.EventPlayerScore {
	out := make([]*pb.EventPlayerScore, len(settle.Scores))
	for i, sc := range settle.Scores {
		out[i] = &pb.EventPlayerScore{
			UserId: sc.UserID, Seat: sc.Seat, RuleScore: sc.RuleScore,
			Rank: sc.Rank, TeamId: sc.TeamID,
		}
	}
	return out
}

func protoCoins(deltas []settlement.CoinDelta) []*pb.EventPlayerCoin {
	out := make([]*pb.EventPlayerCoin, len(deltas))
	for i, d := range deltas {
		out[i] = &pb.EventPlayerCoin{UserId: d.UserID, Delta: d.Delta, AuditSn: d.AuditSN}
	}
	return out
}

// FinishRound commits settlement push, optional score_field wallet apply, and ends the round.
// Caller must hold room.Lock.
func FinishRound(ctx context.Context, deps Deps, room *runtime.RoomRuntime, settle engine.SettlementResult, route string) error {
	scoreProto := protoScores(settle)
	ev := &pb.SettlementEvent{
		IsValid: settle.Valid, WinnerId: settle.WinnerID, Scores: scoreProto, Draw: settle.Draw,
	}

	var monetary *settlement.Result
	if room.RoomMode == "score_field" && deps.Settlement != nil && deps.Wallet != nil && settle.Valid {
		tx, err := deps.Wallet.BeginTx(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback() //nolint:errcheck
		monetary, err = deps.Settlement.ApplyScoreField(
			ctx, tx, room.GameID, room.RoomID, room.RoundID,
			room.Config.BaseScore, settle, deps.Audit.Next, deps.SkipUser,
		)
		if err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		ev.RakeAmount = monetary.RakeAmount
		ev.CoinDelta = protoCoins(monetary.CoinDeltas)
	}

	settleEv, err := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Settlement{Settlement: ev}})
	if err != nil {
		return err
	}
	if err := deps.Committer.CommitEventsLocked(ctx, room, []engine.GameEvent{
		{Type: engine.EventSettlement, PushRoute: "onSettlement", Payload: settleEv},
	}, route); err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"settle": settle,
		"rake":   ev.RakeAmount,
		"coins":  monetary,
	})
	sn := deps.Audit.Next()
	_ = deps.ActionLog.InsertSettlement(ctx, room.RoomID, room.RoundID, room.GameID, sn, payload)
	winners := []int64{}
	if settle.WinnerID != 0 {
		winners = append(winners, int64(settle.WinnerID))
	}
	_ = deps.ActionLog.EndRound(ctx, room.RoundID, "ended", winners, sn)
	for _, s := range room.Seats {
		s.Ready = false
		s.Trusteeship = false
	}
	return nil
}
