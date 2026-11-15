package engine

import "time"

type EventType string

const (
	EventRoomState    EventType = "ROOM_STATE"
	EventBoardInit    EventType = "BOARD_INIT"
	EventDeal         EventType = "DEAL"
	EventTurn         EventType = "TURN"
	EventPlay         EventType = "PLAY"
	EventPass         EventType = "PASS"
	EventMove         EventType = "MOVE"
	EventAlert        EventType = "ALERT"
	EventRoundInvalid EventType = "ROUND_INVALID"
	EventSettlement   EventType = "SETTLEMENT"
)

type GameEvent struct {
	Type      EventType
	PushRoute string
	ActorUID  uint64
	Seat      uint32
	Payload   []byte
}

type Player struct {
	UserID   uint64
	Seat     uint32
	Nickname string
}

type GameConfig struct {
	GameID    string
	BaseScore int64
	Players   int
	// Seed for reproducible RNG; 0 means the game may use time-based entropy.
	Seed int64
}

type ActionKind int

const (
	ActionPlay ActionKind = iota + 1
	ActionPass
	ActionMove
)

// Action is a player move. Prefer Kind for Play/Pass/Move; use Opcode+Payload
// for game-specific ops (bid, peng, gang, …).
type Action struct {
	Kind             ActionKind
	Seat             uint32
	Cards            []uint32
	FromRow, FromCol int
	ToRow, ToCol     int
	Opcode           string
	Payload          []byte
}

type RoundEnd struct {
	Valid      bool
	WinnerID   uint64
	WinnerSeat uint32
	Reason     string
}

type PlayerScore struct {
	UserID    uint64
	Seat      uint32
	RuleScore int32
	// Rank is 1-based placement; 0 means unused (simple games may ignore).
	Rank int32
	// TeamID groups teammates for team-based scoring; empty if unused.
	TeamID string
}

type SettlementResult struct {
	Valid    bool
	WinnerID uint64
	// Draw is true for multi-way ties / no single winner; WinnerID may be 0.
	Draw   bool
	Scores []PlayerScore
}

type GameMeta struct {
	GameID     string
	MinPlayers int
	MaxPlayers int
}

type GameEngine interface {
	Meta() GameMeta
	NewState(cfg GameConfig, players []Player) (GameState, []GameEvent, error)
	ApplyAction(state GameState, action Action) (GameState, []GameEvent, error)
	OnTick(state GameState, now time.Time) (GameState, []GameEvent, error)
	VisibleState(state GameState, seat uint32) (interface{}, error)
	CheckRoundEnd(state GameState) (RoundEnd, bool)
	CalcSettlement(state GameState, end RoundEnd) (SettlementResult, error)
}

type GameState interface {
	GameID() string
	Clone() GameState
}

// TurnAware is implemented by in-play game states so platform bot runners
// can read whose turn it is without importing a concrete game package.
type TurnAware interface {
	TurnSeat() uint32
	IsPlaying() bool
}

// Phased is optional on GameState for multi-stage games (dealing/bidding/…).
type Phased interface {
	Phase() string
}

// Snapshotter is optional on GameEngine for reconnect snapshots beyond action-log sync.
type Snapshotter interface {
	Snapshot(state GameState, seat uint32) ([]GameEvent, error)
}

// SpectatorViewer is optional on GameEngine for full-table spectator views.
type SpectatorViewer interface {
	SpectatorState(state GameState) (interface{}, error)
}

// BotPicker chooses a bot action for a seat. Implementations live in each
// game package; platform code must only use this interface.
type BotPicker interface {
	Pick(state GameState, seat uint32) (Action, bool)
	ActionRoute(action Action) string
	SettlementRoute() string
}
