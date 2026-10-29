package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"
)

type alertEvent struct {
	ID             int64
	ProbeID        string
	Target         string
	PreviousStatus string
	CurrentStatus  string
	Error          string
	Attempts       int
}

func claimAlert(ctx context.Context, db *sql.DB) (*alertEvent, error) {
	var event alertEvent

	err := db.QueryRowContext(ctx, `
		UPDATE alert_outbox
		SET claimed_until = NOW() + INTERVAL '30 seconds'
		WHERE id = (
			SELECT id
			FROM alert_outbox
			WHERE delivered_at IS NULL
			  AND next_attempt_at <= NOW()
			  AND (claimed_until IS NULL OR claimed_until < NOW())
			ORDER BY next_attempt_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, probe_id, target, previous_status,
		          current_status, COALESCE(error, ''), attempts
	`).Scan(
		&event.ID,
		&event.ProbeID,
		&event.Target,
		&event.PreviousStatus,
		&event.CurrentStatus,
		&event.Error,
		&event.Attempts,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim alert: %w", err)
	}

	return &event, nil
}

func retryDelay(attempts int) time.Duration {
	seconds := math.Pow(2, float64(attempts))
	if seconds > 900 {
		seconds = 900
	}
	return time.Duration(seconds) * time.Second
}

func markAlertDelivered(ctx context.Context, db *sql.DB, id int64) error {
	_, err := db.ExecContext(ctx, `
		UPDATE alert_outbox
		SET delivered_at = NOW(),
		    claimed_until = NULL,
		    attempts = attempts + 1
		WHERE id = $1 AND delivered_at IS NULL
	`, id)
	return err
}

func rescheduleAlert(ctx context.Context, db *sql.DB, event *alertEvent) error {
	delay := retryDelay(event.Attempts + 1)

	_, err := db.ExecContext(ctx, `
		UPDATE alert_outbox
		SET attempts = attempts + 1,
		    claimed_until = NULL,
		    next_attempt_at = NOW() + ($2 * INTERVAL '1 second')
		WHERE id = $1 AND delivered_at IS NULL
	`, event.ID, delay.Seconds())
	return err
}

func runAlertWorker(ctx context.Context, db *sql.DB, sender *discordSender) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Println("Discord alert worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Discord alert worker stopped")
			return
		case <-ticker.C:
			processPendingAlerts(ctx, db, sender)
		}
	}
}

func processPendingAlerts(ctx context.Context, db *sql.DB, sender *discordSender) {
	for i := 0; i < 10; i++ {
		if ctx.Err() != nil {
			return
		}

		workCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		event, err := claimAlert(workCtx, db)
		cancel()

		if err != nil {
			log.Printf("Could not claim Discord alert: %v", err)
			return
		}
		if event == nil {
			return
		}

		sendCtx, sendCancel := context.WithTimeout(ctx, 6*time.Second)
		err = sender.send(sendCtx, *event)
		sendCancel()

		updateCtx, updateCancel := context.WithTimeout(ctx, 3*time.Second)
		if err == nil {
			if updateErr := markAlertDelivered(updateCtx, db, event.ID); updateErr != nil {
				log.Printf("Could not mark alert %d delivered: %v", event.ID, updateErr)
			}
		} else {
			log.Printf("Discord alert %d failed: %v", event.ID, err)
			if updateErr := rescheduleAlert(updateCtx, db, event); updateErr != nil {
				log.Printf("Could not reschedule alert %d: %v", event.ID, updateErr)
			}
		}
		updateCancel()
	}
}
