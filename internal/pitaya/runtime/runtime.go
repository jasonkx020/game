package runtime

import (
	"sync"

	"github.com/google/uuid"

	"github.com/example/game/internal/game/engine"
)

type Seat struct {
	UserID   uint64
	Seat     uint32
	Nickname string
	Ready    bool
	Online   bool
}

type RoomRuntime struct {
	mu          sync.Mutex
	RoomID      uuid.UUID
	GameID      string
	Seats       map[uint64]*Seat
	EngineState engine.GameState
	RoundID     uuid.UUID
	RoundNo     int
	ActionSeq   int
	RoomSeq     int
	Config      engine.GameConfig
}

type Store struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]*RoomRuntime
}

func NewStore() *Store {
	return &Store{rooms: make(map[uuid.UUID]*RoomRuntime)}
}

func (s *Store) GetOrCreate(roomID uuid.UUID, gameID string, maxPlayers int) *RoomRuntime {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.rooms[roomID]; ok {
		return r
	}
	r := &RoomRuntime{
		RoomID: roomID, GameID: gameID,
		Seats: make(map[uint64]*Seat),
		Config: engine.GameConfig{GameID: gameID, Players: maxPlayers, BaseScore: 1},
	}
	s.rooms[roomID] = r
	return r
}

func (s *Store) Get(roomID uuid.UUID) (*RoomRuntime, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rooms[roomID]
	return r, ok
}

func (r *RoomRuntime) Lock()   { r.mu.Lock() }
func (r *RoomRuntime) Unlock() { r.mu.Unlock() }

func (r *RoomRuntime) Players() []engine.Player {
	out := make([]engine.Player, 0, len(r.Seats))
	for _, s := range r.Seats {
		out = append(out, engine.Player{UserID: s.UserID, Seat: s.Seat, Nickname: s.Nickname})
	}
	return out
}

func (r *RoomRuntime) AllReady() bool {
	if len(r.Seats) < 3 {
		return false
	}
	for _, s := range r.Seats {
		if !s.Ready {
			return false
		}
	}
	return true
}
