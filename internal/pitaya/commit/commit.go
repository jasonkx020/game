package commit

import (
	"context"
	"log/slog"
	"time"

	pitaya "github.com/topfreegames/pitaya/v2"
	"google.golang.org/protobuf/proto"

	"github.com/example/game/internal/audit"
	"github.com/example/game/internal/game/engine"
	pb "github.com/example/game/internal/gen/pitaya/pitaya"
	"github.com/example/game/internal/platform/actionlog"
	"github.com/example/game/internal/pitaya/runtime"
)

type Committer struct {
	Log   *actionlog.Repo
	Audit *audit.Generator
}

// CommitEvents locks the room then commits. Callers that already hold room.Lock
// must use CommitEventsLocked instead (sync.Mutex is not reentrant).
func (c *Committer) CommitEvents(ctx context.Context, room *runtime.RoomRuntime, events []engine.GameEvent, c2sRoute string) error {
	room.Lock()
	defer room.Unlock()
	return c.CommitEventsLocked(ctx, room, events, c2sRoute)
}

// CommitEventsLocked commits events while the caller already holds room.Lock.
func (c *Committer) CommitEventsLocked(ctx context.Context, room *runtime.RoomRuntime, events []engine.GameEvent, c2sRoute string) error {
	for _, ev := range events {
		room.ActionSeq++
		sn := c.Audit.Next()
		actor := int64(ev.ActorUID)
		var actorPtr *int64
		if actor > 0 {
			actorPtr = &actor
		}
		seat := int16(ev.Seat)
		var seatPtr *int16
		if ev.Seat > 0 || ev.Type == engine.EventPlay || ev.Type == engine.EventPass || ev.Type == engine.EventMove {
			seatPtr = &seat
		}
		if err := c.Log.Insert(ctx, actionlog.Entry{
			RoomID: room.RoomID, RoundID: room.RoundID, ActionSeq: room.ActionSeq,
			AuditSN: sn, EventType: string(ev.Type), ActorUserID: actorPtr, Seat: seatPtr,
			Payload: ev.Payload, PushRoute: ev.PushRoute, C2SRoute: c2sRoute,
		}); err != nil {
			return err
		}
		if err := c.broadcast(ctx, room, ev, sn); err != nil {
			slog.Error("push failed", "audit_sn", sn, "err", err)
		}
	}
	return nil
}

func (c *Committer) broadcast(ctx context.Context, room *runtime.RoomRuntime, ev engine.GameEvent, auditSN uint64) error {
	msg, err := EventToPush(room, ev, auditSN)
	if err != nil {
		return err
	}
	return pitaya.GroupBroadcast(ctx, "game", room.RoomID.String(), ev.PushRoute, msg)
}

func EventToPush(room *runtime.RoomRuntime, ev engine.GameEvent, auditSN uint64) (proto.Message, error) {
	meta := &pb.EventMeta{
		AuditSn: auditSN, ActionSeq: uint32(room.ActionSeq),
		RoundId: room.RoundID.String(), RoundNo: uint32(room.RoundNo),
		RoomId: room.RoomID.String(), ServerTs: time.Now().UnixMilli(),
	}
	header := &pb.PushHeader{Meta: meta, GameId: room.GameID}

	var ge pb.GameEvent
	if err := proto.Unmarshal(ev.Payload, &ge); err != nil {
		return nil, err
	}

	switch ev.PushRoute {
	case "onBoardInit":
		if b := ge.GetBoardInit(); b != nil {
			return &pb.BoardInitPush{Header: header, Cells: b.Cells, FirstSeat: b.FirstSeat}, nil
		}
	case "onMoveResult":
		if m := ge.GetMove(); m != nil {
			captured := make([]*pb.CapturedCell, len(m.Captured))
			for i, c := range m.Captured {
				captured[i] = &pb.CapturedCell{Row: c.Row, Col: c.Col}
			}
			return &pb.MoveResultPush{Header: header, Seat: m.Seat,
				FromRow: m.FromRow, FromCol: m.FromCol, ToRow: m.ToRow, ToCol: m.ToCol,
				Captured: captured, NextSeat: m.NextSeat}, nil
		}
	case "onDeal":
		if d := ge.GetDeal(); d != nil {
			return &pb.DealPush{Header: header, RoundId: room.RoundID.String(), HandCards: d.HandCards,
				FirstSeat: d.FirstSeat, DealerSeat: d.DealerSeat}, nil
		}
	case "onTurnNotify":
		if t := ge.GetTurn(); t != nil {
			return &pb.TurnNotifyPush{Header: header, CurrentSeat: t.CurrentSeat, TimeoutMs: t.TimeoutMs,
				MustPlay: t.MustPlay, LastPlaySeat: t.LastPlaySeat}, nil
		}
	case "onPlayResult":
		if p := ge.GetPlay(); p != nil {
			return &pb.PlayResultPush{Header: header, Seat: p.Seat, Cards: p.Cards, PlayType: p.PlayType, NextSeat: p.NextSeat}, nil
		}
		if p := ge.GetPass(); p != nil {
			return &pb.PlayResultPush{Header: header, Seat: p.Seat, PlayType: 2, NextSeat: p.NextSeat}, nil
		}
	case "onAlert":
		if a := ge.GetAlert(); a != nil {
			return &pb.AlertPush{Header: header, UserId: a.UserId, Seat: a.Seat, HandCount: a.HandCount}, nil
		}
	case "onSettlement":
		if s := ge.GetSettlement(); s != nil {
			scores := make([]*pb.PlayerScore, len(s.Scores))
			for i, sc := range s.Scores {
				scores[i] = &pb.PlayerScore{UserId: sc.UserId, Seat: sc.Seat, RuleScore: sc.RuleScore}
			}
			return &pb.SettlementPush{Header: header, RoundId: room.RoundID.String(), IsValid: s.IsValid,
				WinnerId: s.WinnerId, Scores: scores}, nil
		}
	case "onRoomState":
		if rs := ge.GetRoomState(); rs != nil {
			players := make([]*pb.PlayerSeat, len(rs.Players))
			for i, p := range rs.Players {
				players[i] = &pb.PlayerSeat{UserId: p.UserId, Seat: p.Seat, Nickname: p.Nickname, Ready: p.Ready, Online: p.Online}
			}
			return &pb.RoomStatePush{Header: header, Phase: rs.Phase, RoomMode: rs.RoomMode, Players: players, RoundNo: rs.RoundNo}, nil
		}
	}
	return &pb.ErrorPush{Header: header, Code: 500, Message: "unknown event mapping"}, nil
}
