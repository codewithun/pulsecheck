
import "server-only";

import { centralApiFetch } from "./central-api";
import type { AgentsResponse } from "./types";

export async function getAgents(): Promise<AgentsResponse> {
  return centralApiFetch<AgentsResponse>("/api/v1/agents");
}
