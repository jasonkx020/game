package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/example/game/internal/audit"
	"github.com/example/game/internal/config"
	"github.com/example/game/internal/platform/catalog"
	"github.com/example/game/internal/platform/club"
	"github.com/example/game/internal/platform/companion"
	"github.com/example/game/internal/platform/httpmw"
	"github.com/example/game/internal/platform/lobby"
	"github.com/example/game/internal/platform/metrics"
	"github.com/example/game/internal/platform/room"
	"github.com/example/game/internal/platform/user"
	"github.com/example/game/internal/platform/wallet"
	goredis "github.com/redis/go-redis/v9"
)

type Server struct {
	cfg     *config.Config
	users   *user.Service
	wallet  *wallet.Service
	rooms   *room.Service
	clubs   *club.Service
	catalog *catalog.Service
	lobby     *lobby.Service
	companion *companion.Service
	metrics   *metrics.Service
	audit   *audit.Generator
	rdb     *goredis.Client
}

func New(
	cfg *config.Config,
	users *user.Service,
	wallet *wallet.Service,
	rooms *room.Service,
	clubs *club.Service,
	catalog *catalog.Service,
	lobbySvc *lobby.Service,
	companionSvc *companion.Service,
	metrics *metrics.Service,
	gen *audit.Generator,
	rdb *goredis.Client,
) *Server {
	return &Server{
		cfg: cfg, users: users, wallet: wallet, rooms: rooms,
		clubs: clubs, catalog: catalog, lobby: lobbySvc, companion: companionSvc, metrics: metrics,
		audit: gen, rdb: rdb,
	}
}

func (s *Server) Register(r *gin.Engine) {
	r.Use(gin.Recovery(), httpmw.RequestID(), httpmw.CORS(s.cfg.CORSOrigins))

	sig := httpmw.Signature(httpmw.SignatureConfig{
		AppID: s.cfg.AppID, AppSecret: s.cfg.HMACSecret, Skip: false,
	}, s.rdb)

	v1 := r.Group("/v1")
	v1.Use(sig)
	{
		v1.POST("/auth/login", s.login)
		v1.POST("/auth/refresh", httpmw.JWT(s.cfg.JWTSecret), s.refresh)

		auth := v1.Group("")
		auth.Use(httpmw.JWT(s.cfg.JWTSecret))
		{
			auth.GET("/user/profile", s.profile)
			auth.GET("/user/settings", s.getUserSettings)
			auth.PUT("/user/settings", s.putUserSettings)

			auth.GET("/lobby/recommendations", s.lobbyRecommendations)
			auth.POST("/companion/sessions", s.createCompanionSession)
			auth.GET("/companion/sessions/:session_id/messages", s.listCompanionMessages)
			auth.POST("/companion/sessions/:session_id/chat", s.companionChat)
			auth.GET("/companion/personas", s.listCompanionPersonas)

			auth.GET("/wallet/room-card", s.roomCardBalance)
			auth.GET("/wallet/game-coin/:game_id", s.gameCoinBalance)
			auth.POST("/wallet/room-card/recharge", s.mockRecharge)
			auth.GET("/wallet/recharge/history", s.rechargeHistory)

			auth.GET("/games", s.listGames)
			auth.GET("/games/:game_id/config", s.getGameConfig)

			auth.GET("/lobby/games", s.listLobbyGames)
			auth.PUT("/lobby/games", s.updateLobbyGames)

			auth.POST("/clubs", s.createClub)
			auth.GET("/clubs/:club_id", s.getClub)
			auth.GET("/clubs/:club_id/members", s.listClubMembers)
			auth.POST("/clubs/:club_id/members", s.addClubMember)
			auth.DELETE("/clubs/:club_id/members/:user_id", s.removeClubMember)
			auth.GET("/clubs/:club_id/room-card", s.clubPoolBalance)
			auth.POST("/clubs/:club_id/room-card/transfer", s.transferToClubPool)

			auth.POST("/rooms", s.createRoom)
			auth.GET("/rooms/:room_id", s.getRoom)
			auth.POST("/rooms/:room_id/join", s.joinRoom)
		}

		admin := v1.Group("/admin")
		admin.Use(httpmw.JWT(s.cfg.JWTSecret), httpmw.RequireRole(s.cfg.JWTSecret, user.RolePlatformAdmin, user.RoleClubAdmin))
		{
			admin.GET("/metrics/overview", s.adminMetricsOverview)
			admin.GET("/metrics/room-cards", s.adminMetricsRoomCards)
			admin.GET("/clubs", s.adminListClubs)
		}
	}
}

func (s *Server) userID(c *gin.Context) (int64, error) {
	token := c.GetString("access_token")
	if token == "" {
		auth := c.GetHeader("Authorization")
		if len(auth) > 7 {
			token = auth[7:]
		}
	}
	uid, err := user.ParseUserID(token, []byte(s.cfg.JWTSecret))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "invalid token", "request_id": c.GetString("request_id")})
		return 0, err
	}
	c.Set("user_id", strconv.FormatInt(uid, 10))
	return uid, nil
}

func parseClubID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("club_id"), 10, 64)
}

func parseTargetUserID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("user_id"), 10, 64)
}

func (s *Server) createRoom(c *gin.Context) {
	uid, err := s.userID(c)
	if err != nil {
		return
	}
	var req struct {
		GameID      string `json:"game_id"`
		RoomMode    string `json:"room_mode"`
		PlayerCount int    `json:"player_count"`
		ClubID      *int64 `json:"club_id"`
		FillBots    bool   `json:"fill_bots"`
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
	if req.PlayerCount == 0 {
		req.PlayerCount = 4
	}
	if err := s.catalog.ValidatePlayerCount(c.Request.Context(), req.GameID, req.PlayerCount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	cost, err := s.catalog.RoomCardCost(c.Request.Context(), req.GameID, req.PlayerCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	if req.ClubID != nil {
		if err := s.clubs.RequireAdmin(c.Request.Context(), *req.ClubID, uid); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error(), "request_id": c.GetString("request_id")})
			return
		}
	}
	idem := c.GetHeader("Idempotency-Key")
	auditSN := s.audit.Next()
	r, err := s.rooms.Create(c.Request.Context(), room.CreateParams{
		OwnerID: uid, ClubID: req.ClubID, GameID: req.GameID, RoomMode: req.RoomMode,
		PlayerCount: req.PlayerCount, Config: map[string]interface{}{
			"base_score": 1,
			"fill_bots":  req.FillBots,
		},
		WSURL: s.cfg.PitayaWSURL, IdempotencyKey: idem, RoomCardCost: cost, AuditSN: auditSN,
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": 2001, "message": err.Error(), "request_id": c.GetString("request_id"), "audit_sn": auditSN})
		return
	}
	_ = s.lobby.TouchLastPlayed(c.Request.Context(), uid, req.GameID)
	c.JSON(http.StatusCreated, gin.H{
		"room_id":  r.RoomID.String(),
		"ws_url":   r.WSURL,
		"game_id":  r.GameID,
		"audit_sn": auditSN,
		"cost":     cost,
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
	uid, err := s.userID(c)
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
