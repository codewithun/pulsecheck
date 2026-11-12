import "server-only";

import { centralApiFetch } from "./central-api";
import type {
  AgentDetailResponse,
  AgentsResponse,
} from "./types";

export async function getAgents(): Promise<AgentsResponse> {
  return centralApiFetch<AgentsResponse>("/api/v1/agents");
}

interface GetAgentDetailParams {
  limit?: number;
  offset?: number;
}

export async function getAgentDetail(
  probeId: string,
  params: GetAgentDetailParams = {},
): Promise<AgentDetailResponse> {
  const query = new URLSearchParams();

  if (params.limit !== undefined) {
    query.set("limit", String(params.limit));
  }

  if (params.offset !== undefined) {
    query.set("offset", String(params.offset));
  }

  return centralApiFetch<AgentDetailResponse>(
    `/api/v1/agents/${encodeURIComponent(probeId)}?${query.toString()}`,
  );
}
