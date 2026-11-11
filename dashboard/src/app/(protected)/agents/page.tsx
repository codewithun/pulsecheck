
import { Suspense } from "react";
import { connection } from "next/server";

import DashboardShell from "@/components/layout/dashboard-shell";
import AgentsTable from "@/components/agents/agents-table";
import { getAgents } from "@/lib/api/agents";
import type { Agent } from "@/lib/api/types";

function LoadingFallback() {
  return (
    <div className="flex min-h-64 items-center justify-center">
      <p className="text-sm text-slate-500">Loading agents...</p>
    </div>
  );
}

async function AgentsContent() {
  await connection();

  let agents: Agent[] = [];
  let apiError = false;

  try {
    const response = await getAgents();
    agents = response.data;
  } catch (error) {
    console.error("Failed to load agents:", error);
    apiError = true;
  }

  return (
    <AgentsTable
      agents={agents}
      apiError={apiError}
    />
  );
}

export default function AgentsPage() {
  return (
    <DashboardShell>
      <Suspense fallback={<LoadingFallback />}>
        <AgentsContent />
      </Suspense>
    </DashboardShell>
  );
}
