package room

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	pitaya "github.com/topfreegames/pitaya/v2"
	"github.com/topfreegames/pitaya/v2/component"
	"google.golang.org/protobuf/proto"

	"github.com/example/game/internal/audit"
	"github.com/example/game/internal/game"
	"github.com/example/game/internal/game/engine"
	pb "github.com/example/game/internal/gen/pitaya/pitaya"
	"github.com/example/game/internal/platform/actionlog"
	"github.com/example/game/internal/pitaya/commit"
	"github.com/example/game/internal/pitaya/roommeta"
	"github.com/example/game/internal/pitaya/runtime"
)

type Handler struct {
	component.Base
	store     *runtime.Store
	committer *commit.Committer
	audit     *audit.Generator
	actionLog *actionlog.Repo
	roomMeta  *roommeta.Repo
}

func New(store *runtime.Store, committer *commit.Committer, gen *audit.Generator, log *actionlog.Repo, db *sqlx.DB) *Handler {
	return &Handler{
		store: store, committer: committer, audit: gen, actionLog: log,
		roomMeta: roommeta.NewRepo(db),
	}
}

func (h *Handler) Join(ctx context.Context, req *pb.JoinReq) (*pb.JoinRsp, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	uid := s.UID()
	if uid == "" {
		return nil, pitaya.Error(nil, "AUTH")
	}
	userID, _ := strconv.ParseUint(uid, 10, 64)
	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, err
	}

	meta, err := h.roomMeta.Get(ctx, roomID)
	if err != nil {
		return nil, pitaya.Error(nil, "ROOM")
	}
	eng, err := game.Get(meta.GameID)
	if err != nil {
		return nil, err
	}
	gameMeta := eng.Meta()

	room := h.store.GetOrCreate(roomID, meta.GameID, meta.PlayerCount)
	room.Lock()
	defer room.Unlock()

	if _, ok := room.Seats[userID]; !ok {
		if room.PlayerCount() >= gameMeta.MaxPlayers {
			return nil, pitaya.Error(nil, "SEAT_FULL")
		}
		seat := uint32(room.PlayerCount())
		room.Seats[userID] = &runtime.Seat{
			UserID: userID, Seat: seat, Nickname: "p" + uid, Online: true,
			Role: runtime.SeatRolePlayer,
		}
		room.RoomSeq++
		_ = h.actionLog.InsertRoomEvent(ctx, roomID, room.RoomSeq, "JOIN", int64(userID), h.audit.Next(), []byte("{}"))
	}
	group := roomID.String()
	_ = pitaya.GroupCreate(ctx, group)
	_ = pitaya.GroupAddMember(ctx, group, uid)

	return &pb.JoinRsp{RoomId: req.RoomId, GameId: room.GameID, RoomMode: pb.RoomMode_ROOM_CARD}, nil
}

func (h *Handler) Ready(ctx context.Context, req *pb.ReadyReq) (*pb.ReadyRsp, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	uid := s.UID()
	userID, _ := strconv.ParseUint(uid, 10, 64)
	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, err
	}
	room, ok := h.store.Get(roomID)
	if !ok {
		return nil, pitaya.Error(nil, "ROOM")
	}
	room.Lock()
	defer room.Unlock()
	if seat, ok := room.Seats[userID]; ok {
		seat.Ready = true
	}

	eng, err := game.Get(room.GameID)
	if err != nil {
		return nil, err
	}
	if !room.AllReady(eng.Meta().MinPlayers) {
		return &pb.ReadyRsp{}, nil
	}

	room.RoundNo++
	cfgBytes, _ := json.Marshal(map[string]interface{}{"base_score": 1})
	round, err := h.actionLog.CreateRound(ctx, roomID, room.RoundNo, room.GameID, cfgBytes)
	if err != nil {
		return nil, err
	}
	room.RoundID = round.RoundID
	room.ActionSeq = 0

	state, events, err := eng.NewState(room.Config, room.Players())
	if err != nil {
		return nil, err
	}
	room.EngineState = state
	if err := h.committer.CommitEvents(ctx, room, events, "game.room.ready"); err != nil {
		return nil, err
	}
	return &pb.ReadyRsp{}, nil
}

func (h *Handler) Leave(ctx context.Context, req *pb.LeaveReq) (*pb.LeaveRsp, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	uid := s.UID()
	userID, _ := strconv.ParseUint(uid, 10, 64)
	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, err
	}
	if room, ok := h.store.Get(roomID); ok {
		room.Lock()
		delete(room.Seats, userID)
		room.RoomSeq++
		_ = h.actionLog.InsertRoomEvent(ctx, roomID, room.RoomSeq, "LEAVE", int64(userID), h.audit.Next(), []byte("{}"))
		room.Unlock()
	}
	_ = pitaya.GroupRemoveMember(ctx, req.RoomId, uid)
	return &pb.LeaveRsp{}, nil
}

func (h *Handler) Sync(ctx context.Context, req *pb.SyncReq) (*pb.SyncRsp, error) {
	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, err
	}
	roundID, err := uuid.Parse(req.RoundId)
	if err != nil {
		return nil, err
	}
	entries, err := h.actionLog.ListSince(ctx, roundID, int(req.SinceActionSeq))
	if err != nil {
		return nil, err
	}
	room, ok := h.store.Get(roomID)
	if !ok {
		return nil, pitaya.Error(nil, "ROOM")
	}
	var pushes []*pb.SyncPushItem
	for _, e := range entries {
		ev := engine.GameEvent{Type: engine.EventType(e.EventType), PushRoute: e.PushRoute, Payload: e.Payload}
		msg, err := commit.EventToPush(room, ev, e.AuditSN)
		if err != nil {
			continue
		}
		data, _ := proto.Marshal(msg)
		pushes = append(pushes, &pb.SyncPushItem{PushRoute: e.PushRoute, PushBody: data})
	}
	latest := req.SinceActionSeq
	if len(entries) > 0 {
		latest = uint32(entries[len(entries)-1].ActionSeq)
	}
	return &pb.SyncRsp{RoundId: req.RoundId, LatestActionSeq: latest, Pushes: pushes}, nil
}
