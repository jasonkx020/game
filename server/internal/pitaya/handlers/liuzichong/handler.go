package liuzichong

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	pitaya "github.com/topfreegames/pitaya/v2"
	"github.com/topfreegames/pitaya/v2/component"

	"github.com/example/game/internal/game"
	"github.com/example/game/internal/game/engine"
	pb "github.com/example/game/internal/gen/pitaya/pitaya"
	"github.com/example/game/internal/platform/actionlog"
	"github.com/example/game/internal/pitaya/bot"
	"github.com/example/game/internal/pitaya/commit"
	"github.com/example/game/internal/pitaya/runtime"
	"github.com/example/game/internal/pitaya/settleapply"
	"github.com/example/game/pkg/audit"
	"github.com/example/game/pkg/platform/risk"
	"github.com/example/game/pkg/platform/settlement"
	"github.com/example/game/pkg/platform/wallet"
)

type Handler struct {
	component.Base
	store  *runtime.Store
	settle settleapply.Deps
}

func New(store *runtime.Store, committer *commit.Committer, gen *audit.Generator, log *actionlog.Repo, w *wallet.Service, settler *settlement.Service, riskSvc *risk.Service) *Handler {
	return &Handler{
		store: store,
		settle: settleapply.Deps{
			Wallet: w, Settlement: settler, Committer: committer, Audit: gen, ActionLog: log,
			Risk: riskSvc, SkipUser: bot.IsBot,
		},
	}
}

func (h *Handler) Move(ctx context.Context, req *pb.MoveReq) (*pb.MoveRsp, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	uid := s.UID()
	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, err
	}
	room, ok := h.store.Get(roomID)
	if !ok || room.EngineState == nil {
		return nil, pitaya.Error(nil, "ROOM")
	}
	room.Lock()
	defer room.Unlock()

	userID := parseUID(uid)
	seat, ok := room.Seats[userID]
	if !ok {
		return nil, pitaya.Error(nil, "SEAT")
	}

	eng, err := game.Get(room.GameID)
	if err != nil {
		return nil, err
	}
	action := engine.Action{
		Kind: engine.ActionMove, Seat: seat.Seat,
		FromRow: int(req.FromRow), FromCol: int(req.FromCol),
		ToRow: int(req.ToRow), ToCol: int(req.ToCol),
	}
	newState, events, err := eng.ApplyAction(room.EngineState, action)
	if err != nil {
		return nil, pitaya.Error(err, "GAME")
	}
	room.EngineState = newState
	if err := h.settle.Committer.CommitEventsLocked(ctx, room, events, "game.liuzichong.move"); err != nil {
		return nil, err
	}

	if end, ok := eng.CheckRoundEnd(newState); ok {
		settle, _ := eng.CalcSettlement(newState, end)
		_ = settleapply.FinishRound(ctx, h.settle, room, settle, "game.liuzichong.settlement")
	} else {
		_ = bot.RunPending(ctx, room, h.settle)
	}

	meta := &pb.EventMeta{ActionSeq: uint32(room.ActionSeq), AuditSn: h.settle.Audit.Next()}
	if len(events) > 0 {
		meta.ActionSeq = uint32(room.ActionSeq - len(events) + 1)
	}
	return &pb.MoveRsp{Meta: meta}, nil
}

func parseUID(uid string) uint64 {
	v, _ := strconv.ParseUint(uid, 10, 64)
	return v
}
