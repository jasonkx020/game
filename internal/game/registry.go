package game

import (
	"fmt"

	"github.com/example/game/internal/game/dawugui"
	"github.com/example/game/internal/game/engine"
)

func Get(gameID string) (engine.GameEngine, error) {
	switch gameID {
	case dawugui.GameID:
		return dawugui.New(), nil
	default:
		return nil, fmt.Errorf("unknown game: %s", gameID)
	}
}
