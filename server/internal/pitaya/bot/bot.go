package bot

import (
	"fmt"
)

// BotUIDBase is the synthetic user_id range for room filler bots (no Pitaya session).
const BotUIDBase uint64 = 9_000_000_000_000_001

func UIDForSeat(seat uint32) uint64 {
	return BotUIDBase + uint64(seat)
}

func IsBot(userID uint64) bool {
	return userID >= BotUIDBase && userID < BotUIDBase+1000
}

func Nickname(seat uint32) string {
	return fmt.Sprintf("bot_%d", seat)
}
