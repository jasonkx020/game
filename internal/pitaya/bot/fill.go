package bot

import (
	"github.com/example/game/internal/pitaya/runtime"
)

// FillSeats adds ready bot players until target player count is reached.
func FillSeats(room *runtime.RoomRuntime, target int) {
	for room.PlayerCount() < target {
		seat := uint32(room.PlayerCount())
		uid := UIDForSeat(seat)
		room.Seats[uid] = &runtime.Seat{
			UserID: uid, Seat: seat, Nickname: Nickname(seat),
			Ready: true, Online: true, Role: runtime.SeatRolePlayer,
		}
	}
}

// MarkBotsReady sets all bot seats to ready (used before round start).
func MarkBotsReady(room *runtime.RoomRuntime) {
	for _, s := range room.Seats {
		if IsBot(s.UserID) {
			s.Ready = true
		}
	}
}

func UserIDForSeat(room *runtime.RoomRuntime, seat uint32) (uint64, bool) {
	for uid, s := range room.Seats {
		if s.Seat == seat && s.Role == runtime.SeatRolePlayer {
			return uid, true
		}
	}
	return 0, false
}
