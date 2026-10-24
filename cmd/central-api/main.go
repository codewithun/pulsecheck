package main

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type CheckResult struct {
	ProbeID    string    `json:"probe_id"`
	Target     string    `json:"target"`
	Status     string    `json:"status"`
	StatusCode *int      `json:"status_code,omitempty"`
	LatencyMS  float64   `json:"latency_ms"`
	Error      string    `json:"error,omitempty"`
	Protocol   string    `json:"protocol"`
	CheckedAt  time.Time `json:"checked_at"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ResultsResponse struct {
	Data    []CheckResult `json:"data"`
	Limit   int           `json:"limit"`
	Offset  int           `json:"offset"`
	HasMore bool          `json:"has_more"`
}

func main() {
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	apiToken := os.Getenv("API_TOKEN")
	if apiToken == "" {
		log.Fatal("API_TOKEN is required")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatal("Open database:", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(ctx)
	cancel()
	if err != nil {
		log.Fatal("Connect to database:", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{
				Error: "database unavailable",
			})
			return
		}

		writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
	})

	mux.Handle("POST /api/v1/results", authenticate(apiToken, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

			var result CheckResult
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()

			if err := decoder.Decode(&result); err != nil {
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

			result.ProbeID = strings.TrimSpace(result.ProbeID)
			result.Target = strings.TrimSpace(result.Target)
			result.Protocol = strings.ToUpper(strings.TrimSpace(result.Protocol))

			if result.ProbeID == "" || len(result.ProbeID) > 100 ||
				result.Target == "" {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "probe_id and target are required",
				})
				return
			}

			if result.Status != "UP" &&
				result.Status != "DOWN" &&
				result.Status != "INVALID" {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "status must be UP, DOWN, or INVALID",
				})
				return
			}

			if result.Protocol != "HTTP" && result.Protocol != "TCP" {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "protocol must be HTTP or TCP",
				})
				return
			}

			if result.LatencyMS < 0 ||
				result.LatencyMS > 24*60*60*1000 {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "latency_ms is out of range",
				})
				return
			}

			if result.CheckedAt.IsZero() {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "checked_at is required",
				})
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			_, err := db.ExecContext(ctx, `
				INSERT INTO check_results
					(probe_id, target, status, status_code,
					 latency_ms, error, protocol, checked_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`,
				result.ProbeID,
				result.Target,
				result.Status,
				result.StatusCode,
				result.LatencyMS,
				nullIfEmpty(result.Error),
				result.Protocol,
				result.CheckedAt,
			)
			if err != nil {
				log.Printf("Insert check result: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to save result",
				})
				return
			}

			writeJSON(w, http.StatusCreated, map[string]string{
				"status": "saved",
			})
		},
	)))

	mux.Handle("GET /api/v1/results", authenticate(apiToken, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()

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
						Error: "offset must be zero or greater",
					})
					return
				}
				offset = parsed
			}

			probeID := strings.TrimSpace(query.Get("probe_id"))
			target := strings.TrimSpace(query.Get("target"))

			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			rows, err := db.QueryContext(ctx, `
				SELECT probe_id, target, status, status_code,
				       latency_ms, error, protocol, checked_at
				FROM check_results
				WHERE ($1 = '' OR probe_id = $1)
				  AND ($2 = '' OR target = $2)
				ORDER BY checked_at DESC, id DESC
				LIMIT $3 OFFSET $4
			`, probeID, target, limit+1, offset)
			if err != nil {
				log.Printf("Query check results: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to retrieve results",
				})
				return
			}
			defer rows.Close()

			results := make([]CheckResult, 0, limit+1)

			for rows.Next() {
				var result CheckResult
				var statusCode sql.NullInt64
				var resultError sql.NullString

				if err := rows.Scan(
					&result.ProbeID,
					&result.Target,
					&result.Status,
					&statusCode,
					&result.LatencyMS,
					&resultError,
					&result.Protocol,
					&result.CheckedAt,
				); err != nil {
					log.Printf("Scan check result: %v", err)
					writeJSON(w, http.StatusInternalServerError, ErrorResponse{
						Error: "failed to retrieve results",
					})
					return
				}

				if statusCode.Valid {
					code := int(statusCode.Int64)
					result.StatusCode = &code
				}
				if resultError.Valid {
					result.Error = resultError.String
				}

				results = append(results, result)
			}

			if err := rows.Err(); err != nil {
				log.Printf("Iterate check results: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to retrieve results",
				})
				return
			}

			hasMore := len(results) > limit
			if hasMore {
				results = results[:limit]
			}

			writeJSON(w, http.StatusOK, ResultsResponse{
				Data:    results,
				Limit:   limit,
				Offset:  offset,
				HasMore: hasMore,
			})
		},
	)))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("PulseCheck Central API listening on %s", addr)
	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func authenticate(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		header := r.Header.Get("Authorization")

		if !strings.HasPrefix(header, prefix) ||
			subtle.ConstantTimeCompare(
				[]byte(strings.TrimPrefix(header, prefix)),
				[]byte(token),
			) != 1 {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{
				Error: "unauthorized",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
