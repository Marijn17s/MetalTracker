package poller

import (
	"context"
	"log"
	"time"

	"MetalTracker/middleman/internal/service"
)

// hourSettle waits a few seconds after the hour so /hourly has the :00 bucket.
const hourSettle = 8 * time.Second

// RunHourly polls the current UTC hour on start, then again shortly after each :00.
func RunHourly(ctx context.Context, svc *service.Service, interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}
	runOnce := func() {
		if err := svc.PollLatest(ctx); err != nil {
			log.Printf("middleman poll: %v", err)
			return
		}
		log.Printf("middleman poll: stored hourly snapshot")
	}

	runOnce()

	for {
		now := time.Now().UTC()
		next := now.Truncate(interval).Add(interval).Add(hourSettle)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			runOnce()
		}
	}
}
