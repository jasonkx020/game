package dawugui

import (
	"github.com/example/game/internal/game/dawugui/internal/rules"
	"github.com/example/game/internal/game/engine"
)

// GameID is the catalog / room game_id for dawugui.
const GameID = rules.GameID

// New returns the dawugui GameEngine.
func New() engine.GameEngine {
	return rules.New()
}

// NewBotPicker returns the dawugui bot policy.
func NewBotPicker() engine.BotPicker {
	return rules.NewBotPicker()
}
