import { Suspense } from "react";
import { connection } from "next/server";

import DashboardShell from "@/components/layout/dashboard-shell";
import OverviewDashboard from "@/components/overview/overview-dashboard";
import { getOverview } from "@/lib/api/overview";
import type { OverviewResponse } from "@/lib/api/types";

function LoadingFallback() {
  return (
    <div className="flex min-h-64 items-center justify-center">
      <p className="text-sm text-slate-500">Loading overview...</p>
    </div>
  );
}

async function OverviewContent() {
  await connection();

  let overview: OverviewResponse | null = null;
  let apiError = false;

  try {
    overview = await getOverview("24h");
  } catch (error) {
    console.error("Failed to load overview:", error);
    apiError = true;
  }

  return (
    <OverviewDashboard
      overview={overview}
      apiError={apiError}
    />
  );
}

export default function Home() {
  return (
    <DashboardShell>
      <Suspense fallback={<LoadingFallback />}>
        <OverviewContent />
      </Suspense>
    </DashboardShell>
  );
}
