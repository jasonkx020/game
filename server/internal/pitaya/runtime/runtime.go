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
	SeatRoleSpectator
)

type Seat struct {
	UserID      uint64
	Seat        uint32
	Nickname    string
	Ready       bool
	Online      bool
	Role        SeatRole
	Trusteeship bool
	// Risk presence (not pushed to clients)
	ClientIP string
	DeviceID string
	Lat      *float64
	Lng      *float64
}

type RoomRuntime struct {
	mu          sync.Mutex
	RoomID      uuid.UUID
	GameID      string
	RoomMode    string // room_card | score_field
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

type CreateOpts struct {
	GameID      string
	MaxPlayers  int
	BaseScore   int64
	RoomMode    string
}

func (s *Store) GetOrCreate(roomID uuid.UUID, gameID string, maxPlayers int) *RoomRuntime {
	return s.GetOrCreateWith(roomID, CreateOpts{GameID: gameID, MaxPlayers: maxPlayers, BaseScore: 1, RoomMode: "room_card"})
}

func (s *Store) GetOrCreateWith(roomID uuid.UUID, opts CreateOpts) *RoomRuntime {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.rooms[roomID]; ok {
		return r
	}
	base := opts.BaseScore
	if base <= 0 {
		base = 1
	}
	mode := opts.RoomMode
	if mode == "" {
		mode = "room_card"
	}
	r := &RoomRuntime{
		RoomID: roomID, GameID: opts.GameID, RoomMode: mode,
		Seats:  make(map[uint64]*Seat),
		Config: engine.GameConfig{GameID: opts.GameID, Players: opts.MaxPlayers, BaseScore: base},
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

// PlayerSeats returns player seat snapshots sorted by seat index.
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

// AllSeats returns all seats including spectators, sorted by seat then role.
func (r *RoomRuntime) AllSeats() []Seat {
	out := make([]Seat, 0, len(r.Seats))
	for _, s := range r.Seats {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Seat != out[j].Seat {
			return out[i].Seat < out[j].Seat
		}
		return out[i].Role < out[j].Role
	})
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

// NextSpectatorSeat allocates a high seat index for spectators (does not collide with players).
func (r *RoomRuntime) NextSpectatorSeat() uint32 {
	var max uint32
	for _, s := range r.Seats {
		if s.Seat >= max {
			max = s.Seat + 1
		}
	}
	if max < 100 {
		return 100
	}
	return max
}

// MarkOffline sets Online=false for userID if seated (disconnect hook).
func (r *RoomRuntime) MarkOffline(userID uint64) bool {
	s, ok := r.Seats[userID]
	if !ok {
		return false
	}
	s.Online = false
	return true
}
