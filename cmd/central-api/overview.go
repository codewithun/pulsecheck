package main

import (
	"context"
	"database/sql"
	"log"
	"math"
	"net/http"
	"time"
)

type OverviewAgentStats struct {
	Total   int `json:"total"`
	Online  int `json:"online"`
	Offline int `json:"offline"`
}

type OverviewTargetStats struct {
	Total   int `json:"total"`
	UP      int `json:"up"`
	DOWN    int `json:"down"`
	UNKNOWN int `json:"unknown"`
}

type OverviewCheckStats struct {
	Total   int `json:"total"`
	UP      int `json:"up"`
	DOWN    int `json:"down"`
	INVALID int `json:"invalid"`
}

type OverviewAlertStats struct {
	Pending   int `json:"pending"`
	Delivered int `json:"delivered"`
}

type OverviewResponse struct {
	Period       string              `json:"period"`
	Agents       OverviewAgentStats  `json:"agents"`
	Targets      OverviewTargetStats `json:"targets"`
	Checks       OverviewCheckStats  `json:"checks"`
	Availability *float64            `json:"availability"`
	Alerts       OverviewAlertStats  `json:"alerts"`
}

func overviewHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		period := r.URL.Query().Get("period")
		if period == "" {
			period = "24h"
		}

		periodSeconds := map[string]int64{
			"1h":  60 * 60,
			"24h": 24 * 60 * 60,
			"7d":  7 * 24 * 60 * 60,
			"30d": 30 * 24 * 60 * 60,
		}

		seconds, valid := periodSeconds[period]
		if !valid {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "invalid period; allowed values: 1h, 24h, 7d, 30d",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		response := OverviewResponse{
			Period: period,
		}

		err := db.QueryRowContext(ctx, `
			SELECT
				COUNT(*),
				COUNT(*) FILTER (
					WHERE last_seen >= NOW() - INTERVAL '90 seconds'
				),
				COUNT(*) FILTER (
					WHERE last_seen < NOW() - INTERVAL '90 seconds'
				)
			FROM agents
		`).Scan(
			&response.Agents.Total,
			&response.Agents.Online,
			&response.Agents.Offline,
		)
		if err != nil {
			log.Printf("Get overview agent stats: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve overview",
			})
			return
		}

		err = db.QueryRowContext(ctx, `
			WITH target_pairs AS (
				SELECT DISTINCT probe_id, target
				FROM check_results
			)
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE ms.status = 'UP'),
				COUNT(*) FILTER (WHERE ms.status = 'DOWN'),
				COUNT(*) FILTER (WHERE ms.status IS NULL)
			FROM target_pairs tp
			LEFT JOIN monitoring_state ms
				ON ms.probe_id = tp.probe_id
				AND ms.target = tp.target
		`).Scan(
			&response.Targets.Total,
			&response.Targets.UP,
			&response.Targets.DOWN,
			&response.Targets.UNKNOWN,
		)
		if err != nil {
			log.Printf("Get overview target stats: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve overview",
			})
			return
		}

		err = db.QueryRowContext(ctx, `
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE status = 'UP'),
				COUNT(*) FILTER (WHERE status = 'DOWN'),
				COUNT(*) FILTER (WHERE status = 'INVALID')
			FROM check_results
			WHERE checked_at >= NOW() - ($1 * INTERVAL '1 second')
		`, seconds).Scan(
			&response.Checks.Total,
			&response.Checks.UP,
			&response.Checks.DOWN,
			&response.Checks.INVALID,
		)
		if err != nil {
			log.Printf("Get overview check stats: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve overview",
			})
			return
		}

		validChecks := response.Checks.UP + response.Checks.DOWN
		if validChecks > 0 {
			availability := math.Round(
				float64(response.Checks.UP)/float64(validChecks)*10000,
			) / 100
			response.Availability = &availability
		}

		err = db.QueryRowContext(ctx, `
			SELECT
				COUNT(*) FILTER (WHERE delivered_at IS NULL),
				COUNT(*) FILTER (
					WHERE delivered_at >= NOW() - ($1 * INTERVAL '1 second')
				)
			FROM alert_outbox
		`, seconds).Scan(
			&response.Alerts.Pending,
			&response.Alerts.Delivered,
		)
		if err != nil {
			log.Printf("Get overview alert stats: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve overview",
			})
			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}
