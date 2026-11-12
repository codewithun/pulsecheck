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

type AgentTargetSummary struct {
	Total   int `json:"total"`
	UP      int `json:"up"`
	DOWN    int `json:"down"`
	Unknown int `json:"unknown"`
}

type AgentTargetsResponse struct {
	Data    []TargetItem `json:"data"`
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
	HasMore bool         `json:"has_more"`
}

type AgentDetailResponse struct {
	Agent   Agent                `json:"agent"`
	Summary AgentTargetSummary   `json:"summary"`
	Targets AgentTargetsResponse `json:"targets"`
}

func agentDetailHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		probeID := strings.TrimSpace(r.PathValue("probe_id"))
		if probeID == "" || len(probeID) > 100 {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "invalid probe_id",
			})
			return
		}

		query := r.URL.Query()

		limit := 20
		if raw := query.Get("limit"); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 1 || value > 100 {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "limit must be between 1 and 100",
				})
				return
			}
			limit = value
		}

		offset := 0
		if raw := query.Get("offset"); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 0 {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "offset must be a non-negative integer",
				})
				return
			}
			offset = value
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var agent Agent

		err := db.QueryRowContext(ctx, `
			SELECT
				probe_id,
				hostname,
				last_seen,
				CASE
					WHEN last_seen >= NOW() - INTERVAL '90 seconds'
					THEN 'ONLINE'
					ELSE 'OFFLINE'
				END
			FROM agents
			WHERE probe_id = $1
		`, probeID).Scan(
			&agent.ProbeID,
			&agent.Hostname,
			&agent.LastSeen,
			&agent.Status,
		)

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, ErrorResponse{
				Error: "agent not found",
			})
			return
		}
		if err != nil {
			log.Printf("Get agent detail: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve agent",
			})
			return
		}

		var summary AgentTargetSummary

		err = db.QueryRowContext(ctx, `
			WITH target_pairs AS (
				SELECT DISTINCT probe_id, target
				FROM check_results
				WHERE probe_id = $1
			),
			target_statuses AS (
				SELECT
					CASE
						WHEN ms.status = 'UP' THEN 'UP'
						WHEN ms.status = 'DOWN' THEN 'DOWN'
						ELSE 'UNKNOWN'
					END AS status
				FROM target_pairs tp
				LEFT JOIN monitoring_state ms
					ON ms.probe_id = tp.probe_id
					AND ms.target = tp.target
			)
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE status = 'UP'),
				COUNT(*) FILTER (WHERE status = 'DOWN'),
				COUNT(*) FILTER (WHERE status = 'UNKNOWN')
			FROM target_statuses
		`, probeID).Scan(
			&summary.Total,
			&summary.UP,
			&summary.DOWN,
			&summary.Unknown,
		)

		if err != nil {
			log.Printf("Get agent target summary: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve agent target summary",
			})
			return
		}

		rows, err := db.QueryContext(ctx, `
			WITH target_pairs AS (
				SELECT DISTINCT probe_id, target
				FROM check_results
				WHERE probe_id = $1
			),
			latest_checks AS (
				SELECT DISTINCT ON (probe_id, target)
					probe_id,
					target,
					status,
					checked_at,
					latency_ms,
					protocol,
					error
				FROM check_results
				WHERE probe_id = $1
				ORDER BY probe_id, target, checked_at DESC, id DESC
			)
			SELECT
				tp.probe_id,
				tp.target,
				CASE
					WHEN ms.status = 'UP' THEN 'UP'
					WHEN ms.status = 'DOWN' THEN 'DOWN'
					ELSE 'UNKNOWN'
				END AS operational_status,
				lc.status AS last_check_status,
				lc.checked_at,
				lc.latency_ms,
				lc.protocol,
				COALESCE(lc.error, '')
			FROM target_pairs tp
			LEFT JOIN monitoring_state ms
				ON ms.probe_id = tp.probe_id
				AND ms.target = tp.target
			LEFT JOIN latest_checks lc
				ON lc.probe_id = tp.probe_id
				AND lc.target = tp.target
			ORDER BY tp.target
			LIMIT $2 OFFSET $3
		`, probeID, limit+1, offset)

		if err != nil {
			log.Printf("Get agent targets: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve agent targets",
			})
			return
		}
		defer rows.Close()

		targets := make([]TargetItem, 0, limit+1)

		for rows.Next() {
			var item TargetItem

			if err := rows.Scan(
				&item.ProbeID,
				&item.Target,
				&item.Status,
				&item.LastCheckStatus,
				&item.LastCheckedAt,
				&item.LatencyMS,
				&item.Protocol,
				&item.Error,
			); err != nil {
				log.Printf("Scan agent target: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to process agent targets",
				})
				return
			}

			targets = append(targets, item)
		}

		if err := rows.Err(); err != nil {
			log.Printf("Iterate agent targets: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve agent targets",
			})
			return
		}

		hasMore := len(targets) > limit
		if hasMore {
			targets = targets[:limit]
		}

		writeJSON(w, http.StatusOK, AgentDetailResponse{
			Agent:   agent,
			Summary: summary,
			Targets: AgentTargetsResponse{
				Data:    targets,
				Limit:   limit,
				Offset:  offset,
				HasMore: hasMore,
			},
		})
	}
}
