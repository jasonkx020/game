package audit

import (
	"sync"
	"time"
)

// Snowflake: 41-bit timestamp | 10-bit worker | 12-bit sequence
type Generator struct {
	mu       sync.Mutex
	workerID int64
	sequence int64
	lastMs   int64
}

func NewGenerator(workerID int64) *Generator {
	return &Generator{workerID: workerID & 0x3FF}
}

func (g *Generator) Next() uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli()
	if now == g.lastMs {
		g.sequence = (g.sequence + 1) & 0xFFF
		if g.sequence == 0 {
			for now <= g.lastMs {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		g.sequence = 0
	}
	g.lastMs = now

	id := (uint64(now) << 22) | (uint64(g.workerID) << 12) | uint64(g.sequence)
	return id
}
