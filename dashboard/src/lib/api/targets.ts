
import "server-only";

import { centralApiFetch } from "./central-api";
import type {
  TargetPeriod,
  TargetStatus,
  TargetsResponse,
} from "./types";

interface GetTargetsParams {
  limit?: number;
  offset?: number;
  probe_id?: string;
  status?: TargetStatus;
  search?: string;
  period?: TargetPeriod;
}

export async function getTargets(
  params: GetTargetsParams = {},
): Promise<TargetsResponse> {
  const query = new URLSearchParams();

  if (params.limit !== undefined) {
    query.set("limit", String(params.limit));
  }

  if (params.offset !== undefined) {
    query.set("offset", String(params.offset));
  }

  if (params.probe_id) {
    query.set("probe_id", params.probe_id);
  }

  if (params.status) {
    query.set("status", params.status);
  }

  if (params.search) {
    query.set("search", params.search);
  }

  if (params.period) {
    query.set("period", params.period);
  }

  return centralApiFetch<TargetsResponse>(
    `/api/v1/targets?${query.toString()}`,
  );
}
