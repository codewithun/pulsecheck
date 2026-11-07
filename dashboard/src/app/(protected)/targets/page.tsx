
import DashboardShell from "@/components/layout/dashboard-shell";
import TargetsTable from "@/components/targets/targets-table";
import { getTargets } from "@/lib/api/targets";
import type { TargetsResponse } from "@/lib/api/types";

export default async function TargetsPage() {
  let initialTargets: TargetsResponse = {
    data: [],
    limit: 10,
    offset: 0,
    has_more: false,
  };

  let apiError = false;

  try {
    initialTargets = await getTargets({
      limit: 10,
      offset: 0,
    });
  } catch (error) {
    console.error("Failed to load targets:", error);
    apiError = true;
  }

  return (
    <DashboardShell>
      <TargetsTable
        initialTargets={initialTargets}
        apiError={apiError}
      />
    </DashboardShell>
  );
}
