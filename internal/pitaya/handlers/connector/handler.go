package connector

import (
	"context"
	"strconv"

	pitaya "github.com/topfreegames/pitaya/v2"
	"github.com/topfreegames/pitaya/v2/component"

	"github.com/example/game/internal/config"
	pb "github.com/example/game/internal/gen/pitaya/pitaya"
	"github.com/example/game/internal/platform/user"
)

type Handler struct {
	component.Base
	cfg *config.Config
}

func New(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}

func (h *Handler) Entry(ctx context.Context, req *pb.EntryReq) (*pb.EntryRsp, error) {
	return &pb.EntryRsp{ServerVersion: "1.0.0", HeartbeatIntervalSec: 15}, nil
}

func (h *Handler) Bind(ctx context.Context, req *pb.BindReq) (*pb.BindRsp, error) {
	typ, err := user.ParsePrincipalType(req.AccessToken, []byte(h.cfg.JWTSecret))
	if err != nil || typ != user.PrincipalTypePlayer {
		return nil, pitaya.Error(err, "AUTH")
	}
	uid, err := user.ParseUserID(req.AccessToken, []byte(h.cfg.JWTSecret))
	if err != nil {
		return nil, pitaya.Error(err, "AUTH")
	}
	s := pitaya.GetSessionFromCtx(ctx)
	if err := s.Bind(ctx, strconv.FormatInt(uid, 10)); err != nil {
		return nil, err
	}
	return &pb.BindRsp{UserId: uint64(uid), Nickname: "player" + strconv.FormatInt(uid, 10)}, nil
}
