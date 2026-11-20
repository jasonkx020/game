package discovery

import "time"

const (
	KeyPrefix    = "discovery:connector:"
	EventChannel = "discovery:connector:events"
	DefaultTTL   = 3 * time.Second
	DefaultTick  = 1 * time.Second
)

// Connector is one live WS endpoint (full set stays server-side only).
type Connector struct {
	ID          string    `json:"id"`
	WSURL       string    `json:"ws_url"`
	WSDirectURL string    `json:"ws_direct_url,omitempty"`
	Load        int       `json:"load"`
	Version     string    `json:"version,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Event is published on register / refresh / unregister.
type Event struct {
	Op        string     `json:"op"` // upsert | delete
	ID        string     `json:"id"`
	Connector *Connector `json:"connector,omitempty"`
}

// AssignResult is what a player may see — never a full connector list.
type AssignResult struct {
	WSGatewayURL   string     `json:"ws_gateway_url,omitempty"`
	HTTPGatewayURL string     `json:"http_gateway_url,omitempty"`
	Connector      *Connector `json:"connector,omitempty"`
}
