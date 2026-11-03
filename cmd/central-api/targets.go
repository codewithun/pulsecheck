package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type TargetItem struct {
	ProbeID         string    `json:"probe_id"`
	Target          string    `json:"target"`
	Status          string    `json:"status"`
	LastCheckStatus string    `json:"last_check_status"`
	LastCheckedAt   time.Time `json:"last_checked_at"`
	LatencyMS       float64   `json:"latency_ms"`
	Protocol        string    `json:"protocol"`
	Error           string    `json:"error,omitempty"`
}

type TargetsResponse struct {
	Data    []TargetItem `json:"data"`
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
	HasMore bool         `json:"has_more"`
}

func targetsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		// Pagination
		limit := 50
		offset := 0

		if value := query.Get("limit"); value != "" {
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed < 1 || parsed > 100 {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "limit must be between 1 and 100",
				})
				return
			}
			limit = parsed
		}

		if value := query.Get("offset"); value != "" {
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed < 0 {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "offset must be a non-negative integer",
				})
				return
			}
			offset = parsed
		}

		// Filters
		probeID := strings.TrimSpace(query.Get("probe_id"))
		target := strings.TrimSpace(query.Get("target"))
		status := strings.ToUpper(strings.TrimSpace(query.Get("status")))
		period := strings.TrimSpace(query.Get("period"))

		// Validate operational status
		switch status {
		case "", "UP", "DOWN", "UNKNOWN":
		default:
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "status must be UP, DOWN, or UNKNOWN",
			})
			return
		}

		// Validate period
		var periodSeconds int64

		switch period {
		case "":
			periodSeconds = 0
		case "1h":
			periodSeconds = 60 * 60
		case "24h":
			periodSeconds = 24 * 60 * 60
		case "7d":
			periodSeconds = 7 * 24 * 60 * 60
		case "30d":
			periodSeconds = 30 * 24 * 60 * 60
		default:
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "period must be 1h, 24h, 7d, or 30d",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		rows, err := db.QueryContext(ctx, `
			WITH target_pairs AS (
				SELECT DISTINCT
					cr.probe_id,
					cr.target
				FROM check_results cr
				WHERE (
					$4 = 0
					OR EXISTS (
						SELECT 1
						FROM check_results recent
						WHERE recent.probe_id = cr.probe_id
						  AND recent.target = cr.target
						  AND recent.checked_at >= NOW() - ($4 * INTERVAL '1 second')
					)
				)
			),
			latest_checks AS (
				SELECT DISTINCT ON (probe_id, target)
					probe_id,
					target,
					status,
					latency_ms,
					protocol,
					error,
					checked_at
				FROM check_results
				ORDER BY
					probe_id,
					target,
					checked_at DESC,
					id DESC
			),
			target_data AS (
				SELECT
					tp.probe_id,
					tp.target,
					COALESCE(ms.status, 'UNKNOWN') AS status,
					lc.status AS last_check_status,
					lc.checked_at AS last_checked_at,
					lc.latency_ms,
					lc.protocol,
					lc.error
				FROM target_pairs tp
				JOIN latest_checks lc
					ON lc.probe_id = tp.probe_id
					AND lc.target = tp.target
				LEFT JOIN monitoring_state ms
					ON ms.probe_id = tp.probe_id
					AND ms.target = tp.target
			)
			SELECT
				probe_id,
				target,
				status,
				last_check_status,
				last_checked_at,
				latency_ms,
				protocol,
				error
			FROM target_data
			WHERE ($1 = '' OR probe_id = $1)
			  AND ($2 = '' OR target ILIKE '%' || $2 || '%')
			  AND ($3 = '' OR status = $3)
			ORDER BY
				last_checked_at DESC,
				probe_id,
				target
			LIMIT $5
			OFFSET $6
		`,
			probeID,
			target,
			status,
			periodSeconds,
			limit+1,
			offset,
		)
		if err != nil {
			log.Printf("failed to query targets: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve targets",
			})
			return
		}
		defer rows.Close()

		items := make([]TargetItem, 0, limit+1)

		for rows.Next() {
			var item TargetItem
			var errorMessage sql.NullString

			err := rows.Scan(
				&item.ProbeID,
				&item.Target,
				&item.Status,
				&item.LastCheckStatus,
				&item.LastCheckedAt,
				&item.LatencyMS,
				&item.Protocol,
				&errorMessage,
			)
			if err != nil {
				log.Printf("failed to scan target row: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to retrieve targets",
				})
				return
			}

			if errorMessage.Valid {
				item.Error = errorMessage.String
			}

			items = append(items, item)
		}

		if err := rows.Err(); err != nil {
			log.Printf("failed while reading target rows: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve targets",
			})
			return
		}

		hasMore := len(items) > limit

		if hasMore {
			items = items[:limit]
		}

		response := TargetsResponse{
			Data:    items,
			Limit:   limit,
			Offset:  offset,
			HasMore: hasMore,
		}

		writeJSON(w, http.StatusOK, response)
	}
}
