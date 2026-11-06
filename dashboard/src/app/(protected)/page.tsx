import DashboardShell from "@/components/layout/dashboard-shell";
import OverviewDashboard from "@/components/overview/overview-dashboard";
import { getOverview } from "@/lib/api/overview";
import type { OverviewResponse } from "@/lib/api/types";

export default async function Home() {
  let overview: OverviewResponse | null = null;
  let apiError = false;

  try {
    overview = await getOverview("24h");
  } catch (error) {
    console.error("Failed to load overview:", error);
    apiError = true;
  }

  return (
    <DashboardShell>
      <OverviewDashboard
        overview={overview}
        apiError={apiError}
      />
    </DashboardShell>
  );
}