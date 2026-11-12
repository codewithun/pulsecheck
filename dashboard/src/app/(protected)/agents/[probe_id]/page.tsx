
import { Suspense } from "react";
import { connection } from "next/server";
import { notFound } from "next/navigation";

import DashboardShell from "@/components/layout/dashboard-shell";
import AgentDetail from "@/components/agents/agent-detail";
import { getAgentDetail } from "@/lib/api/agents";
import { CentralApiError } from "@/lib/api/central-api";

interface AgentDetailPageProps {
  params: Promise<{
    probe_id: string;
  }>;
  searchParams: Promise<{
    offset?: string;
  }>;
}

function LoadingFallback() {
  return (
    <div className="flex min-h-64 items-center justify-center">
      <p className="text-sm text-slate-500">
        Loading agent details...
      </p>
    </div>
  );
}

async function AgentDetailContent({
  params,
  searchParams,
}: AgentDetailPageProps) {
  await connection();

  const { probe_id } = await params;
  const query = await searchParams;

  const parsedOffset = Number(query.offset ?? 0);
  const offset =
    Number.isInteger(parsedOffset) && parsedOffset >= 0
      ? parsedOffset
      : 0;

  let detail;

  try {
    detail = await getAgentDetail(probe_id, {
      limit: 20,
      offset,
    });
  } catch (error) {
    if (
      error instanceof CentralApiError &&
      error.status === 404
    ) {
      notFound();
    }

    console.error("Failed to load agent detail:", error);
    throw error;
  }

  return <AgentDetail detail={detail} />;
}

export default function AgentDetailPage({
  params,
  searchParams,
}: AgentDetailPageProps) {
  return (
    <DashboardShell>
      <Suspense fallback={<LoadingFallback />}>
        <AgentDetailContent
          params={params}
          searchParams={searchParams}
        />
      </Suspense>
    </DashboardShell>
  );
}
