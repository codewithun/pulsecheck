import { Suspense } from "react";
import { connection } from "next/server";

import DashboardShell from "@/components/layout/dashboard-shell";
import TargetsTable from "@/components/targets/targets-table";
import { getTargets } from "@/lib/api/targets";
import type { TargetsResponse } from "@/lib/api/types";

function LoadingFallback() {
  return (
    <div className="flex min-h-64 items-center justify-center">
      <p className="text-sm text-slate-500">Loading targets...</p>
    </div>
  );
}

async function TargetsContent() {
  await connection();

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
    <TargetsTable
      initialTargets={initialTargets}
      apiError={apiError}
    />
  );
}

export default function TargetsPage() {
  return (
    <DashboardShell>
      <Suspense fallback={<LoadingFallback />}>
        <TargetsContent />
      </Suspense>
    </DashboardShell>
  );
}
