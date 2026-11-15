package connector

import (
	"context"
	"strconv"

	pitaya "github.com/topfreegames/pitaya/v2"
	"github.com/topfreegames/pitaya/v2/component"

	"github.com/example/game/pkg/config"
	pb "github.com/example/game/internal/gen/pitaya/pitaya"
	"github.com/example/game/pkg/platform/player"
	"github.com/example/game/pkg/platform/user"
)

type Handler struct {
	component.Base
	cfg     *config.Config
	players *player.Service
}

func New(cfg *config.Config, players *player.Service) *Handler {
	return &Handler{cfg: cfg, players: players}
}

func (h *Handler) Entry(ctx context.Context, req *pb.EntryReq) (*pb.EntryRsp, error) {
	return &pb.EntryRsp{ServerVersion: "1.0.0", HeartbeatIntervalSec: 15}, nil
}

func (h *Handler) Bind(ctx context.Context, req *pb.BindReq) (*pb.BindRsp, error) {
	claims, err := user.ParseAccessClaims(req.AccessToken, []byte(h.cfg.JWTSecret))
	if err != nil || claims.PrincipalType != user.PrincipalTypePlayer {
		return nil, pitaya.Error(err, "AUTH")
	}
	p, err := h.players.EnsureFromAuth(ctx, claims.AccountID, claims.Phone, claims.Nickname)
	if err != nil {
		return nil, pitaya.Error(err, "AUTH")
	}
	s := pitaya.GetSessionFromCtx(ctx)
	if err := s.Bind(ctx, strconv.FormatInt(p.ID, 10)); err != nil {
		return nil, err
	}
	nick := p.Nickname
	if nick == "" {
		nick = "player" + strconv.FormatInt(p.ID, 10)
	}
	return &pb.BindRsp{UserId: uint64(p.ID), Nickname: nick}, nil
}
