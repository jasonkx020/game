package room

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

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
	"github.com/example/game/internal/pitaya/bot"
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

	seatInfo, ok := room.Seats[userID]
	if !ok {
		if room.PlayerCount() >= gameMeta.MaxPlayers {
			return nil, pitaya.Error(nil, "SEAT_FULL")
		}
		seat := uint32(room.PlayerCount())
		seatInfo = &runtime.Seat{
			UserID: userID, Seat: seat, Nickname: "p" + uid, Online: true,
			Role: runtime.SeatRolePlayer,
		}
		room.Seats[userID] = seatInfo
		room.RoomSeq++
		_ = h.actionLog.InsertRoomEvent(ctx, roomID, room.RoomSeq, "JOIN", int64(userID), h.audit.Next(), []byte("{}"))
	}
	if meta.FillBots() {
		bot.FillSeats(room, meta.PlayerCount)
	}
	group := roomID.String()
	_ = pitaya.GroupCreate(ctx, group)
	_ = pitaya.GroupAddMember(ctx, group, uid)

	phase := pb.RoomPhase_IDLE
	if room.EngineState != nil {
		phase = pb.RoomPhase_PLAYING
	}
	players := protoPlayers(room)
	h.pushRoomState(ctx, room, phase, players)

	return &pb.JoinRsp{
		RoomId:   req.RoomId,
		GameId:   room.GameID,
		RoomMode: pb.RoomMode_ROOM_CARD,
		Seat:     seatInfo.Seat,
		Players:  players,
	}, nil
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
	bot.MarkBotsReady(room)

	eng, err := game.Get(room.GameID)
	if err != nil {
		return nil, err
	}
	if !room.AllReady(eng.Meta().MinPlayers) {
		h.pushRoomState(ctx, room, pb.RoomPhase_IDLE, protoPlayers(room))
		return &pb.ReadyRsp{}, nil
	}

	room.RoundNo++
	cfgBytes, _ := json.Marshal(map[string]interface{}{"base_score": 1})
	round, err := h.actionLog.CreateRound(ctx, roomID, room.RoundNo, room.GameID, cfgBytes)
	if err != nil {
		return nil, pitaya.Error(err, "ROUND")
	}
	room.RoundID = round.RoundID
	room.ActionSeq = 0

	state, events, err := eng.NewState(room.Config, room.Players())
	if err != nil {
		return nil, pitaya.Error(err, "GAME")
	}
	room.EngineState = state
	if err := h.committer.CommitEventsLocked(ctx, room, events, "game.room.ready"); err != nil {
		return nil, err
	}
	h.pushRoomState(ctx, room, pb.RoomPhase_PLAYING, protoPlayers(room))
	_ = bot.RunDawuguiBots(ctx, room, h.committer, h.audit, h.actionLog)
	_ = bot.RunLiuzichongBots(ctx, room, h.committer, h.audit, h.actionLog)
	go h.tickRoom(roomID)
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
		h.pushRoomState(ctx, room, pb.RoomPhase_IDLE, protoPlayers(room))
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

func (h *Handler) tickRoom(roomID uuid.UUID) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		room, ok := h.store.Get(roomID)
		if !ok {
			return
		}
		room.Lock()
		if room.EngineState == nil || room.RoundID == uuid.Nil {
			room.Unlock()
			return
		}
		eng, err := game.Get(room.GameID)
		if err != nil {
			room.Unlock()
			return
		}
		if end, ok := eng.CheckRoundEnd(room.EngineState); ok {
			_ = end
			room.Unlock()
			return
		}
		newState, events, err := eng.OnTick(room.EngineState, time.Now())
		if err != nil {
			room.Unlock()
			continue
		}
		if len(events) == 0 {
			if end, ended := eng.CheckRoundEnd(newState); !ended {
				_ = end
				room.Unlock()
				continue
			}
			// Timeout end with no push events — still settle below.
		} else {
			room.EngineState = newState
			_ = h.committer.CommitEventsLocked(context.Background(), room, events, "game.room.tick")
		}
		room.EngineState = newState
		if end, ok := eng.CheckRoundEnd(newState); ok {
			settle, _ := eng.CalcSettlement(newState, end)
			scoreProto := make([]*pb.EventPlayerScore, len(settle.Scores))
			for i, sc := range settle.Scores {
				scoreProto[i] = &pb.EventPlayerScore{UserId: sc.UserID, Seat: sc.Seat, RuleScore: sc.RuleScore}
			}
			settleEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Settlement{Settlement: &pb.SettlementEvent{
				IsValid: settle.Valid, WinnerId: settle.WinnerID, Scores: scoreProto,
			}}})
			_ = h.committer.CommitEventsLocked(context.Background(), room, []engine.GameEvent{
				{Type: engine.EventSettlement, PushRoute: "onSettlement", Payload: settleEv},
			}, "game.room.tick")
			payload, _ := json.Marshal(settle)
			sn := h.audit.Next()
			_ = h.actionLog.InsertSettlement(context.Background(), roomID, room.RoundID, room.GameID, sn, payload)
			_ = h.actionLog.EndRound(context.Background(), room.RoundID, "ended", []int64{int64(settle.WinnerID)}, sn)
			for _, s := range room.Seats {
				s.Ready = false
			}
			room.Unlock()
			return
		}
		_ = bot.RunDawuguiBots(context.Background(), room, h.committer, h.audit, h.actionLog)
		_ = bot.RunLiuzichongBots(context.Background(), room, h.committer, h.audit, h.actionLog)
		room.Unlock()
	}
}

func protoPlayers(room *runtime.RoomRuntime) []*pb.PlayerSeat {
	seats := room.PlayerSeats()
	out := make([]*pb.PlayerSeat, len(seats))
	for i, s := range seats {
		out[i] = &pb.PlayerSeat{
			UserId: s.UserID, Seat: s.Seat, Nickname: s.Nickname,
			Ready: s.Ready, Online: s.Online,
		}
	}
	return out
}

func (h *Handler) pushRoomState(ctx context.Context, room *runtime.RoomRuntime, phase pb.RoomPhase, players []*pb.PlayerSeat) {
	msg := &pb.RoomStatePush{
		Header: &pb.PushHeader{
			Meta: &pb.EventMeta{
				RoomId:   room.RoomID.String(),
				RoundNo:  uint32(room.RoundNo),
				ServerTs: time.Now().UnixMilli(),
			},
			GameId: room.GameID,
		},
		Phase:    phase,
		RoomMode: pb.RoomMode_ROOM_CARD,
		Players:  players,
		RoundNo:  uint32(room.RoundNo),
	}
	_ = pitaya.GroupBroadcast(ctx, "game", room.RoomID.String(), "onRoomState", msg)
}
