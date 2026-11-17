package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/example/game/pkg/audit"
	"github.com/example/game/pkg/config"
	"github.com/example/game/pkg/platform/catalog"
	"github.com/example/game/pkg/platform/club"
	"github.com/example/game/internal/platform/companion"
	"github.com/example/game/pkg/httpmw"
	"github.com/example/game/internal/platform/lobby"
	"github.com/example/game/pkg/platform/player"
	"github.com/example/game/internal/platform/replay"
	"github.com/example/game/pkg/platform/risk"
	"github.com/example/game/internal/platform/room"
	"github.com/example/game/pkg/platform/user"
	"github.com/example/game/pkg/platform/wallet"
	goredis "github.com/redis/go-redis/v9"
)

type Server struct {
	cfg       *config.Config
	players   *player.Service
	wallet    *wallet.Service
	rooms     *room.Service
	clubs     *club.Service
	catalog   *catalog.Service
	lobby     *lobby.Service
	companion *companion.Service
	replay    *replay.Service
	risk      *risk.Service
	audit     *audit.Generator
	rdb       *goredis.Client
}

func New(
	cfg *config.Config,
	players *player.Service,
	wallet *wallet.Service,
	rooms *room.Service,
	clubs *club.Service,
	catalog *catalog.Service,
	lobbySvc *lobby.Service,
	companionSvc *companion.Service,
	replaySvc *replay.Service,
	riskSvc *risk.Service,
	gen *audit.Generator,
	rdb *goredis.Client,
) *Server {
	return &Server{
		cfg: cfg, players: players, wallet: wallet, rooms: rooms,
		clubs: clubs, catalog: catalog, lobby: lobbySvc, companion: companionSvc,
		replay: replaySvc, risk: riskSvc, audit: gen, rdb: rdb,
	}
}

func (s *Server) Register(r *gin.Engine) {
	r.Use(gin.Recovery(), httpmw.RequestID(), httpmw.CORS(s.cfg.CORSOrigins))

	sig := httpmw.Signature(httpmw.SignatureConfig{
		AppID: s.cfg.AppID, AppSecret: s.cfg.HMACSecret, Skip: false,
	}, s.rdb)

	jwtPlayer := httpmw.RequirePrincipalType(s.cfg.JWTPublicKey, user.PrincipalTypePlayer)

	v1 := r.Group("/v1")
	v1.Use(sig)
	{
		// Auth login/refresh/logout 由独立 auth-service 提供（默认 :8081）
		// Admin 运营 API 由独立 admin-api 提供（默认 :8082）；本进程不依赖 admin-api。

		clubRead := v1.Group("")
		clubRead.Use(httpmw.JWT())
		{
			clubRead.GET("/clubs/:club_id", s.getClub)
			clubRead.GET("/clubs/:club_id/members", s.listClubMembers)
			clubRead.GET("/clubs/:club_id/room-card", s.clubPoolBalance)
		}

		catalogRead := v1.Group("")
		catalogRead.Use(httpmw.JWT())
		{
			catalogRead.GET("/games", s.listGames)
			catalogRead.GET("/games/:game_id/config", s.getGameConfig)
			catalogRead.GET("/companion/personas", s.listCompanionPersonas)
		}

		auth := v1.Group("")
		auth.Use(httpmw.JWT(), jwtPlayer)
		{
			auth.GET("/user/profile", s.profile)
			auth.PUT("/user/profile", s.putProfile)
			auth.GET("/user/settings", s.getUserSettings)
			auth.PUT("/user/settings", s.putUserSettings)
			auth.GET("/users/me/matches", s.listMyMatches)

			auth.GET("/lobby/recommendations", s.lobbyRecommendations)
			auth.POST("/companion/sessions", s.createCompanionSession)
			auth.GET("/companion/sessions/:session_id/messages", s.listCompanionMessages)
			auth.POST("/companion/sessions/:session_id/chat", s.companionChat)

			auth.GET("/wallet/room-card", s.roomCardBalance)
			auth.GET("/wallet/game-coin/:game_id", s.gameCoinBalance)
			auth.POST("/wallet/room-card/recharge", s.mockRecharge)
			auth.GET("/wallet/recharge/history", s.rechargeHistory)

			auth.GET("/lobby/games", s.listLobbyGames)
			auth.PUT("/lobby/games", s.updateLobbyGames)

			auth.POST("/rooms", s.createRoom)
			auth.GET("/rooms/:room_id", s.getRoom)
			auth.POST("/rooms/:room_id/join", s.joinRoom)
		}
	}
}

func (s *Server) playerID(c *gin.Context) (int64, error) {
	claims, err := s.accessClaims(c)
	if err != nil {
		return 0, err
	}
	p, err := s.players.EnsureFromAuth(c.Request.Context(), claims.AccountID, claims.Phone, claims.Nickname, claims.IsMinor)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "player link failed", "request_id": c.GetString("request_id")})
		return 0, err
	}
	c.Set("user_id", strconv.FormatInt(p.ID, 10))
	c.Set("auth_account_id", strconv.FormatInt(claims.AccountID, 10))
	return p.ID, nil
}

func (s *Server) accessClaims(c *gin.Context) (*user.AccessClaims, error) {
	token := c.GetString("access_token")
	if token == "" {
		auth := c.GetHeader("Authorization")
		if len(auth) > 7 {
			token = auth[7:]
		}
	}
	claims, err := user.ParseAccessClaims(token, s.cfg.JWTPublicKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "invalid token", "request_id": c.GetString("request_id")})
		return nil, err
	}
	return claims, nil
}

func parseClubID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("club_id"), 10, 64)
}

func (s *Server) createRoom(c *gin.Context) {
	uid, err := s.playerID(c)
	if err != nil {
		return
	}
	var req struct {
		GameID      string `json:"game_id"`
		RoomMode    string `json:"room_mode"`
		PlayerCount int    `json:"player_count"`
		ClubID      *int64 `json:"club_id"`
		FillBots    bool   `json:"fill_bots"`
		TierID      string `json:"tier_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	if req.GameID == "" {
		req.GameID = "dawugui"
	}
	if req.RoomMode == "" {
		req.RoomMode = "room_card"
	}
	if req.RoomMode != "room_card" && req.RoomMode != "score_field" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "room_mode must be room_card or score_field", "request_id": c.GetString("request_id")})
		return
	}
	if req.PlayerCount == 0 {
		req.PlayerCount = 4
	}
	if err := s.catalog.ValidatePlayerCount(c.Request.Context(), req.GameID, req.PlayerCount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}

	var cost int64
	baseScore := int64(1)
	cfg := map[string]interface{}{
		"base_score": baseScore,
		"fill_bots":  req.FillBots,
	}

	if req.RoomMode == "score_field" {
		tier, err := s.catalog.GetTier(c.Request.Context(), req.GameID, req.TierID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "tier_id required for score_field", "request_id": c.GetString("request_id")})
			return
		}
		bal, err := s.wallet.GameCoinBalance(c.Request.Context(), uid, req.GameID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
			return
		}
		if bal < tier.MinCoin {
			c.JSON(http.StatusConflict, gin.H{"code": 2002, "message": "insufficient game coin for tier", "request_id": c.GetString("request_id")})
			return
		}
		cost = 0
		baseScore = tier.BaseScore
		cfg["base_score"] = baseScore
		cfg["tier_id"] = tier.TierID
	} else {
		cost, err = s.catalog.RoomCardCost(c.Request.Context(), req.GameID, req.PlayerCount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
			return
		}
	}

	if req.ClubID != nil {
		if err := s.clubs.RequireMember(c.Request.Context(), *req.ClubID, uid); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error(), "request_id": c.GetString("request_id")})
			return
		}
	}
	idem := c.GetHeader("Idempotency-Key")
	auditSN := s.audit.Next()
	r, err := s.rooms.Create(c.Request.Context(), room.CreateParams{
		OwnerID: uid, ClubID: req.ClubID, GameID: req.GameID, RoomMode: req.RoomMode,
		PlayerCount: req.PlayerCount, Config: cfg,
		WSURL: s.cfg.PitayaWSURL, IdempotencyKey: idem, RoomCardCost: cost, AuditSN: auditSN,
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": 2001, "message": err.Error(), "request_id": c.GetString("request_id"), "audit_sn": auditSN})
		return
	}
	_ = s.lobby.TouchLastPlayed(c.Request.Context(), uid, req.GameID)
	c.JSON(http.StatusCreated, gin.H{
		"room_id":        r.RoomID.String(),
		"ws_url":         r.WSURL,
		"game_id":        r.GameID,
		"room_mode":      r.RoomMode,
		"audit_sn":       auditSN,
		"cost":           cost,
		"room_card_cost": cost,
		"base_score":     baseScore,
	})
}

func (s *Server) getRoom(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("room_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid room_id", "request_id": c.GetString("request_id")})
		return
	}
	r, err := s.rooms.Get(c.Request.Context(), rid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found", "request_id": c.GetString("request_id")})
		return
	}
	resp := gin.H{
		"room_id": r.RoomID.String(), "game_id": r.GameID, "room_mode": r.RoomMode,
		"status": r.Status, "player_count": r.PlayerCount, "ws_url": r.WSURL,
	}
	if r.ClubID != nil {
		resp["club_id"] = *r.ClubID
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Server) joinRoom(c *gin.Context) {
	uid, err := s.playerID(c)
	if err != nil {
		return
	}
	rid, err := uuid.Parse(c.Param("room_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid room_id", "request_id": c.GetString("request_id")})
		return
	}
	wsURL, err := s.rooms.JoinAllowed(c.Request.Context(), rid, uid)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ws_url": wsURL})
}
