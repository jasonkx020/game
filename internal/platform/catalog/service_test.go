package catalog

import (
	"testing"
)

func TestDefaultRoomCardCost(t *testing.T) {
	cases := map[int]int64{
		3: 2, 4: 2, 5: 3, 2: 2,
	}
	for pc, want := range cases {
		if got := defaultRoomCardCost(pc); got != want {
			t.Errorf("playerCount %d: got %d want %d", pc, got, want)
		}
	}
}

func TestGameConfigRoomCardCost(t *testing.T) {
	cfg := &GameConfig{
		RoomCardCost: map[string]int64{"3": 2, "4": 2, "5": 3},
	}
	if cfg.RoomCardCost["4"] != 2 {
		t.Fatal("expected cost 2 for 4 players")
	}
}
