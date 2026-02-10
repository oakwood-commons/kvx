package ui

import "github.com/oakwood-commons/kvx/internal/limiter"

//nolint:unused // Variable is set by SetLimiterConfig but may be used in future versions.
var activeLimiterConfig limiter.Config

// SetLimiterConfig stores the limiter configuration for subsequent TUI sessions.
func SetLimiterConfig(cfg limiter.Config) {
	activeLimiterConfig = cfg
}
