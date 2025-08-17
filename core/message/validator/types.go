package validator

import (
	"time"
)

// ValidationResult contains the result of packet validation
type ValidationResult struct {
    IsValid         bool
    Error           error
    IsReplay        bool
    IsDuplicate     bool
    IsOutOfOrder    bool
    ProcessedSeq    uint64
}

// ValidatorConfig holds configuration for the message validator
type ValidatorConfig struct {
    NonceTTL            time.Duration // How long to remember nonces
    DuplicateTTL        time.Duration // How long to remember packet hashes
    TimestampTolerance  time.Duration // How much timestamp drift to allow
    MaxOutOfOrderWindow time.Duration // Maximum time window for out-of-order messages
    CleanupInterval     time.Duration // How often to run cleanup
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *ValidatorConfig {
    return &ValidatorConfig{
        NonceTTL:            5 * time.Minute,
        DuplicateTTL:        3 * time.Minute,
        TimestampTolerance:  30 * time.Second,
        MaxOutOfOrderWindow: 2 * time.Minute,
        CleanupInterval:     1 * time.Minute,
    }
}
