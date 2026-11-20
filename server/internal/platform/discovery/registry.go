package discovery

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	goredis "github.com/redis/go-redis/v9"
)

// RegistryConfig holds gateway URLs returned with assign (not the full node list).
type RegistryConfig struct {
	WSGatewayURL   string
	HTTPGatewayURL string
}

// Registry keeps an in-memory live set fed by Redis load + Pub/Sub.
type Registry struct {
	rdb *goredis.Client
	cfg RegistryConfig

	mu         sync.RWMutex
	connectors map[string]Connector
}

func NewRegistry(rdb *goredis.Client, cfg RegistryConfig) *Registry {
	return &Registry{
		rdb:        rdb,
		cfg:        cfg,
		connectors: make(map[string]Connector),
	}
}

// Start loads Redis snapshot then subscribes until ctx is done.
func (r *Registry) Start(ctx context.Context) {
	if r == nil {
		return
	}
	if list, err := loadAllRedis(ctx, r.rdb); err != nil {
		slog.Warn("discovery load", "err", err)
	} else {
		r.mu.Lock()
		for _, c := range list {
			r.connectors[c.ID] = c
		}
		r.mu.Unlock()
		slog.Info("discovery loaded", "count", len(list))
	}
	go r.subscribe(ctx)
}

func (r *Registry) subscribe(ctx context.Context) {
	if r.rdb == nil {
		return
	}
	sub := r.rdb.Subscribe(ctx, EventChannel)
	defer sub.Close()
	ch := sub.Channel()
	slog.Info("discovery subscribed", "channel", EventChannel)
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var ev Event
			if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
				slog.Warn("discovery bad event", "raw", msg.Payload, "err", err)
				continue
			}
			r.applyEvent(ev)
		}
	}
}

func (r *Registry) applyEvent(ev Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch ev.Op {
	case "upsert":
		if ev.Connector != nil && ev.Connector.ID != "" {
			r.connectors[ev.Connector.ID] = *ev.Connector
		} else if ev.ID != "" && ev.Connector != nil {
			r.connectors[ev.ID] = *ev.Connector
		}
	case "delete":
		id := ev.ID
		if id == "" && ev.Connector != nil {
			id = ev.Connector.ID
		}
		if id != "" {
			delete(r.connectors, id)
		}
	}
}

func (r *Registry) snapshot() []Connector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Connector, 0, len(r.connectors))
	for _, c := range r.connectors {
		out = append(out, c)
	}
	return out
}

// AssignForUser returns gateway URLs plus at most one connector.
func (r *Registry) AssignForUser(userID int64, excludeID string) AssignResult {
	res := AssignResult{
		WSGatewayURL:   r.cfg.WSGatewayURL,
		HTTPGatewayURL: r.cfg.HTTPGatewayURL,
	}
	if r == nil {
		return res
	}
	res.Connector = Assign(userID, r.snapshot(), excludeID)
	return res
}
