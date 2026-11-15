package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/example/admin-service/internal/authclient"
	"github.com/example/admin-service/internal/webui"
	"github.com/example/game/pkg/audit"
	"github.com/example/game/pkg/config"
	"github.com/example/game/pkg/httpmw"
	"github.com/example/game/pkg/platform/adminuser"
	"github.com/example/game/pkg/platform/catalog"
	"github.com/example/game/pkg/platform/club"
	"github.com/example/game/pkg/platform/metrics"
	"github.com/example/game/pkg/platform/moderation"
	"github.com/example/game/pkg/platform/player"
	"github.com/example/game/pkg/platform/user"
	"github.com/example/game/pkg/platform/wallet"
	goredis "github.com/redis/go-redis/v9"
)

// Server is the ops/admin HTTP API. It shares the game DB schema with game-api
// but is a separate process: stopping it must not affect player login/lobby/gameplay.
type Server struct {
	cfg        *config.Config
	admins     *adminuser.Service
	players    *player.Service
	wallet     *wallet.Service
	clubs      *club.Service
	catalog    *catalog.Service
	metrics    *metrics.Service
	moderation *moderation.Service
	authClient *authclient.Client
	audit      *audit.Generator
	rdb        *goredis.Client
	db         *sqlx.DB
}

func New(
	cfg *config.Config,
	admins *adminuser.Service,
	players *player.Service,
	wallet *wallet.Service,
	clubs *club.Service,
	catalog *catalog.Service,
	metrics *metrics.Service,
	mod *moderation.Service,
	authCLI *authclient.Client,
	gen *audit.Generator,
	rdb *goredis.Client,
	sqlDB *sqlx.DB,
) *Server {
	return &Server{
		cfg: cfg, admins: admins, players: players, wallet: wallet,
		clubs: clubs, catalog: catalog, metrics: metrics, moderation: mod,
		authClient: authCLI, audit: gen, rdb: rdb, db: sqlDB,
	}
}

func (s *Server) Register(r *gin.Engine) {
	r.Use(gin.Recovery(), httpmw.RequestID(), httpmw.CORS(s.cfg.CORSOrigins))

	sig := httpmw.Signature(httpmw.SignatureConfig{
		AppID: s.cfg.AppID, AppSecret: s.cfg.HMACSecret, Skip: false,
	}, s.rdb)

	jwtAdmin := httpmw.RequirePrincipalType(s.cfg.JWTSecret, user.PrincipalTypeAdmin)

	v1 := r.Group("/v1")
	v1.Use(sig)
	{
		admin := v1.Group("/admin")
		admin.Use(httpmw.JWT(s.cfg.JWTSecret), jwtAdmin, httpmw.RequireRole(s.cfg.JWTSecret, user.RolePlatformAdmin, user.RoleClubAdmin))
		{
			admin.POST("/clubs", s.createClub)
			admin.POST("/clubs/:club_id/members", s.addClubMember)
			admin.DELETE("/clubs/:club_id/members/:user_id", s.removeClubMember)
			admin.POST("/clubs/:club_id/room-card/transfer", s.transferToClubPool)

			admin.GET("/metrics/overview", s.adminMetricsOverview)
			admin.GET("/metrics/room-cards", s.adminMetricsRoomCards)
			admin.GET("/clubs", s.adminListClubs)

			admin.GET("/wallet/room-card", s.adminRoomCardBalance)
			admin.POST("/wallet/room-card/recharge", s.adminMockRecharge)
			admin.GET("/wallet/recharge/history", s.adminRechargeHistory)

			adminPlatform := admin.Group("")
			adminPlatform.Use(httpmw.RequireRole(s.cfg.JWTSecret, user.RolePlatformAdmin))
			{
				adminPlatform.GET("/users", s.adminListUsers)
				adminPlatform.POST("/users", s.createAdminUser)
				adminPlatform.GET("/users/:user_id", s.getAdminUser)
				adminPlatform.PATCH("/users/:user_id", s.patchAdminUser)
				adminPlatform.POST("/users/:user_id/logout", s.forceLogoutAdminUser)
				adminPlatform.GET("/players", s.adminListPlayers)

				adminPlatform.POST("/players/:player_id/ban", s.banPlayer)
				adminPlatform.POST("/players/:player_id/unban", s.unbanPlayer)
				adminPlatform.POST("/players/:player_id/kick", s.kickPlayer)

				adminPlatform.POST("/users/:user_id/disable", s.disableAdminUser)
				adminPlatform.POST("/users/:user_id/enable", s.enableAdminUser)
				adminPlatform.PATCH("/users/:user_id/role", s.patchAdminRole)
			}
		}

		adminRead := v1.Group("")
		adminRead.Use(httpmw.JWT(s.cfg.JWTSecret), jwtAdmin)
		{
			adminRead.GET("/games", s.listGames)
			adminRead.GET("/games/:game_id/config", s.getGameConfig)
			adminRead.GET("/clubs/:club_id", s.getClub)
			adminRead.GET("/clubs/:club_id/members", s.listClubMembers)
			adminRead.GET("/companion/personas", s.listCompanionPersonas)
		}
	}

	webui.Mount(r)
}

func (s *Server) adminID(c *gin.Context) (int64, error) {
	claims, err := s.accessClaims(c)
	if err != nil {
		return 0, err
	}
	u, err := s.admins.EnsureFromAuth(c.Request.Context(), claims.AccountID, claims.Phone, claims.Nickname, claims.Role)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "admin link failed", "request_id": c.GetString("request_id")})
		return 0, err
	}
	c.Set("user_id", strconv.FormatInt(u.ID, 10))
	c.Set("auth_account_id", strconv.FormatInt(claims.AccountID, 10))
	return u.ID, nil
}

func (s *Server) accessClaims(c *gin.Context) (*user.AccessClaims, error) {
	token := c.GetString("access_token")
	if token == "" {
		auth := c.GetHeader("Authorization")
		if len(auth) > 7 {
			token = auth[7:]
		}
	}
	claims, err := user.ParseAccessClaims(token, []byte(s.cfg.JWTSecret))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "invalid token", "request_id": c.GetString("request_id")})
		return nil, err
	}
	return claims, nil
}

func parseClubID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("club_id"), 10, 64)
}

func parseTargetUserID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("user_id"), 10, 64)
}
