package dawugui

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	pitaya "github.com/topfreegames/pitaya/v2"
	"github.com/topfreegames/pitaya/v2/component"
	"google.golang.org/protobuf/proto"

	"github.com/example/game/internal/audit"
	"github.com/example/game/internal/game/engine"
	"github.com/example/game/internal/game"
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

func (h *Handler) Playcards(ctx context.Context, req *pb.PlayCardsReq) (*pb.PlayCardsRsp, error) {
	return h.apply(ctx, req.RoomId, engine.Action{Kind: engine.ActionPlay, Cards: req.Cards})
}

func (h *Handler) Pass(ctx context.Context, req *pb.PassReq) (*pb.PassRsp, error) {
	_, err := h.applyPass(ctx, req.RoomId)
	return &pb.PassRsp{}, err
}

func (h *Handler) apply(ctx context.Context, roomIDStr string, action engine.Action) (*pb.PlayCardsRsp, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	uid := s.UID()
	roomID, err := uuid.Parse(roomIDStr)
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
	action.Seat = seat.Seat

	eng, err := game.Get(room.GameID)
	if err != nil {
		return nil, err
	}
	newState, events, err := eng.ApplyAction(room.EngineState, action)
	if err != nil {
		return nil, pitaya.Error(err, "GAME")
	}
	room.EngineState = newState
	if err := h.committer.CommitEvents(ctx, room, events, "game.dawugui.playcards"); err != nil {
		return nil, err
	}

	if end, ok := eng.CheckRoundEnd(newState); ok {
		settle, _ := eng.CalcSettlement(newState, end)
		settleEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Settlement{Settlement: &pb.SettlementEvent{
			IsValid: settle.Valid, WinnerId: settle.WinnerID,
		}}})
		_ = h.committer.CommitEvents(ctx, room, []engine.GameEvent{
			{Type: engine.EventSettlement, PushRoute: "onSettlement", Payload: settleEv},
		}, "game.dawugui.settlement")
		payload, _ := json.Marshal(settle)
		sn := h.audit.Next()
		_ = h.actionLog.InsertSettlement(ctx, roomID, room.RoundID, room.GameID, sn, payload)
		_ = h.actionLog.EndRound(ctx, room.RoundID, "ended", []int64{int64(settle.WinnerID)}, sn)
		for _, seat := range room.Seats {
			seat.Ready = false
		}
	}

	meta := &pb.EventMeta{ActionSeq: uint32(room.ActionSeq), AuditSn: h.audit.Next()}
	if len(events) > 0 {
		meta.ActionSeq = uint32(room.ActionSeq - len(events) + 1)
	}
	return &pb.PlayCardsRsp{Meta: meta}, nil
}

func (h *Handler) applyPass(ctx context.Context, roomIDStr string) (*pb.PlayCardsRsp, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	uid := s.UID()
	userID := parseUID(uid)
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		return nil, err
	}
	room, ok := h.store.Get(roomID)
	if !ok || room.EngineState == nil {
		return nil, pitaya.Error(nil, "ROOM")
	}
	room.Lock()
	defer room.Unlock()
	seat, ok := room.Seats[userID]
	if !ok {
		return nil, pitaya.Error(nil, "SEAT")
	}
	eng, _ := game.Get(room.GameID)
	newState, events, err := eng.ApplyAction(room.EngineState, engine.Action{Kind: engine.ActionPass, Seat: seat.Seat})
	if err != nil {
		return nil, pitaya.Error(err, "GAME")
	}
	room.EngineState = newState
	if err := h.committer.CommitEvents(ctx, room, events, "game.dawugui.pass"); err != nil {
		return nil, err
	}
	return &pb.PlayCardsRsp{}, nil
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
