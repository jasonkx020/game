package dawugui

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/example/game/internal/game/engine"
	pb "github.com/example/game/internal/gen/pitaya/pitaya"
	"google.golang.org/protobuf/proto"
)

const GameID = "dawugui"

const (
	phaseIdle = iota
	phasePlaying
	phaseEnded
)

type State struct {
	Phase         int
	Players       []engine.Player
	Hands         map[uint32][]uint32
	CurrentSeat   uint32
	FirstSeat     uint32
	LastPlaySeat  uint32
	LastPlayCards []uint32
	LastPlayType  int
	MustPlay      bool
	Passed        map[uint32]bool
	PlayCount     int
	WinnerSeat    uint32
	AlertSeats    map[uint32]bool
}

func (s *State) GameID() string { return GameID }
func (s *State) Clone() engine.GameState {
	cp := *s
	cp.Hands = make(map[uint32][]uint32, len(s.Hands))
	for k, v := range s.Hands {
		cp.Hands[k] = append([]uint32(nil), v...)
	}
	cp.Passed = make(map[uint32]bool, len(s.Passed))
	for k, v := range s.Passed {
		cp.Passed[k] = v
	}
	cp.AlertSeats = make(map[uint32]bool, len(s.AlertSeats))
	for k, v := range s.AlertSeats {
		cp.AlertSeats[k] = v
	}
	cp.LastPlayCards = append([]uint32(nil), s.LastPlayCards...)
	return &cp
}

type Engine struct{}

func New() *Engine { return &Engine{} }

func (e *Engine) Meta() engine.GameMeta {
	return engine.GameMeta{GameID: GameID, MinPlayers: 3, MaxPlayers: 5}
}

func (e *Engine) NewState(cfg engine.GameConfig, players []engine.Player) (engine.GameState, []engine.GameEvent, error) {
	if len(players) < 3 || len(players) > 5 {
		return nil, nil, fmt.Errorf("invalid player count")
	}
	deck := newDeck()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })

	hands := make(map[uint32][]uint32)
	for i, p := range players {
		hands[p.Seat] = nil
		_ = i
	}
	idx := 0
	for idx < len(deck) {
		for _, p := range players {
			if idx >= len(deck) {
				break
			}
			hands[p.Seat] = append(hands[p.Seat], deck[idx])
			idx++
		}
	}
	for seat := range hands {
		sort.Slice(hands[seat], func(i, j int) bool { return cardRank(hands[seat][i]) < cardRank(hands[seat][j]) })
	}

	firstSeat := players[0].Seat
	for _, p := range players {
		for _, c := range hands[p.Seat] {
			if c == diamond3() {
				firstSeat = p.Seat
				break
			}
		}
	}

	st := &State{
		Phase: phasePlaying, Players: players, Hands: hands,
		CurrentSeat: firstSeat, FirstSeat: firstSeat, MustPlay: true,
		Passed: make(map[uint32]bool), AlertSeats: make(map[uint32]bool),
	}
	var events []engine.GameEvent
	for _, p := range players {
		ev, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Deal{Deal: &pb.DealEvent{
			HandCards: hands[p.Seat], FirstSeat: firstSeat, DealerSeat: p.Seat, TargetSeat: p.Seat,
		}}})
		events = append(events, engine.GameEvent{Type: engine.EventDeal, PushRoute: "onDeal", Seat: p.Seat, Payload: ev})
	}
	turnEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Turn{Turn: &pb.TurnEvent{
		CurrentSeat: firstSeat, TimeoutMs: 30000, MustPlay: true,
	}}})
	events = append(events, engine.GameEvent{Type: engine.EventTurn, PushRoute: "onTurnNotify", Payload: turnEv})
	return st, events, nil
}

func (e *Engine) ApplyAction(state engine.GameState, action engine.Action) (engine.GameState, []engine.GameEvent, error) {
	st, ok := state.(*State)
	if !ok {
		return nil, nil, fmt.Errorf("invalid state")
	}
	if st.Phase != phasePlaying {
		return nil, nil, fmt.Errorf("not in playing phase")
	}
	if action.Seat != st.CurrentSeat {
		return nil, nil, fmt.Errorf("not your turn")
	}

	var events []engine.GameEvent
	if action.Kind == engine.ActionPass {
		if st.MustPlay && st.PlayCount == 0 {
			return nil, nil, fmt.Errorf("must play first")
		}
		if len(st.LastPlayCards) == 0 {
			return nil, nil, fmt.Errorf("cannot pass on free play")
		}
		st.Passed[action.Seat] = true
		next := e.nextSeat(st, action.Seat)
		if e.allPassedExcept(st, st.LastPlaySeat) {
			st.LastPlayCards = nil
			st.LastPlayType = 0
			st.Passed = make(map[uint32]bool)
			st.CurrentSeat = st.LastPlaySeat
			st.MustPlay = false
		} else {
			st.CurrentSeat = next
		}
		passEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Pass{Pass: &pb.PassEvent{
			Seat: action.Seat, NextSeat: st.CurrentSeat,
		}}})
		events = append(events, engine.GameEvent{Type: engine.EventPass, PushRoute: "onPlayResult", Seat: action.Seat, Payload: passEv})
	} else {
		if err := validatePlay(st, action.Cards); err != nil {
			return nil, nil, err
		}
		removeCards(st, action.Seat, action.Cards)
		st.PlayCount++
		st.LastPlaySeat = action.Seat
		st.LastPlayCards = append([]uint32(nil), action.Cards...)
		st.LastPlayType = playType(action.Cards)
		st.Passed = make(map[uint32]bool)
		st.MustPlay = false

		if len(st.Hands[action.Seat]) == 1 {
			st.AlertSeats[action.Seat] = true
			alertEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Alert{Alert: &pb.AlertEvent{
				Seat: action.Seat, HandCount: 1,
			}}})
			events = append(events, engine.GameEvent{Type: engine.EventAlert, PushRoute: "onAlert", Seat: action.Seat, Payload: alertEv})
		}
		if len(st.Hands[action.Seat]) == 0 {
			st.Phase = phaseEnded
			st.WinnerSeat = action.Seat
			playEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Play{Play: &pb.PlayEvent{
				Seat: action.Seat, Cards: action.Cards, PlayType: 1,
			}}})
			events = append(events, engine.GameEvent{Type: engine.EventPlay, PushRoute: "onPlayResult", Seat: action.Seat, Payload: playEv})
			return st, events, nil
		}
		st.CurrentSeat = e.nextSeat(st, action.Seat)
		playEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Play{Play: &pb.PlayEvent{
			Seat: action.Seat, Cards: action.Cards, PlayType: 1, NextSeat: st.CurrentSeat,
		}}})
		events = append(events, engine.GameEvent{Type: engine.EventPlay, PushRoute: "onPlayResult", Seat: action.Seat, Payload: playEv})
	}

	turnEv, _ := proto.Marshal(&pb.GameEvent{Body: &pb.GameEvent_Turn{Turn: &pb.TurnEvent{
		CurrentSeat: st.CurrentSeat, TimeoutMs: 30000, MustPlay: len(st.LastPlayCards) > 0,
		LastPlaySeat: st.LastPlaySeat,
	}}})
	events = append(events, engine.GameEvent{Type: engine.EventTurn, PushRoute: "onTurnNotify", Payload: turnEv})
	return st, events, nil
}

func (e *Engine) OnTick(state engine.GameState, now time.Time) (engine.GameState, []engine.GameEvent, error) {
	_ = now
	return state, nil, nil
}

func (e *Engine) VisibleState(state engine.GameState, seat uint32) (interface{}, error) {
	st := state.(*State)
	hand := append([]uint32(nil), st.Hands[seat]...)
	return map[string]interface{}{
		"seat": seat, "hand": hand, "current_seat": st.CurrentSeat,
	}, nil
}

func (e *Engine) CheckRoundEnd(state engine.GameState) (engine.RoundEnd, bool) {
	st := state.(*State)
	if st.Phase == phaseEnded {
		for _, p := range st.Players {
			if p.Seat == st.WinnerSeat {
				return engine.RoundEnd{Valid: true, WinnerID: p.UserID, WinnerSeat: p.Seat}, true
			}
		}
	}
	return engine.RoundEnd{}, false
}

func (e *Engine) CalcSettlement(state engine.GameState, end engine.RoundEnd) (engine.SettlementResult, error) {
	st := state.(*State)
	scores := make([]engine.PlayerScore, len(st.Players))
	for i, p := range st.Players {
		score := int32(-1)
		if p.Seat == end.WinnerSeat {
			score = int32(len(st.Players) - 1)
		}
		scores[i] = engine.PlayerScore{UserID: p.UserID, Seat: p.Seat, RuleScore: score}
	}
	return engine.SettlementResult{Valid: end.Valid, WinnerID: end.WinnerID, Scores: scores}, nil
}

func (e *Engine) nextSeat(st *State, from uint32) uint32 {
	seats := make([]uint32, len(st.Players))
	for i, p := range st.Players {
		seats[i] = p.Seat
	}
	sort.Slice(seats, func(i, j int) bool { return seats[i] < seats[j] })
	for i, s := range seats {
		if s == from {
			return seats[(i+1)%len(seats)]
		}
	}
	return from
}

func (e *Engine) allPassedExcept(st *State, except uint32) bool {
	for _, p := range st.Players {
		if p.Seat == except {
			continue
		}
		if !st.Passed[p.Seat] {
			return false
		}
	}
	return true
}

func newDeck() []uint32 {
	d := make([]uint32, 0, 54)
	for suit := uint32(0); suit < 4; suit++ {
		for rank := uint32(0); rank < 13; rank++ {
			d = append(d, suit*13+rank)
		}
	}
	d = append(d, 52, 53)
	return d
}

func diamond3() uint32 { return 3*13 + 0 }

func cardRank(c uint32) int {
	if c >= 52 {
		return 16 + int(c-52)
	}
	r := int(c % 13)
	if r == 12 {
		return 15
	}
	return r + 3
}

func playType(cards []uint32) int {
	switch len(cards) {
	case 1:
		return 1
	case 2:
		return 2
	case 3:
		return 3
	case 4:
		return 4
	default:
		return 0
	}
}

func validatePlay(st *State, cards []uint32) error {
	if len(cards) == 0 {
		return fmt.Errorf("empty cards")
	}
	hand := st.Hands[st.CurrentSeat]
	if !hasCards(hand, cards) {
		return fmt.Errorf("cards not in hand")
	}
	pt := playType(cards)
	if pt == 0 {
		return fmt.Errorf("invalid play type")
	}
	if st.PlayCount == 0 {
		if !contains(cards, diamond3()) {
			return fmt.Errorf("first play must include diamond 3")
		}
	}
	if len(st.LastPlayCards) > 0 {
		if pt != st.LastPlayType {
			return fmt.Errorf("must play same type")
		}
		if playRank(cards) <= playRank(st.LastPlayCards) {
			return fmt.Errorf("must beat last play")
		}
	}
	ranks := make([]int, len(cards))
	for i, c := range cards {
		if c >= 52 && len(cards) > 1 {
			return fmt.Errorf("joker cannot combo")
		}
		ranks[i] = cardRank(c)
	}
	for i := 1; i < len(ranks); i++ {
		if ranks[i] != ranks[0] {
			return fmt.Errorf("invalid combo")
		}
	}
	return nil
}

func playRank(cards []uint32) int {
	if len(cards) == 0 {
		return 0
	}
	return cardRank(cards[0])
}

func hasCards(hand, play []uint32) bool {
	tmp := append([]uint32(nil), hand...)
	for _, c := range play {
		found := false
		for i, h := range tmp {
			if h == c {
				tmp = append(tmp[:i], tmp[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func removeCards(st *State, seat uint32, cards []uint32) {
	h := st.Hands[seat]
	for _, c := range cards {
		for i, x := range h {
			if x == c {
				h = append(h[:i], h[i+1:]...)
				break
			}
		}
	}
	st.Hands[seat] = h
}

func contains(cards []uint32, c uint32) bool {
	for _, x := range cards {
		if x == c {
			return true
		}
	}
	return false
}
