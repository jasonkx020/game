package opskick

import (
	"context"
	"encoding/json"

	goredis "github.com/redis/go-redis/v9"
)

const Channel = "ops:player_kick"

type Payload struct {
	PlayerID int64  `json:"player_id"`
	Reason   string `json:"reason"`
	AuditSN  uint64 `json:"audit_sn"`
}

// Publish asks the game process to kick a player (no-op if offline / rdb nil).
func Publish(ctx context.Context, rdb *goredis.Client, playerID int64, reason string, auditSN uint64) error {
	if rdb == nil || playerID <= 0 {
		return nil
	}
	b, err := json.Marshal(Payload{PlayerID: playerID, Reason: reason, AuditSN: auditSN})
	if err != nil {
		return err
	}
	return rdb.Publish(ctx, Channel, b).Err()
}
