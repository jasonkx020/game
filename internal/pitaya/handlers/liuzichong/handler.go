package liuzichong

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	pitaya "github.com/topfreegames/pitaya/v2"
	"github.com/topfreegames/pitaya/v2/component"
	"google.golang.org/protobuf/proto"

	"github.com/example/game/internal/audit"
	"github.com/example/game/internal/game"
	"github.com/example/game/internal/game/engine"
	pb "github.com/example/game/internal/gen/pitaya/pitaya"
	"github.com/example/game/internal/platform/actionlog"
	"github.com/example/game/internal/pitaya/commit"
	"github.com/example/game/internal/pitaya/runtime"
)

type Handler struct {
	component.Base
	store     *runtime.Store
	committer *commit.Committer
	audit     *audit.Generator
	actionLog *actionlog.Repo
}

func New(store *runtime.Store, committer *commit.Committer, gen *audit.Generator, log *actionlog.Repo) *Handler {
	return &Handler{store: store, committer: committer, audit: gen, actionLog: log}
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
	if err := h.committer.CommitEvents(ctx, room, events, "game.liuzichong.move"); err != nil {
		return nil, err
	}

	if end, ok := eng.CheckRoundEnd(newState); ok {
		settle, _ := eng.CalcSettlement(newState, end)
		scoreProto := make([]*pb.EventPlayerScore, len(settle.Scores))
		for i, sc := range settle.Scores {
			scoreProto[i] = &pb.EventPlayerScore{UserId: sc.UserID, Seat: sc.Seat, RuleScore: sc.RuleScore}
		}
		settleEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Settlement{Settlement: &pb.SettlementEvent{
			IsValid: settle.Valid, WinnerId: settle.WinnerID, Scores: scoreProto,
		}}})
		_ = h.committer.CommitEvents(ctx, room, []engine.GameEvent{
			{Type: engine.EventSettlement, PushRoute: "onSettlement", Payload: settleEv},
		}, "game.liuzichong.settlement")
		payload, _ := json.Marshal(settle)
		sn := h.audit.Next()
		_ = h.actionLog.InsertSettlement(ctx, roomID, room.RoundID, room.GameID, sn, payload)
		_ = h.actionLog.EndRound(ctx, room.RoundID, "ended", []int64{int64(settle.WinnerID)}, sn)
		for _, s := range room.Seats {
			s.Ready = false
		}
	}

	meta := &pb.EventMeta{ActionSeq: uint32(room.ActionSeq), AuditSn: h.audit.Next()}
	if len(events) > 0 {
		meta.ActionSeq = uint32(room.ActionSeq - len(events) + 1)
	}
	return &pb.MoveRsp{Meta: meta}, nil
}

func parseUID(uid string) uint64 {
	var v uint64
	for i := 0; i < len(uid); i++ {
		if uid[i] >= '0' && uid[i] <= '9' {
			v = v*10 + uint64(uid[i]-'0')
		}
	}
	return v
}
