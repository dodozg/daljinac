package agent

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"
)

// failureLogEvery limits how often an ongoing outage is logged. Without it a
// server outage would write a warning every minute into each machine's
// event log.
const failureLogEvery = time.Hour

// Loop calls report every interval until ctx is cancelled. The first call
// comes after a random delay of up to one interval, so 500 agents do not
// all report in the same second after a server restart or a power cut.
//
// Failures are logged when they start, at most once per failureLogEvery
// while they last, and once more when reports get through again.
func Loop(ctx context.Context, log *slog.Logger, interval time.Duration, report func(context.Context) error) {
	wait := time.Duration(rand.Int64N(int64(interval)))
	var (
		failures   int
		lastLogged time.Time
	)
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = interval

		err := report(ctx)
		if ctx.Err() != nil {
			return // stopping; a cancelled request is not a failure
		}
		switch {
		case err == nil && failures > 0:
			log.Info("reports are getting through again", "failed_attempts", failures)
			failures = 0
		case err != nil:
			failures++
			if failures == 1 || time.Since(lastLogged) >= failureLogEvery {
				log.Warn("report failed", "err", err, "failed_attempts", failures)
				lastLogged = time.Now()
			}
		}
	}
}
