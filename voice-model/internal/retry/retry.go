package retry

import (
	"context"
	"fmt"
	"math"
	"time"
)

type Config struct {
	MaxAttempts int
	BaseDelay   time.Duration
}

func DefaultConfig() Config {
	return Config{
		MaxAttempts: 3,
		BaseDelay:   time.Second,
	}
}

func Do(ctx context.Context, cfg Config, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("retry cancelled: %w", err)
		}
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if attempt < cfg.MaxAttempts-1 {
			delay := cfg.BaseDelay * time.Duration(math.Pow(2, float64(attempt)))
			select {
			case <-ctx.Done():
				return fmt.Errorf("retry cancelled: %w", ctx.Err())
			case <-time.After(delay):
			}
		}
	}
	return fmt.Errorf("after %d attempts: %w", cfg.MaxAttempts, lastErr)
}
