package connector

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	pitaya "github.com/topfreegames/pitaya/v2"
	"github.com/topfreegames/pitaya/v2/component"
	"github.com/topfreegames/pitaya/v2/session"

	pb "github.com/example/game/internal/gen/pitaya/pitaya"
	"github.com/example/game/pkg/config"
	"github.com/example/game/pkg/platform/player"
	"github.com/example/game/pkg/platform/risk"
	"github.com/example/game/pkg/platform/user"
)

type Handler struct {
	component.Base
	cfg     *config.Config
	players *player.Service
	risk    *risk.Service
}

func New(cfg *config.Config, players *player.Service, riskSvc *risk.Service) *Handler {
	return &Handler{cfg: cfg, players: players, risk: riskSvc}
}

func (h *Handler) Entry(ctx context.Context, req *pb.EntryReq) (*pb.EntryRsp, error) {
	return &pb.EntryRsp{ServerVersion: "1.0.0", HeartbeatIntervalSec: 15}, nil
}

func (h *Handler) Bind(ctx context.Context, req *pb.BindReq) (*pb.BindRsp, error) {
	claims, err := user.ParseAccessClaims(req.AccessToken, h.cfg.JWTPublicKey)
	if err != nil || claims.PrincipalType != user.PrincipalTypePlayer {
		return nil, pitaya.Error(err, "AUTH")
	}
	p, err := h.players.EnsureFromAuth(ctx, claims.AccountID, claims.Phone, claims.Nickname, claims.IsMinor)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "frozen") {
			return nil, pitaya.Error(err, risk.CodeFrozen)
		}
		return nil, pitaya.Error(err, "AUTH")
	}
	s := pitaya.GetSessionFromCtx(ctx)
	ip := sessionIP(s)
	deviceID := strings.TrimSpace(req.DeviceId)
	platform := ""
	if h.risk != nil {
		if err := h.risk.CheckBind(ctx, p.ID, deviceID, ip, platform, time.Now()); err != nil {
			return nil, pitaya.Error(err, riskCode(err))
		}
	}
	if err := s.Bind(ctx, strconv.FormatInt(p.ID, 10)); err != nil {
		return nil, err
	}
	_ = s.SetData(map[string]interface{}{
		"device_id": deviceID,
		"client_ip": ip,
	})
	nick := p.Nickname
	if nick == "" {
		nick = "player" + strconv.FormatInt(p.ID, 10)
	}
	return &pb.BindRsp{UserId: uint64(p.ID), Nickname: nick}, nil
}

func sessionIP(s session.Session) string {
	if s == nil {
		return ""
	}
	addr := s.RemoteAddr()
	if addr == nil {
		return ""
	}
	if ta, ok := addr.(*net.TCPAddr); ok {
		return ta.IP.String()
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}

func riskCode(err error) string {
	switch {
	case errors.Is(err, risk.ErrDeviceLimit):
		return risk.CodeDevice
	case errors.Is(err, risk.ErrCurfew):
		return risk.CodeCurfew
	case errors.Is(err, risk.ErrPlaytime):
		return risk.CodePlaytime
	case errors.Is(err, risk.ErrFrozen):
		return risk.CodeFrozen
	default:
		return "RISK"
	}
}
