import "server-only";

import { centralApiFetch } from "@/lib/api/central-api";
import type {
  TargetGroup,
  TargetGroupsResponse,
  TargetMetadata,
  TargetTag,
  TargetTagsResponse,
  UpdateTargetMetadataRequest,
} from "@/lib/api/types";

export async function getTargetGroups() {
  return centralApiFetch<TargetGroupsResponse>(
    "/api/v1/target-groups",
  );
}

export async function createTargetGroup(data: {
  name: string;
  description: string;
}) {
  return centralApiFetch<TargetGroup>(
    "/api/v1/target-groups",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    },
  );
}

export async function updateTargetGroup(
  id: number,
  data: { name: string; description: string },
) {
  return centralApiFetch<TargetGroup>(
    `/api/v1/target-groups/${id}`,
    {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    },
  );
}

export async function deleteTargetGroup(id: number) {
  return centralApiFetch<void>(
    `/api/v1/target-groups/${id}`,
    { method: "DELETE" },
  );
}

export async function getTargetTags() {
  return centralApiFetch<TargetTagsResponse>(
    "/api/v1/target-tags",
  );
}

export async function createTargetTag(data: { name: string }) {
  return centralApiFetch<TargetTag>(
    "/api/v1/target-tags",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    },
  );
}

export async function deleteTargetTag(id: number) {
  return centralApiFetch<void>(
    `/api/v1/target-tags/${id}`,
    { method: "DELETE" },
  );
}

export async function getTargetMetadata(
  probeId: string,
  target: string,
) {
  const params = new URLSearchParams({
    probe_id: probeId,
    target,
  });

  return centralApiFetch<TargetMetadata>(
    `/api/v1/target-metadata?${params.toString()}`,
  );
}

export async function updateTargetMetadata(
  data: UpdateTargetMetadataRequest,
) {
  return centralApiFetch<TargetMetadata>(
    "/api/v1/target-metadata",
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    },
  );
}