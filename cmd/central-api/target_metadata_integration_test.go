package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func metadataTestRequest(
	t *testing.T,
	handler http.HandlerFunc,
	method string,
	path string,
	body any,
) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody bytes.Buffer

	if body != nil {
		if err := json.NewEncoder(&requestBody).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &requestBody)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	handler(rec, req)

	return rec
}

func metadataTestPathRequest(
	t *testing.T,
	handler http.HandlerFunc,
	method string,
	path string,
	id string,
	body any,
) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody bytes.Buffer

	if body != nil {
		if err := json.NewEncoder(&requestBody).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &requestBody)
	req.SetPathValue("id", id)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	handler(rec, req)

	return rec
}

func decodeMetadataTestResponse[T any](
	t *testing.T,
	rec *httptest.ResponseRecorder,
) T {
	t.Helper()

	var result T

	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, rec.Body.String())
	}

	return result
}

func TestTargetMetadataIntegration(t *testing.T) {
	db := openTestDB(t)

	probeID := fmt.Sprintf("metadata-test-%d", time.Now().UnixNano())
	target := "https://metadata-test.invalid"

	ctx := context.Background()

	err := saveCheckResult(ctx, db, CheckResult{
		ProbeID:   probeID,
		Target:    target,
		Status:    "UP",
		LatencyMS: 10,
		Protocol:  "HTTP",
		CheckedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create test target: %v", err)
	}

	var groupIDs []int64
	var tagIDs []int64

	t.Cleanup(func() {
		_, _ = db.Exec(
			`DELETE FROM target_tag_assignments WHERE probe_id = $1`,
			probeID,
		)

		_, _ = db.Exec(
			`DELETE FROM target_group_assignments WHERE probe_id = $1`,
			probeID,
		)

		_, _ = db.Exec(
			`DELETE FROM check_results WHERE probe_id = $1`,
			probeID,
		)

		_, _ = db.Exec(
			`DELETE FROM monitoring_state WHERE probe_id = $1`,
			probeID,
		)

		_, _ = db.Exec(
			`DELETE FROM alert_outbox WHERE probe_id = $1`,
			probeID,
		)

		for _, id := range groupIDs {
			_, _ = db.Exec(
				`DELETE FROM target_groups WHERE id = $1`,
				id,
			)
		}

		for _, id := range tagIDs {
			_, _ = db.Exec(
				`DELETE FROM target_tags WHERE id = $1`,
				id,
			)
		}
	})

	groupHandler := targetGroupsHandler(db)
	tagHandler := targetTagsHandler(db)
	metadataHandler := targetMetadataHandler(db)

	// -----------------------------------------------------------------
	// Create group
	// -----------------------------------------------------------------

	groupResponse := metadataTestRequest(
		t,
		groupHandler,
		http.MethodPost,
		"/api/v1/target-groups",
		CreateGroupRequest{
			Name:        "Integration Production",
			Description: "Integration test group",
		},
	)

	if groupResponse.Code != http.StatusCreated {
		t.Fatalf(
			"create group: got %d, want %d; body=%s",
			groupResponse.Code,
			http.StatusCreated,
			groupResponse.Body.String(),
		)
	}

	group := decodeMetadataTestResponse[TargetGroup](t, groupResponse)
	groupIDs = append(groupIDs, group.ID)

	if group.Name != "Integration Production" {
		t.Fatalf("group name = %q", group.Name)
	}

	// -----------------------------------------------------------------
	// Reject duplicate group name (case insensitive)
	// -----------------------------------------------------------------

	duplicateGroup := metadataTestRequest(
		t,
		groupHandler,
		http.MethodPost,
		"/api/v1/target-groups",
		CreateGroupRequest{
			Name: "integration production",
		},
	)

	if duplicateGroup.Code != http.StatusConflict {
		t.Fatalf(
			"duplicate group: got %d, want %d",
			duplicateGroup.Code,
			http.StatusConflict,
		)
	}

	// -----------------------------------------------------------------
	// Update group
	// -----------------------------------------------------------------

	groupID := strconv.FormatInt(group.ID, 10)

	updatedGroup := metadataTestPathRequest(
		t,
		groupHandler,
		http.MethodPatch,
		"/api/v1/target-groups/"+groupID,
		groupID,
		UpdateGroupRequest{
			Name:        "Integration Production Updated",
			Description: "Updated description",
		},
	)

	if updatedGroup.Code != http.StatusOK {
		t.Fatalf(
			"update group: got %d; body=%s",
			updatedGroup.Code,
			updatedGroup.Body.String(),
		)
	}

	updated := decodeMetadataTestResponse[TargetGroup](t, updatedGroup)

	if updated.Name != "Integration Production Updated" {
		t.Fatalf("updated group name = %q", updated.Name)
	}

	// -----------------------------------------------------------------
	// Create tags
	// -----------------------------------------------------------------

	for _, name := range []string{"Critical", "Payment"} {
		response := metadataTestRequest(
			t,
			tagHandler,
			http.MethodPost,
			"/api/v1/target-tags",
			CreateTagRequest{
				Name: name,
			},
		)

		if response.Code != http.StatusCreated {
			t.Fatalf(
				"create tag %q: got %d; body=%s",
				name,
				response.Code,
				response.Body.String(),
			)
		}

		tag := decodeMetadataTestResponse[TargetTag](t, response)
		tagIDs = append(tagIDs, tag.ID)

		if tag.Name != "critical" && tag.Name != "payment" {
			t.Fatalf("tag name was not normalized: %q", tag.Name)
		}
	}

	// -----------------------------------------------------------------
	// Reject duplicate tag
	// -----------------------------------------------------------------

	duplicateTag := metadataTestRequest(
		t,
		tagHandler,
		http.MethodPost,
		"/api/v1/target-tags",
		CreateTagRequest{
			Name: "CRITICAL",
		},
	)

	if duplicateTag.Code != http.StatusConflict {
		t.Fatalf(
			"duplicate tag: got %d, want %d",
			duplicateTag.Code,
			http.StatusConflict,
		)
	}

	// -----------------------------------------------------------------
	// Assign group and tags to target
	// -----------------------------------------------------------------

	metadataPut := metadataTestRequest(
		t,
		metadataHandler,
		http.MethodPut,
		"/api/v1/target-metadata",
		TargetMetadataRequest{
			ProbeID: probeID,
			Target:  target,
			GroupID: &group.ID,
			Tags:    tagIDs,
		},
	)

	if metadataPut.Code != http.StatusOK {
		t.Fatalf(
			"assign metadata: got %d; body=%s",
			metadataPut.Code,
			metadataPut.Body.String(),
		)
	}

	metadata := decodeMetadataTestResponse[TargetMetadata](t, metadataPut)

	if metadata.Group == nil || metadata.Group.ID != group.ID {
		t.Fatal("assigned group was not returned")
	}

	if len(metadata.Tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(metadata.Tags))
	}

	// -----------------------------------------------------------------
	// Read metadata
	// -----------------------------------------------------------------

	getPath := "/api/v1/target-metadata?probe_id=" +
		probeID + "&target=" + target

	metadataGet := metadataTestRequest(
		t,
		metadataHandler,
		http.MethodGet,
		getPath,
		nil,
	)

	if metadataGet.Code != http.StatusOK {
		t.Fatalf(
			"get metadata: got %d; body=%s",
			metadataGet.Code,
			metadataGet.Body.String(),
		)
	}

	readMetadata := decodeMetadataTestResponse[TargetMetadata](t, metadataGet)

	if readMetadata.Group == nil || readMetadata.Group.ID != group.ID {
		t.Fatal("GET returned incorrect group")
	}

	if len(readMetadata.Tags) != 2 {
		t.Fatalf("GET returned %d tags, want 2", len(readMetadata.Tags))
	}

	// -----------------------------------------------------------------
	// Invalid tag must not partially modify existing metadata
	// -----------------------------------------------------------------

	invalidMetadata := metadataTestRequest(
		t,
		metadataHandler,
		http.MethodPut,
		"/api/v1/target-metadata",
		TargetMetadataRequest{
			ProbeID: probeID,
			Target:  target,
			Tags:    []int64{tagIDs[0], 999999999},
		},
	)

	if invalidMetadata.Code != http.StatusBadRequest {
		t.Fatalf(
			"invalid tag assignment: got %d, want %d; body=%s",
			invalidMetadata.Code,
			http.StatusBadRequest,
			invalidMetadata.Body.String(),
		)
	}

	afterInvalid := metadataTestRequest(
		t,
		metadataHandler,
		http.MethodGet,
		getPath,
		nil,
	)

	if afterInvalid.Code != http.StatusOK {
		t.Fatalf(
			"get metadata after invalid update: got %d; body=%s",
			afterInvalid.Code,
			afterInvalid.Body.String(),
		)
	}

	preserved := decodeMetadataTestResponse[TargetMetadata](t, afterInvalid)

	if preserved.Group == nil || preserved.Group.ID != group.ID {
		t.Fatal("group changed after failed metadata update")
	}

	if len(preserved.Tags) != 2 {
		t.Fatalf(
			"tags changed after failed update: got %d, want 2",
			len(preserved.Tags),
		)
	}

	// -----------------------------------------------------------------
	// Replace metadata with no group and no tags
	// -----------------------------------------------------------------

	clearMetadata := metadataTestRequest(
		t,
		metadataHandler,
		http.MethodPut,
		"/api/v1/target-metadata",
		TargetMetadataRequest{
			ProbeID: probeID,
			Target:  target,
			GroupID: nil,
			Tags:    []int64{},
		},
	)

	if clearMetadata.Code != http.StatusOK {
		t.Fatalf(
			"clear metadata: got %d; body=%s",
			clearMetadata.Code,
			clearMetadata.Body.String(),
		)
	}

	cleared := decodeMetadataTestResponse[TargetMetadata](t, clearMetadata)

	if cleared.Group != nil {
		t.Fatal("group should be cleared")
	}

	if len(cleared.Tags) != 0 {
		t.Fatalf("tags should be empty, got %d", len(cleared.Tags))
	}

	// -----------------------------------------------------------------
	// Reject metadata for nonexistent target
	// -----------------------------------------------------------------

	missingTarget := metadataTestRequest(
		t,
		metadataHandler,
		http.MethodPut,
		"/api/v1/target-metadata",
		TargetMetadataRequest{
			ProbeID: probeID,
			Target:  "https://missing-target.invalid",
		},
	)

	if missingTarget.Code != http.StatusNotFound {
		t.Fatalf(
			"missing target: got %d, want %d",
			missingTarget.Code,
			http.StatusNotFound,
		)
	}

	// -----------------------------------------------------------------
	// Delete group and tags
	// -----------------------------------------------------------------

	groupDelete := metadataTestPathRequest(
		t,
		groupHandler,
		http.MethodDelete,
		"/api/v1/target-groups/"+groupID,
		groupID,
		nil,
	)

	if groupDelete.Code != http.StatusNoContent {
		t.Fatalf(
			"delete group: got %d, want %d",
			groupDelete.Code,
			http.StatusNoContent,
		)
	}

	groupIDs = nil

	for _, id := range tagIDs {
		tagID := strconv.FormatInt(id, 10)

		tagDelete := metadataTestPathRequest(
			t,
			tagHandler,
			http.MethodDelete,
			"/api/v1/target-tags/"+tagID,
			tagID,
			nil,
		)

		if tagDelete.Code != http.StatusNoContent {
			t.Fatalf(
				"delete tag %d: got %d, want %d",
				id,
				tagDelete.Code,
				http.StatusNoContent,
			)
		}
	}

	tagIDs = nil
}
