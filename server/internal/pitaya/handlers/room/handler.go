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

	"github.com/example/game/internal/game"
	"github.com/example/game/internal/game/engine"
	pb "github.com/example/game/internal/gen/pitaya/pitaya"
	"github.com/example/game/internal/platform/actionlog"
	"github.com/example/game/internal/pitaya/bot"
	"github.com/example/game/internal/pitaya/commit"
	"github.com/example/game/internal/pitaya/roommeta"
	"github.com/example/game/internal/pitaya/runtime"
	"github.com/example/game/internal/pitaya/settleapply"
	"github.com/example/game/pkg/audit"
	"github.com/example/game/pkg/platform/settlement"
	"github.com/example/game/pkg/platform/wallet"
)

type Handler struct {
	component.Base
	store     *runtime.Store
	committer *commit.Committer
	audit     *audit.Generator
	actionLog *actionlog.Repo
	roomMeta  *roommeta.Repo
	settle    settleapply.Deps
}

func New(store *runtime.Store, committer *commit.Committer, gen *audit.Generator, log *actionlog.Repo, db *sqlx.DB, w *wallet.Service, settler *settlement.Service) *Handler {
	return &Handler{
		store: store, committer: committer, audit: gen, actionLog: log,
		roomMeta: roommeta.NewRepo(db),
		settle: settleapply.Deps{
			Wallet: w, Settlement: settler, Committer: committer, Audit: gen, ActionLog: log,
			SkipUser: bot.IsBot,
		},
	}
}

func roomModeProto(mode string) pb.RoomMode {
	if mode == "score_field" {
		return pb.RoomMode_SCORE_FIELD
	}
	return pb.RoomMode_ROOM_CARD
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

	room := h.store.GetOrCreateWith(roomID, runtime.CreateOpts{
		GameID: meta.GameID, MaxPlayers: meta.PlayerCount,
		BaseScore: meta.BaseScore(), RoomMode: meta.NormalizedRoomMode(),
	})
	room.Lock()
	defer room.Unlock()

	rejoined := false
	seatInfo, ok := room.Seats[userID]
	if ok {
		rejoined = true
		seatInfo.Online = true
		seatInfo.Trusteeship = false // reconnect clears trusteeship
	} else if req.GetAsSpectator() {
		seatInfo = &runtime.Seat{
			UserID: userID, Seat: room.NextSpectatorSeat(), Nickname: "p" + uid,
			Online: true, Role: runtime.SeatRoleSpectator,
		}
		room.Seats[userID] = seatInfo
		room.RoomSeq++
		_ = h.actionLog.InsertRoomEvent(ctx, roomID, room.RoomSeq, "JOIN_SPECTATOR", int64(userID), h.audit.Next(), []byte("{}"))
	} else {
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
	if meta.FillBots() && !req.GetAsSpectator() {
		bot.FillSeats(room, meta.PlayerCount)
	}
	group := roomID.String()
	_ = pitaya.GroupCreate(ctx, group)
	_ = pitaya.GroupAddMember(ctx, group, uid)

	roomIDCopy, userIDCopy := roomID, userID
	s.OnClose(func() {
		if r, ok := h.store.Get(roomIDCopy); ok {
			r.Lock()
			changed := r.MarkOffline(userIDCopy)
			if changed {
				phase := pb.RoomPhase_IDLE
				if r.EngineState != nil {
					phase = pb.RoomPhase_PLAYING
				}
				h.pushRoomState(context.Background(), r, phase, protoSeats(r))
			}
			r.Unlock()
		}
	})

	phase := pb.RoomPhase_IDLE
	if room.EngineState != nil {
		phase = pb.RoomPhase_PLAYING
	}
	players := protoSeats(room)
	h.pushRoomState(ctx, room, phase, players)

	rsp := &pb.JoinRsp{
		RoomId:   req.RoomId,
		GameId:   room.GameID,
		RoomMode: roomModeProto(room.RoomMode),
		Seat:     seatInfo.Seat,
		Players:  players,
		Rejoined: rejoined,
		Role:     uint32(seatInfo.Role),
	}
	if phased, ok := room.EngineState.(engine.Phased); ok {
		rsp.GamePhase = phased.Phase()
	}
	if room.EngineState != nil {
		if snapper, ok := eng.(engine.Snapshotter); ok {
			events, err := snapper.Snapshot(room.EngineState, seatInfo.Seat)
			if err == nil {
				rsp.SnapshotPushes = h.eventsToSyncItems(room, events)
			}
		}
	}

	return rsp, nil
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
		if seat.Role == runtime.SeatRoleSpectator {
			return &pb.ReadyRsp{}, nil
		}
		seat.Ready = true
	}
	bot.MarkBotsReady(room)

	eng, err := game.Get(room.GameID)
	if err != nil {
		return nil, err
	}
	if !room.AllReady(eng.Meta().MinPlayers) {
		h.pushRoomState(ctx, room, pb.RoomPhase_IDLE, protoSeats(room))
		return &pb.ReadyRsp{}, nil
	}

	room.RoundNo++
	cfgBytes, _ := json.Marshal(map[string]interface{}{"base_score": room.Config.BaseScore, "seed": room.Config.Seed})
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
	h.pushRoomState(ctx, room, pb.RoomPhase_PLAYING, protoSeats(room))
	_ = bot.RunPending(ctx, room, h.settle)
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
		phase := pb.RoomPhase_IDLE
		if room.EngineState != nil {
			phase = pb.RoomPhase_PLAYING
		}
		h.pushRoomState(ctx, room, phase, protoSeats(room))
		room.Unlock()
	}
	_ = pitaya.GroupRemoveMember(ctx, req.RoomId, uid)
	return &pb.LeaveRsp{}, nil
}

func (h *Handler) Trusteeship(ctx context.Context, req *pb.TrusteeshipReq) (*pb.TrusteeshipRsp, error) {
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
	seat, ok := room.Seats[userID]
	if !ok || seat.Role != runtime.SeatRolePlayer {
		return nil, pitaya.Error(nil, "SEAT")
	}
	seat.Trusteeship = req.GetEnabled()
	phase := pb.RoomPhase_IDLE
	if room.EngineState != nil {
		phase = pb.RoomPhase_PLAYING
	}
	h.pushRoomState(ctx, room, phase, protoSeats(room))
	if seat.Trusteeship && room.EngineState != nil {
		_ = bot.RunPending(ctx, room, h.settle)
	}
	return &pb.TrusteeshipRsp{Enabled: seat.Trusteeship}, nil
}

func (h *Handler) Sync(ctx context.Context, req *pb.SyncReq) (*pb.SyncRsp, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	uid := s.UID()
	userID, _ := strconv.ParseUint(uid, 10, 64)
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
	room.Lock()
	defer room.Unlock()

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
	rsp := &pb.SyncRsp{RoundId: req.RoundId, LatestActionSeq: latest, Pushes: pushes}
	if phased, ok := room.EngineState.(engine.Phased); ok {
		rsp.GamePhase = phased.Phase()
	}
	if room.EngineState != nil {
		if eng, err := game.Get(room.GameID); err == nil {
			if snapper, ok := eng.(engine.Snapshotter); ok {
				seatNum := uint32(0)
				if seat, ok := room.Seats[userID]; ok {
					seatNum = seat.Seat
				}
				if events, err := snapper.Snapshot(room.EngineState, seatNum); err == nil {
					rsp.SnapshotPushes = h.eventsToSyncItems(room, events)
				}
			}
		}
	}
	return rsp, nil
}

func (h *Handler) eventsToSyncItems(room *runtime.RoomRuntime, events []engine.GameEvent) []*pb.SyncPushItem {
	out := make([]*pb.SyncPushItem, 0, len(events))
	for _, ev := range events {
		msg, err := commit.EventToPush(room, ev, 0)
		if err != nil {
			continue
		}
		data, _ := proto.Marshal(msg)
		out = append(out, &pb.SyncPushItem{PushRoute: ev.PushRoute, PushBody: data})
	}
	return out
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
				_ = bot.RunPending(context.Background(), room, h.settle)
				room.Unlock()
				continue
			}
		} else {
			room.EngineState = newState
			_ = h.committer.CommitEventsLocked(context.Background(), room, events, "game.room.tick")
		}
		room.EngineState = newState
		if end, ok := eng.CheckRoundEnd(newState); ok {
			settle, _ := eng.CalcSettlement(newState, end)
			_ = settleapply.FinishRound(context.Background(), h.settle, room, settle, "game.room.tick")
			room.Unlock()
			return
		}
		_ = bot.RunPending(context.Background(), room, h.settle)
		room.Unlock()
	}
}

func protoSeats(room *runtime.RoomRuntime) []*pb.PlayerSeat {
	seats := room.AllSeats()
	out := make([]*pb.PlayerSeat, len(seats))
	for i, s := range seats {
		out[i] = &pb.PlayerSeat{
			UserId: s.UserID, Seat: s.Seat, Nickname: s.Nickname,
			Ready: s.Ready, Online: s.Online,
			Role: uint32(s.Role), Trusteeship: s.Trusteeship,
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
		Phase:     phase,
		RoomMode:  roomModeProto(room.RoomMode),
		Players:   players,
		RoundNo:   uint32(room.RoundNo),
		BaseScore: room.Config.BaseScore,
	}
	_ = pitaya.GroupBroadcast(ctx, "game", room.RoomID.String(), "onRoomState", msg)
}
