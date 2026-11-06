import "server-only";

import { centralApiFetch } from "./central-api";
import type { OverviewPeriod, OverviewResponse } from "./types";

export async function getOverview(
  period: OverviewPeriod = "24h",
): Promise<OverviewResponse> {
  return centralApiFetch<OverviewResponse>(
    `/api/v1/overview?period=${period}`,
  );
}
