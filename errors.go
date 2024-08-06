package pool

import "errors"

var (
	ErrNoConnectFunc          = errors.New("no connect function specified")
	ErrNoHealthyFunc          = errors.New("no health function specified")
	ErrInvalidConnectionCount = errors.New("invalid connection count")
	ErrInvalidProbeInterval   = errors.New("invalid probe interval")
)
