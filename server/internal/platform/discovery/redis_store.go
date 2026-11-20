package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func connectorKey(id string) string {
	return KeyPrefix + id
}

func upsertRedis(ctx context.Context, rdb *goredis.Client, c Connector, ttl time.Duration) error {
	if rdb == nil {
		return fmt.Errorf("redis nil")
	}
	if c.ID == "" || c.WSURL == "" {
		return fmt.Errorf("connector id and ws_url required")
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = time.Now().UTC()
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	pipe := rdb.Pipeline()
	pipe.Set(ctx, connectorKey(c.ID), raw, ttl)
	ev, _ := json.Marshal(Event{Op: "upsert", ID: c.ID, Connector: &c})
	pipe.Publish(ctx, EventChannel, ev)
	_, err = pipe.Exec(ctx)
	return err
}

func deleteRedis(ctx context.Context, rdb *goredis.Client, id string) error {
	if rdb == nil || id == "" {
		return nil
	}
	pipe := rdb.Pipeline()
	pipe.Del(ctx, connectorKey(id))
	ev, _ := json.Marshal(Event{Op: "delete", ID: id})
	pipe.Publish(ctx, EventChannel, ev)
	_, err := pipe.Exec(ctx)
	return err
}

func loadAllRedis(ctx context.Context, rdb *goredis.Client) ([]Connector, error) {
	if rdb == nil {
		return nil, nil
	}
	var (
		cursor uint64
		out    []Connector
	)
	for {
		keys, next, err := rdb.Scan(ctx, cursor, KeyPrefix+"*", 100).Result()
		if err != nil {
			return out, err
		}
		for _, key := range keys {
			raw, err := rdb.Get(ctx, key).Bytes()
			if err == goredis.Nil {
				continue
			}
			if err != nil {
				return out, err
			}
			var c Connector
			if err := json.Unmarshal(raw, &c); err != nil || c.ID == "" {
				continue
			}
			out = append(out, c)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return out, nil
}
