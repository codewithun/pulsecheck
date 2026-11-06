
import DashboardShell from "@/components/layout/dashboard-shell";
import AgentsTable from "@/components/agents/agents-table";
import { getAgents } from "@/lib/api/agents";
import type { Agent } from "@/lib/api/types";

export default async function AgentsPage() {
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
    <DashboardShell>
      <AgentsTable
        agents={agents}
        apiError={apiError}
      />
    </DashboardShell>
  );
}
