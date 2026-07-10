package companion

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/example/game/internal/audit"
	"github.com/example/game/internal/config"
	"github.com/example/game/internal/platform/catalog"
	"github.com/example/game/internal/platform/companion/llm"
	"github.com/example/game/internal/platform/companion/tools"
	"github.com/example/game/internal/platform/lobby"
	"github.com/example/game/internal/platform/room"
)

var (
	ErrSessionNotFound = errors.New("companion session not found")
	ErrPersonaNotFound = errors.New("companion persona not found")
)

type Session struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	PersonaID string    `json:"persona_id" db:"persona_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Message struct {
	ID        int64     `json:"id" db:"id"`
	Role      string    `json:"role" db:"role"`
	Content   string    `json:"content" db:"content"`
	ToolName  string    `json:"tool_name,omitempty" db:"tool_name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Recommendation struct {
	GameID string `json:"game_id" db:"game_id"`
	Name   string `json:"name" db:"name"`
	Reason string `json:"reason"`
}

type Service struct {
	db     *sqlx.DB
	llm    *llm.Client
	tools  *tools.Registry
	audit  *audit.Generator
}

func NewService(
	db *sqlx.DB,
	llmClient *llm.Client,
	reg *tools.Registry,
	gen *audit.Generator,
) *Service {
	return &Service{db: db, llm: llmClient, tools: reg, audit: gen}
}

func NewToolRegistry(
	db *sqlx.DB,
	lobbySvc *lobby.Service,
	rooms *room.Service,
	catalogSvc *catalog.Service,
	cfg *config.Config,
	gen *audit.Generator,
) *tools.Registry {
	reg := tools.NewRegistry()
	tools.RegisterListGames(reg, lobbySvc)
	tools.RegisterCreateRoom(reg, tools.RoomDeps{
		Rooms: rooms, Catalog: catalogSvc, Lobby: lobbySvc, Cfg: cfg, Audit: gen,
	})
	tools.RegisterExplainRules(reg, db)
	tools.RegisterRecommend(reg, db)
	tools.RegisterGetRoomStatus(reg, rooms)
	tools.RegisterSuggestAction(reg, db)
	tools.RegisterSummarizeReplay(reg, db)
	return reg
}

func (s *Service) CreateSession(ctx context.Context, userID int64, personaID string) (*Session, error) {
	if personaID == "" {
		personaID = "default"
	}
	var ok bool
	if err := s.db.GetContext(ctx, &ok,
		`SELECT EXISTS(SELECT 1 FROM companion_persona WHERE persona_id=$1 AND enabled)`, personaID); err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrPersonaNotFound
	}
	var sess Session
	err := s.db.QueryRowxContext(ctx, `
		INSERT INTO companion_session (user_id, persona_id) VALUES ($1, $2)
		RETURNING id, user_id, persona_id, created_at`, userID, personaID).StructScan(&sess)
	return &sess, err
}

func (s *Service) GetSession(ctx context.Context, sessionID, userID int64) (*Session, error) {
	var sess Session
	err := s.db.GetContext(ctx, &sess,
		`SELECT id, user_id, persona_id, created_at FROM companion_session WHERE id=$1 AND user_id=$2`,
		sessionID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	return &sess, err
}

func (s *Service) ListMessages(ctx context.Context, sessionID, userID int64, limit int) ([]Message, error) {
	if _, err := s.GetSession(ctx, sessionID, userID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var msgs []Message
	err := s.db.SelectContext(ctx, &msgs, `
		SELECT id, role, content, tool_name, created_at FROM companion_message
		WHERE session_id=$1 ORDER BY id DESC LIMIT $2`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (s *Service) ChatStream(ctx context.Context, sessionID, userID int64, userText string, w io.Writer) error {
	sess, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}
	sn := s.audit.Next()
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO companion_message (session_id, role, content, audit_sn) VALUES ($1, 'user', $2, $3)`,
		sessionID, userText, sn)

	systemPrompt, _ := s.loadPersonaPrompt(ctx, sess.PersonaID)
	history, _ := s.ListMessages(ctx, sessionID, userID, 20)
	msgs := []llm.Message{{Role: "system", Content: systemPrompt}}
	for _, m := range history {
		if m.Role == "user" || m.Role == "assistant" {
			msgs = append(msgs, llm.Message{Role: m.Role, Content: m.Content})
		}
	}

	// Tool round (non-stream)
	toolDefs := toLLMTools(s.tools.Definitions())
	reply, err := s.llm.Chat(ctx, msgs, toolDefs)
	if err != nil {
		return err
	}

	finalText := reply.Content
	if len(reply.ToolCalls) > 0 {
		msgs = append(msgs, reply)
		tc := &tools.Context{UserID: userID}
		for _, call := range reply.ToolCalls {
			res, err := s.tools.Execute(ctx, tc, call.Function.Name, json.RawMessage(call.Function.Arguments))
			if err != nil {
				res = tools.Result{Name: call.Function.Name, Content: fmt.Sprintf(`{"error":%q}`, err.Error())}
			}
			_, _ = s.db.ExecContext(ctx, `
				INSERT INTO companion_message (session_id, role, content, tool_name, audit_sn)
				VALUES ($1, 'tool', $2, $3, $4)`,
				sessionID, res.Content, res.Name, s.audit.Next())
			msgs = append(msgs, llm.Message{
				Role: "tool", Name: res.Name, Content: res.Content, ToolCallID: call.ID,
			})
		}
		reply2, err := s.llm.Chat(ctx, msgs, nil)
		if err != nil {
			return err
		}
		finalText = reply2.Content
	}

	if finalText == "" {
		finalText = s.heuristicTools(ctx, userID, sessionID, userText)
	}

	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO companion_message (session_id, role, content, audit_sn) VALUES ($1, 'assistant', $2, $3)`,
		sessionID, finalText, s.audit.Next())
	_, _ = s.db.ExecContext(ctx, `UPDATE companion_session SET updated_at=$1 WHERE id=$2`, time.Now(), sessionID)

	_, err = fmt.Fprintf(w, "data: %s\n\n", sseJSON(finalText))
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, "data: [DONE]\n\n")
	return err
}

func (s *Service) heuristicTools(ctx context.Context, userID, sessionID int64, text string) string {
	tc := &tools.Context{UserID: userID}
	switch {
	case strings.Contains(text, "复盘") || strings.Contains(text, "战绩"):
		res, _ := s.tools.Execute(ctx, tc, "summarize_replay", nil)
		var parsed map[string]string
		_ = json.Unmarshal([]byte(res.Content), &parsed)
		if parsed["summary"] != "" {
			return parsed["summary"]
		}
		return res.Content
	case strings.Contains(text, "推荐"):
		res, _ := s.tools.Execute(ctx, tc, "recommend_games", nil)
		return "给你推荐这些游戏：" + res.Content + "。想玩哪个跟我说，我可以帮你开房！"
	case strings.Contains(text, "规则"):
		gameID := "dawugui"
		if strings.Contains(text, "六子冲") || strings.Contains(text, "liuzichong") {
			gameID = "liuzichong"
		}
		args, _ := json.Marshal(map[string]string{"game_id": gameID})
		res, _ := s.tools.Execute(ctx, tc, "explain_rules", args)
		var parsed map[string]string
		_ = json.Unmarshal([]byte(res.Content), &parsed)
		if parsed["summary"] != "" {
			return parsed["summary"]
		}
		return res.Content
	case strings.Contains(text, "开") && (strings.Contains(text, "乌龟") || strings.Contains(text, "dawugui")):
		args, _ := json.Marshal(map[string]interface{}{"game_id": "dawugui", "player_count": 4})
		res, _ := s.tools.Execute(ctx, tc, "create_room", args)
		return "已帮你开好房啦！房间信息：" + res.Content + "。点游戏架进入或让我带你进房～"
	case strings.Contains(text, "开") && strings.Contains(text, "六子冲"):
		args, _ := json.Marshal(map[string]interface{}{"game_id": "liuzichong", "player_count": 2})
		res, _ := s.tools.Execute(ctx, tc, "create_room", args)
		return "六子冲房间开好啦：" + res.Content
	default:
		return ""
	}
}

func (s *Service) Recommendations(ctx context.Context, userID int64) ([]Recommendation, error) {
	var rows []Recommendation
	err := s.db.SelectContext(ctx, &rows, `
		SELECT c.game_id, c.name FROM game_catalog c
		LEFT JOIN user_game_prefs p ON p.game_id = c.game_id AND p.user_id = $1
		WHERE c.enabled = true
		ORDER BY p.pinned DESC, p.last_played_at DESC NULLS LAST, c.sort_order
		LIMIT 6`, userID)
	for i := range rows {
		if i == 0 {
			rows[i].Reason = "根据你的游玩记录推荐"
		} else {
			rows[i].Reason = "平台热门"
		}
	}
	return rows, err
}

func (s *Service) ListPersonas(ctx context.Context) ([]map[string]string, error) {
	var rows []struct {
		PersonaID string `db:"persona_id"`
		Name      string `db:"name"`
		AvatarURL string `db:"avatar_url"`
	}
	err := s.db.SelectContext(ctx, &rows,
		`SELECT persona_id, name, avatar_url FROM companion_persona WHERE enabled ORDER BY persona_id`)
	out := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]string{
			"persona_id": r.PersonaID, "name": r.Name, "avatar_url": r.AvatarURL,
		})
	}
	return out, err
}

func (s *Service) loadPersonaPrompt(ctx context.Context, personaID string) (string, error) {
	var prompt string
	err := s.db.GetContext(ctx, &prompt,
		`SELECT system_prompt FROM companion_persona WHERE persona_id=$1`, personaID)
	return prompt, err
}

func toLLMTools(defs []tools.Definition) []llm.ToolDef {
	out := make([]llm.ToolDef, 0, len(defs))
	for _, d := range defs {
		td := llm.ToolDef{Type: "function"}
		td.Function.Name = d.Name
		td.Function.Description = d.Description
		td.Function.Parameters = d.Parameters
		out = append(out, td)
	}
	return out
}

func sseJSON(text string) string {
	b, _ := json.Marshal(map[string]string{"content": text})
	return string(b)
}
