package risk

import "errors"

var (
	ErrDeviceLimit   = errors.New("device account limit exceeded")
	ErrSameIP        = errors.New("same IP seat limit exceeded")
	ErrGPSNear       = errors.New("gps proximity blocked")
	ErrCurfew        = errors.New("minor curfew")
	ErrPlaytime      = errors.New("minor daily playtime exceeded")
	ErrFrozen        = errors.New("player risk frozen")
	ErrMinorRecharge = errors.New("minors cannot recharge")
	ErrNotFound      = errors.New("not found")
	ErrBadRequest    = errors.New("bad request")
	ErrForbidden     = errors.New("forbidden")
)

// Pitaya / HTTP stable codes
const (
	CodeDevice  = "RISK_DEVICE"
	CodeIP      = "RISK_IP"
	CodeGPS     = "RISK_GPS"
	CodeCurfew  = "RISK_CURFEW"
	CodePlaytime = "RISK_PLAYTIME"
	CodeFrozen  = "RISK_FROZEN"
)
