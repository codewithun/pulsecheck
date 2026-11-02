package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type AgentHeartbeatRequest struct {
	ProbeID  string `json:"probe_id"`
	Hostname string `json:"hostname"`
}

type AgentHeartbeatResponse struct {
	Status   string    `json:"status"`
	ProbeID  string    `json:"probe_id"`
	Hostname string    `json:"hostname"`
	LastSeen time.Time `json:"last_seen"`
}

func agentHeartbeatHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var request AgentHeartbeatRequest

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "invalid JSON payload",
			})
			return
		}

		var extra any
		if err := decoder.Decode(&extra); err != nil {
			if !errors.Is(err, io.EOF) {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "invalid trailing JSON",
				})
				return
			}
		} else {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "request must contain one JSON object",
			})
			return
		}

		request.ProbeID = strings.TrimSpace(request.ProbeID)
		request.Hostname = strings.TrimSpace(request.Hostname)

		if request.ProbeID == "" || len(request.ProbeID) > 100 {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "probe_id is required and must not exceed 100 characters",
			})
			return
		}

		if request.Hostname == "" || len(request.Hostname) > 255 {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "hostname is required and must not exceed 255 characters",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var lastSeen time.Time

		err := db.QueryRowContext(ctx, `
			INSERT INTO agents (probe_id, hostname, last_seen, updated_at)
			VALUES ($1, $2, NOW(), NOW())
			ON CONFLICT (probe_id)
			DO UPDATE SET
				hostname = EXCLUDED.hostname,
				last_seen = NOW(),
				updated_at = NOW()
			RETURNING last_seen
		`, request.ProbeID, request.Hostname).Scan(&lastSeen)

		if err != nil {
			log.Printf("Update agent heartbeat: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to update agent heartbeat",
			})
			return
		}

		writeJSON(w, http.StatusOK, AgentHeartbeatResponse{
			Status:   "received",
			ProbeID:  request.ProbeID,
			Hostname: request.Hostname,
			LastSeen: lastSeen,
		})
	}
}

type Agent struct {
	ProbeID  string    `json:"probe_id"`
	Hostname string    `json:"hostname"`
	LastSeen time.Time `json:"last_seen"`
	Status   string    `json:"status"`
}

type AgentListResponse struct {
	Data []Agent `json:"data"`
}

func agentsListHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		rows, err := db.QueryContext(ctx, `
			SELECT
				probe_id,
				hostname,
				last_seen,
				CASE
					WHEN last_seen >= NOW() - INTERVAL '90 seconds'
					THEN 'ONLINE'
					ELSE 'OFFLINE'
				END AS status
			FROM agents
			ORDER BY last_seen DESC
		`)
		if err != nil {
			log.Printf("Get agents: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve agents",
			})
			return
		}
		defer rows.Close()

		agents := make([]Agent, 0)

		for rows.Next() {
			var agent Agent

			if err := rows.Scan(
				&agent.ProbeID,
				&agent.Hostname,
				&agent.LastSeen,
				&agent.Status,
			); err != nil {
				log.Printf("Scan agent: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to process agents",
				})
				return
			}

			agents = append(agents, agent)
		}

		if err := rows.Err(); err != nil {
			log.Printf("Iterate agents: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve agents",
			})
			return
		}

		writeJSON(w, http.StatusOK, AgentListResponse{
			Data: agents,
		})
	}
}
