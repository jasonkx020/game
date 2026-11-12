package game

import (
	"fmt"

	"github.com/example/game/internal/game/dawugui"
	"github.com/example/game/internal/game/engine"
	"github.com/example/game/internal/game/liuzichong"
)

func Get(gameID string) (engine.GameEngine, error) {
	switch gameID {
	case dawugui.GameID:
		return dawugui.New(), nil
	case liuzichong.GameID:
		return liuzichong.New(), nil
	default:
		return nil, fmt.Errorf("unknown game: %s", gameID)
	}
}
