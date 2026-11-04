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

type TargetHistoryItem struct {
	ID         int64     `json:"id"`
	ProbeID    string    `json:"probe_id"`
	Target     string    `json:"target"`
	Status     string    `json:"status"`
	StatusCode *int      `json:"status_code,omitempty"`
	LatencyMS  float64   `json:"latency_ms"`
	Error      string    `json:"error,omitempty"`
	Protocol   string    `json:"protocol"`
	CheckedAt  time.Time `json:"checked_at"`
	ReceivedAt time.Time `json:"received_at"`
}

type TargetHistoryResponse struct {
	Data    []TargetHistoryItem `json:"data"`
	Limit   int                 `json:"limit"`
	Offset  int                 `json:"offset"`
	HasMore bool                `json:"has_more"`
}

func targetHistoryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		probeID := strings.TrimSpace(query.Get("probe_id"))
		target := strings.TrimSpace(query.Get("target"))

		if probeID == "" || target == "" {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "probe_id and target are required",
			})
			return
		}

		if len(probeID) > 100 {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "probe_id must not exceed 100 characters",
			})
			return
		}

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

		var periodSeconds int64

		switch strings.TrimSpace(query.Get("period")) {
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
			SELECT
				id,
				probe_id,
				target,
				status,
				status_code,
				latency_ms,
				error,
				protocol,
				checked_at,
				received_at
			FROM check_results
			WHERE probe_id = $1
			  AND target = $2
			  AND (
				$3 = 0
				OR checked_at >= NOW() - ($3 * INTERVAL '1 second')
			  )
			ORDER BY checked_at DESC, id DESC
			LIMIT $4 OFFSET $5
		`, probeID, target, periodSeconds, limit+1, offset)
		if err != nil {
			log.Printf("Query target history: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve target history",
			})
			return
		}
		defer rows.Close()

		items := make([]TargetHistoryItem, 0, limit+1)

		for rows.Next() {
			var item TargetHistoryItem
			var statusCode sql.NullInt64
			var errorMessage sql.NullString

			if err := rows.Scan(
				&item.ID,
				&item.ProbeID,
				&item.Target,
				&item.Status,
				&statusCode,
				&item.LatencyMS,
				&errorMessage,
				&item.Protocol,
				&item.CheckedAt,
				&item.ReceivedAt,
			); err != nil {
				log.Printf("Scan target history: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to retrieve target history",
				})
				return
			}

			if statusCode.Valid {
				code := int(statusCode.Int64)
				item.StatusCode = &code
			}

			if errorMessage.Valid {
				item.Error = errorMessage.String
			}

			items = append(items, item)
		}

		if err := rows.Err(); err != nil {
			log.Printf("Iterate target history: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve target history",
			})
			return
		}

		hasMore := len(items) > limit
		if hasMore {
			items = items[:limit]
		}

		writeJSON(w, http.StatusOK, TargetHistoryResponse{
			Data:    items,
			Limit:   limit,
			Offset:  offset,
			HasMore: hasMore,
		})
	}
}
