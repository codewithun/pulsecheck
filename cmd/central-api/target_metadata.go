package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type TargetGroup struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TargetTag struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type TargetMetadata struct {
	ProbeID string       `json:"probe_id"`
	Target  string       `json:"target"`
	Group   *TargetGroup `json:"group"`
	Tags    []TargetTag  `json:"tags"`
}

type TargetMetadataRequest struct {
	ProbeID string  `json:"probe_id"`
	Target  string  `json:"target"`
	GroupID *int64  `json:"group_id"`
	Tags    []int64 `json:"tags"`
}

type CreateGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateTagRequest struct {
	Name string `json:"name"`
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "invalid JSON payload",
		})
		return false
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "request must contain one JSON object",
		})
		return false
	}

	return true
}

func metadataDBContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 5*time.Second)
}

// -----------------------------------------------------------------------------
// Target groups
// -----------------------------------------------------------------------------

func targetGroupsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := metadataDBContext(r)
		defer cancel()

		switch r.Method {
		case http.MethodGet:
			rows, err := db.QueryContext(ctx, `
				SELECT id, name, description, created_at, updated_at
				FROM target_groups
				ORDER BY name
			`)
			if err != nil {
				log.Printf("Query target groups: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to retrieve target groups",
				})
				return
			}
			defer rows.Close()

			groups := make([]TargetGroup, 0)

			for rows.Next() {
				var group TargetGroup
				if err := rows.Scan(
					&group.ID,
					&group.Name,
					&group.Description,
					&group.CreatedAt,
					&group.UpdatedAt,
				); err != nil {
					log.Printf("Scan target group: %v", err)
					writeJSON(w, http.StatusInternalServerError, ErrorResponse{
						Error: "failed to retrieve target groups",
					})
					return
				}
				groups = append(groups, group)
			}

			if err := rows.Err(); err != nil {
				log.Printf("Iterate target groups: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to retrieve target groups",
				})
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"data": groups})

		case http.MethodPost:
			var request CreateGroupRequest
			if !decodeJSONBody(w, r, &request) {
				return
			}

			request.Name = strings.TrimSpace(request.Name)
			request.Description = strings.TrimSpace(request.Description)

			if request.Name == "" || len(request.Name) > 100 {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "name is required and must not exceed 100 characters",
				})
				return
			}

			var group TargetGroup
			err := db.QueryRowContext(ctx, `
				INSERT INTO target_groups (name, description)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
				RETURNING id, name, description, created_at, updated_at
			`, request.Name, request.Description).Scan(
				&group.ID,
				&group.Name,
				&group.Description,
				&group.CreatedAt,
				&group.UpdatedAt,
			)

			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusConflict, ErrorResponse{
					Error: "a group with this name already exists",
				})
				return
			}
			if err != nil {
				log.Printf("Create target group: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to create target group",
				})
				return
			}

			writeJSON(w, http.StatusCreated, group)

		case http.MethodPatch:
			id, ok := metadataPathID(w, r)
			if !ok {
				return
			}

			var request UpdateGroupRequest
			if !decodeJSONBody(w, r, &request) {
				return
			}

			request.Name = strings.TrimSpace(request.Name)
			request.Description = strings.TrimSpace(request.Description)

			if request.Name == "" || len(request.Name) > 100 {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "name is required and must not exceed 100 characters",
				})
				return
			}

			var group TargetGroup
			err := db.QueryRowContext(ctx, `
				UPDATE target_groups
				SET name = $2, description = $3, updated_at = NOW()
				WHERE id = $1
				RETURNING id, name, description, created_at, updated_at
			`, id, request.Name, request.Description).Scan(
				&group.ID,
				&group.Name,
				&group.Description,
				&group.CreatedAt,
				&group.UpdatedAt,
			)

			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, ErrorResponse{
					Error: "target group not found",
				})
				return
			}
			if err != nil {
				log.Printf("Update target group: %v", err)
				writeJSON(w, http.StatusConflict, ErrorResponse{
					Error: "failed to update target group; name may already exist",
				})
				return
			}

			writeJSON(w, http.StatusOK, group)

		case http.MethodDelete:
			id, ok := metadataPathID(w, r)
			if !ok {
				return
			}

			result, err := db.ExecContext(ctx, `
				DELETE FROM target_groups WHERE id = $1
			`, id)
			if err != nil {
				log.Printf("Delete target group: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to delete target group",
				})
				return
			}

			affected, _ := result.RowsAffected()
			if affected == 0 {
				writeJSON(w, http.StatusNotFound, ErrorResponse{
					Error: "target group not found",
				})
				return
			}

			w.WriteHeader(http.StatusNoContent)

		default:
			w.Header().Set("Allow", "GET, POST, PATCH, DELETE")
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error: "method not allowed",
			})
		}
	}
}

// -----------------------------------------------------------------------------
// Target tags
// -----------------------------------------------------------------------------

func targetTagsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := metadataDBContext(r)
		defer cancel()

		switch r.Method {
		case http.MethodGet:
			rows, err := db.QueryContext(ctx, `
				SELECT id, name, created_at
				FROM target_tags
				ORDER BY name
			`)
			if err != nil {
				log.Printf("Query target tags: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to retrieve target tags",
				})
				return
			}
			defer rows.Close()

			tags := make([]TargetTag, 0)

			for rows.Next() {
				var tag TargetTag
				if err := rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt); err != nil {
					log.Printf("Scan target tag: %v", err)
					writeJSON(w, http.StatusInternalServerError, ErrorResponse{
						Error: "failed to retrieve target tags",
					})
					return
				}
				tags = append(tags, tag)
			}

			if err := rows.Err(); err != nil {
				log.Printf("Iterate target tags: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to retrieve target tags",
				})
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"data": tags})

		case http.MethodPost:
			var request CreateTagRequest
			if !decodeJSONBody(w, r, &request) {
				return
			}

			request.Name = strings.ToLower(strings.TrimSpace(request.Name))

			if request.Name == "" || len(request.Name) > 50 {
				writeJSON(w, http.StatusBadRequest, ErrorResponse{
					Error: "name is required and must not exceed 50 characters",
				})
				return
			}

			var tag TargetTag
			err := db.QueryRowContext(ctx, `
				INSERT INTO target_tags (name)
				VALUES ($1)
				ON CONFLICT DO NOTHING
				RETURNING id, name, created_at
			`, request.Name).Scan(&tag.ID, &tag.Name, &tag.CreatedAt)

			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusConflict, ErrorResponse{
					Error: "a tag with this name already exists",
				})
				return
			}
			if err != nil {
				log.Printf("Create target tag: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to create target tag",
				})
				return
			}

			writeJSON(w, http.StatusCreated, tag)

		case http.MethodDelete:
			id, ok := metadataPathID(w, r)
			if !ok {
				return
			}

			result, err := db.ExecContext(ctx, `
				DELETE FROM target_tags WHERE id = $1
			`, id)
			if err != nil {
				log.Printf("Delete target tag: %v", err)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "failed to delete target tag",
				})
				return
			}

			affected, _ := result.RowsAffected()
			if affected == 0 {
				writeJSON(w, http.StatusNotFound, ErrorResponse{
					Error: "target tag not found",
				})
				return
			}

			w.WriteHeader(http.StatusNoContent)

		default:
			w.Header().Set("Allow", "GET, POST, DELETE")
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error: "method not allowed",
			})
		}
	}
}

func metadataPathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "id must be a positive integer",
		})
		return 0, false
	}
	return id, true
}

// -----------------------------------------------------------------------------
// Target metadata assignment
// -----------------------------------------------------------------------------

func targetMetadataHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getTargetMetadata(w, r, db)
		case http.MethodPut:
			putTargetMetadata(w, r, db)
		default:
			w.Header().Set("Allow", "GET, PUT")
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error: "method not allowed",
			})
		}
	}
}

func getTargetMetadata(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	query := r.URL.Query()
	probeID := strings.TrimSpace(query.Get("probe_id"))
	target := strings.TrimSpace(query.Get("target"))

	if probeID == "" || target == "" || len(probeID) > 100 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "valid probe_id and target are required",
		})
		return
	}

	ctx, cancel := metadataDBContext(r)
	defer cancel()

	metadata := TargetMetadata{
		ProbeID: probeID,
		Target:  target,
		Tags:    make([]TargetTag, 0),
	}

	var group TargetGroup
	err := db.QueryRowContext(ctx, `
		SELECT g.id, g.name, g.description, g.created_at, g.updated_at
		FROM target_group_assignments a
		JOIN target_groups g ON g.id = a.group_id
		WHERE a.probe_id = $1 AND a.target = $2
	`, probeID, target).Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.CreatedAt,
		&group.UpdatedAt,
	)

	if err == nil {
		metadata.Group = &group
	} else if !errors.Is(err, sql.ErrNoRows) {
		log.Printf("Get target group metadata: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error: "failed to retrieve target metadata",
		})
		return
	}

	rows, err := db.QueryContext(ctx, `
		SELECT t.id, t.name, t.created_at
		FROM target_tag_assignments a
		JOIN target_tags t ON t.id = a.tag_id
		WHERE a.probe_id = $1 AND a.target = $2
		ORDER BY t.name
	`, probeID, target)
	if err != nil {
		log.Printf("Get target tag metadata: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error: "failed to retrieve target metadata",
		})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var tag TargetTag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt); err != nil {
			log.Printf("Scan target metadata tag: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to retrieve target metadata",
			})
			return
		}
		metadata.Tags = append(metadata.Tags, tag)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Iterate target metadata tags: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error: "failed to retrieve target metadata",
		})
		return
	}

	writeJSON(w, http.StatusOK, metadata)
}

func putTargetMetadata(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var request TargetMetadataRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}

	request.ProbeID = strings.TrimSpace(request.ProbeID)
	request.Target = strings.TrimSpace(request.Target)

	if request.ProbeID == "" || len(request.ProbeID) > 100 || request.Target == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "valid probe_id and target are required",
		})
		return
	}

	if len(request.Tags) > 50 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "a target can have at most 50 tags",
		})
		return
	}

	seen := make(map[int64]struct{}, len(request.Tags))
	for _, id := range request.Tags {
		if id < 1 {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "tag IDs must be positive integers",
			})
			return
		}
		if _, exists := seen[id]; exists {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "duplicate tag IDs are not allowed",
			})
			return
		}
		seen[id] = struct{}{}
	}

	if request.GroupID != nil && *request.GroupID < 1 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "group_id must be a positive integer or null",
		})
		return
	}

	ctx, cancel := metadataDBContext(r)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Begin target metadata transaction: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error: "failed to update target metadata",
		})
		return
	}
	defer tx.Rollback()

	var targetExists bool
	err = tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM check_results
			WHERE probe_id = $1 AND target = $2
		)
	`, request.ProbeID, request.Target).Scan(&targetExists)

	if err != nil {
		log.Printf("Validate target metadata target: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error: "failed to validate target",
		})
		return
	}

	if !targetExists {
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Error: "target not found",
		})
		return
	}

	if request.GroupID != nil {
		var groupExists bool
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS (SELECT 1 FROM target_groups WHERE id = $1)
		`, *request.GroupID).Scan(&groupExists)

		if err != nil {
			log.Printf("Validate target group: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to validate target group",
			})
			return
		}

		if !groupExists {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "target group does not exist",
			})
			return
		}
	}

	if len(request.Tags) > 0 {
		var tagCount int
		err = tx.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM target_tags WHERE id = ANY($1)
		`, request.Tags).Scan(&tagCount)

		if err != nil {
			log.Printf("Validate target tags: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to validate target tags",
			})
			return
		}

		if tagCount != len(request.Tags) {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "one or more target tags do not exist",
			})
			return
		}
	}

	if request.GroupID == nil {
		_, err = tx.ExecContext(ctx, `
			DELETE FROM target_group_assignments
			WHERE probe_id = $1 AND target = $2
		`, request.ProbeID, request.Target)
	} else {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO target_group_assignments (probe_id, target, group_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (probe_id, target)
			DO UPDATE SET group_id = EXCLUDED.group_id
		`, request.ProbeID, request.Target, *request.GroupID)
	}

	if err != nil {
		log.Printf("Assign target group: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error: "failed to assign target group",
		})
		return
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM target_tag_assignments
		WHERE probe_id = $1 AND target = $2
	`, request.ProbeID, request.Target)
	if err != nil {
		log.Printf("Replace target tags: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error: "failed to update target tags",
		})
		return
	}

	for _, tagID := range request.Tags {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO target_tag_assignments (probe_id, target, tag_id)
			VALUES ($1, $2, $3)
		`, request.ProbeID, request.Target, tagID)
		if err != nil {
			log.Printf("Assign target tag: %v", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to assign target tags",
			})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Commit target metadata: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error: "failed to save target metadata",
		})
		return
	}

	// Build a request containing the target identity from the PUT body.
	metadataURL := *r.URL
	query := metadataURL.Query()

	query.Set("probe_id", request.ProbeID)
	query.Set("target", request.Target)

	metadataURL.RawQuery = query.Encode()

	metadataRequest := r.Clone(r.Context())
	metadataRequest.URL = &metadataURL

	getTargetMetadata(w, metadataRequest, db)
}
