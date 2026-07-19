package bot

import (
	"math/rand"

	"github.com/example/game/internal/game/engine"
	"github.com/example/game/internal/game/liuzichong"
)

// PickLiuzichongAction prefers capturing moves; ties broken randomly.
func PickLiuzichongAction(st *liuzichong.State, seat uint32) (engine.Action, bool) {
	moves := liuzichong.ListLegalMoves(st, seat)
	if len(moves) == 0 {
		return engine.Action{}, false
	}
	bestCap := -1
	var best []engine.Action
	for _, m := range moves {
		cap := liuzichong.PreviewCaptureCount(st, m)
		if cap > bestCap {
			bestCap = cap
			best = []engine.Action{m}
		} else if cap == bestCap {
			best = append(best, m)
		}
	}
	return best[rand.Intn(len(best))], true
}
