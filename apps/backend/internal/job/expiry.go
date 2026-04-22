package job

import (
	"context"
	"log/slog"
	"time"
)

type reservationExpirer interface {
	CompleteExpired(ctx context.Context, now time.Time) (int64, error)
}

// ExpiryJob periodically completes reservations whose end time has passed.
type ExpiryJob struct {
	reservations reservationExpirer
	interval     time.Duration
}

// NewExpiryJob creates an expiry job that runs on the given interval.
func NewExpiryJob(reservations reservationExpirer, interval time.Duration) *ExpiryJob {
	return &ExpiryJob{reservations: reservations, interval: interval}
}

// Run starts the job and blocks until ctx is cancelled.
func (job *ExpiryJob) Run(ctx context.Context) {
	ticker := time.NewTicker(job.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			count, err := job.reservations.CompleteExpired(ctx, t)
			if err != nil {
				slog.Error("expiry job failed", "err", err)
				continue
			}
			if count > 0 {
				slog.Info("completed expired reservations", "count", count)
			}
		}
	}
}
