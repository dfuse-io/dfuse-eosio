package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/dfuse-io/dauth/authenticator"
	"github.com/dfuse-io/dauth/ratelimiter"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

func init() {
	services := []string{"search", "block", "blockmeta", "token"}
	ratelimiter.RegisterServices(services)
}

func (r *Root) RateLimit(ctx context.Context, method string) error {
	if r.requestRateLimiter == nil {
		return nil
	}

	zlogger := logging.Logger(ctx, zlog)

	creds := authenticator.GetCredentials(ctx)
	userID := creds.GetUserID()

	if !r.requestRateLimiter.Gate(userID, method) {
		if time.Since(r.requestRateLimiterLastLogTime) > 500*time.Millisecond {
			zlogger.Info("rate limited user",
				zap.String("sampling_frequency", "500ms"),
				zap.String("user_id", userID),
				zap.String("method", method),
			)
			r.requestRateLimiterLastLogTime = time.Now()
		}
		return fmt.Errorf("rate limited for method %s", method)
	}
	return nil
}
