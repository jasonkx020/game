package discovery

import (
	"context"
	"log/slog"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// PublisherConfig configures connector self-registration into Redis.
type PublisherConfig struct {
	Enabled     bool
	InstanceID  string
	WSURL       string
	WSDirectURL string
	Version     string
	TTL         time.Duration
	Tick        time.Duration
}

// Publisher registers this process as a connector and refreshes until Stop.
type Publisher struct {
	rdb *goredis.Client
	cfg PublisherConfig

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewPublisher(rdb *goredis.Client, cfg PublisherConfig) *Publisher {
	if cfg.TTL <= 0 {
		cfg.TTL = DefaultTTL
	}
	if cfg.Tick <= 0 {
		cfg.Tick = DefaultTick
	}
	return &Publisher{rdb: rdb, cfg: cfg}
}

// Start writes immediately then refreshes on Tick until Stop or ctx cancel.
func (p *Publisher) Start(parent context.Context) error {
	if p == nil || !p.cfg.Enabled || p.rdb == nil {
		return nil
	}
	if p.cfg.InstanceID == "" || p.cfg.WSURL == "" {
		slog.Warn("discovery publisher skipped: missing instance id or ws url")
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	p.cancel = cancel

	if err := p.write(ctx); err != nil {
		cancel()
		return err
	}
	slog.Info("discovery registered", "id", p.cfg.InstanceID, "ws_url", p.cfg.WSURL)

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		t := time.NewTicker(p.cfg.Tick)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := p.write(ctx); err != nil {
					slog.Warn("discovery refresh", "err", err)
				}
			}
		}
	}()
	return nil
}

func (p *Publisher) write(ctx context.Context) error {
	c := Connector{
		ID:          p.cfg.InstanceID,
		WSURL:       p.cfg.WSURL,
		WSDirectURL: p.cfg.WSDirectURL,
		Version:     p.cfg.Version,
		UpdatedAt:   time.Now().UTC(),
	}
	return upsertRedis(ctx, p.rdb, c, p.cfg.TTL)
}

// Stop cancels refresh and deletes this connector key.
func (p *Publisher) Stop(ctx context.Context) {
	if p == nil {
		return
	}
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	if !p.cfg.Enabled || p.rdb == nil || p.cfg.InstanceID == "" {
		return
	}
	if err := deleteRedis(ctx, p.rdb, p.cfg.InstanceID); err != nil {
		slog.Warn("discovery unregister", "err", err)
		return
	}
	slog.Info("discovery unregistered", "id", p.cfg.InstanceID)
}
