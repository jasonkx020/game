package engine

import "time"

type EventType string

const (
	EventRoomState    EventType = "ROOM_STATE"
	EventDeal         EventType = "DEAL"
	EventTurn         EventType = "TURN"
	EventPlay         EventType = "PLAY"
	EventPass         EventType = "PASS"
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
}

type ActionKind int

const (
	ActionPlay ActionKind = iota + 1
	ActionPass
)

type Action struct {
	Kind   ActionKind
	Seat   uint32
	Cards  []uint32
}

type RoundEnd struct {
	Valid    bool
	WinnerID uint64
	WinnerSeat uint32
	Reason   string
}

type PlayerScore struct {
	UserID    uint64
	Seat      uint32
	RuleScore int32
}

type SettlementResult struct {
	Valid   bool
	WinnerID uint64
	Scores  []PlayerScore
}

type GameMeta struct {
	GameID      string
	MinPlayers  int
	MaxPlayers  int
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
