package discovery

import (
	"encoding/binary"
	"hash/fnv"
	"sort"
)

// Assign picks one connector for userID from live using consistent hashing.
// excludeID, when set, removes that connector before selection (reconnect escape).
func Assign(userID int64, live []Connector, excludeID string) *Connector {
	candidates := make([]Connector, 0, len(live))
	for _, c := range live {
		if c.ID == "" || c.WSURL == "" {
			continue
		}
		if excludeID != "" && c.ID == excludeID {
			continue
		}
		candidates = append(candidates, c)
	}
	n := len(candidates)
	if n == 0 {
		return nil
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	h := fnv.New64a()
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(userID))
	_, _ = h.Write(buf[:])
	idx := int(h.Sum64() % uint64(n))
	c := candidates[idx]
	return &c
}
