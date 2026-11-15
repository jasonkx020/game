// Package dawugui is the public surface for the dawugui game module.
//
// Allowed exports for other packages (except composition roots cmd/game and
// internal/game registry): GameID, New, NewBotPicker.
//
// Implementation details live under dawugui/internal and must not be imported
// by other games or platform packages.
package dawugui
