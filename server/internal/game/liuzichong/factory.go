package liuzichong

import (
	"github.com/example/game/internal/game/engine"
	"github.com/example/game/internal/game/liuzichong/internal/rules"
)

// GameID is the catalog / room game_id for liuzichong.
const GameID = rules.GameID

// New returns the liuzichong GameEngine.
func New() engine.GameEngine {
	return rules.New()
}

// NewBotPicker returns the liuzichong bot policy.
func NewBotPicker() engine.BotPicker {
	return rules.NewBotPicker()
}
