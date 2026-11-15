package game

import (
	"fmt"

	"github.com/example/game/internal/game/dawugui"
	"github.com/example/game/internal/game/engine"
	"github.com/example/game/internal/game/liuzichong"
)

// Get returns the GameEngine for gameID. Composition-root only (with Bot).
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

// Bot returns the BotPicker for gameID. Composition-root only (with Get).
func Bot(gameID string) (engine.BotPicker, error) {
	switch gameID {
	case dawugui.GameID:
		return dawugui.NewBotPicker(), nil
	case liuzichong.GameID:
		return liuzichong.NewBotPicker(), nil
	default:
		return nil, fmt.Errorf("unknown game: %s", gameID)
	}
}
