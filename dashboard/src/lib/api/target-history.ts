
import "server-only";

import { centralApiFetch } from "./central-api";
import type {
  TargetHistoryResponse,
  TargetPeriod,
} from "./types";

interface GetTargetHistoryParams {
  probe_id: string;
  target: string;
  period?: TargetPeriod;
  limit?: number;
  offset?: number;
}

export async function getTargetHistory(
  params: GetTargetHistoryParams,
): Promise<TargetHistoryResponse> {
  const query = new URLSearchParams({
    probe_id: params.probe_id,
    target: params.target,
    limit: String(params.limit ?? 20),
    offset: String(params.offset ?? 0),
  });

  if (params.period) {
    query.set("period", params.period);
  }

  return centralApiFetch<TargetHistoryResponse>(
    `/api/v1/targets/history?${query.toString()}`,
  );
}
