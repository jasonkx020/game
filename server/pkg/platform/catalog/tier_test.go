package catalog

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalTiers(t *testing.T) {
	raw := []byte(`{
		"room_card_cost": {"4": 2},
		"register_gift": {"game_coin": 5000},
		"coin_name": "龟币",
		"tiers": [
			{"tier_id":"novice","name":"新手场","base_score":100,"min_coin":5000},
			{"tier_id":"junior","name":"初级场","base_score":500,"min_coin":25000}
		]
	}`)
	var cfg GameConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Tiers) != 2 {
		t.Fatalf("tiers=%d", len(cfg.Tiers))
	}
	if cfg.Tiers[0].BaseScore != 100 || cfg.Tiers[1].MinCoin != 25000 {
		t.Fatalf("%+v", cfg.Tiers)
	}
}
