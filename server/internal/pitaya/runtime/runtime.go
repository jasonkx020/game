package runtime

import (
	"sort"
	"sync"

	"github.com/google/uuid"

	"github.com/example/game/internal/game/engine"
)

type SeatRole int

const (
	SeatRolePlayer SeatRole = iota
	SeatRoleSpectator // reserved for future observer seats
)

type Seat struct {
	UserID   uint64
	Seat     uint32
	Nickname string
	Ready    bool
	Online   bool
	Role     SeatRole
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
		Seats:  make(map[uint64]*Seat),
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

func (r *RoomRuntime) PlayerCount() int {
	n := 0
	for _, s := range r.Seats {
		if s.Role == SeatRolePlayer {
			n++
		}
	}
	return n
}

func (r *RoomRuntime) Players() []engine.Player {
	out := make([]engine.Player, 0, len(r.Seats))
	for _, s := range r.Seats {
		if s.Role == SeatRolePlayer {
			out = append(out, engine.Player{UserID: s.UserID, Seat: s.Seat, Nickname: s.Nickname})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seat < out[j].Seat })
	return out
}

// PlayerSeats returns proto-ready seat snapshots sorted by seat index.
func (r *RoomRuntime) PlayerSeats() []Seat {
	out := make([]Seat, 0, len(r.Seats))
	for _, s := range r.Seats {
		if s.Role == SeatRolePlayer {
			out = append(out, *s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seat < out[j].Seat })
	return out
}

func (r *RoomRuntime) AllReady(minPlayers int) bool {
	if r.PlayerCount() < minPlayers {
		return false
	}
	for _, s := range r.Seats {
		if s.Role == SeatRolePlayer && !s.Ready {
			return false
		}
	}
	return true
}
