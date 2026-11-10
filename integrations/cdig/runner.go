package cdig

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"
)

// Start does nothing unless configuration is complete AND scheduling is opted in.
// Failures stay in this goroutine; ordinary platform requests do not call CDIG.
func Start(ctx context.Context, db *gorm.DB) {
	c, reason := Load()
	if !c.Enabled {
		if reason != "disabled" {
			log.Printf("CDIG sync disabled: %s", reason)
		}
		return
	}
	if c.Interval == 0 {
		return
	}
	go func() {
		timer := time.NewTicker(c.Interval)
		defer timer.Stop()
		// Deliberately wait until the first interval; rollout never writes on startup.
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
				report, err := (&Sync{Config: c, Client: NewClient(c), DB: db}).Run(runCtx, false)
				cancel()
				if err != nil {
					log.Print("CDIG sync failed; platform remains available")
				} else {
					log.Printf("CDIG sync completed: %v", report.Counts)
				}
			}
		}
	}()
}
