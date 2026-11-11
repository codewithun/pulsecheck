import { Suspense } from "react";
import { connection } from "next/server";
import { notFound } from "next/navigation";

import DashboardShell from "@/components/layout/dashboard-shell";
import TargetDetail from "@/components/targets/target-detail";
import { getTargets } from "@/lib/api/targets";
import { getTargetHistory } from "@/lib/api/target-history";

interface TargetDetailPageProps {
  searchParams: Promise<{
    probe_id?: string;
    target?: string;
  }>;
}

function LoadingFallback() {
  return (
    <div className="flex min-h-64 items-center justify-center">
      <p className="text-sm text-slate-500">
        Loading target details...
      </p>
    </div>
  );
}

async function TargetDetailContent({
  searchParams,
}: TargetDetailPageProps) {
  await connection();

  const params = await searchParams;

  const probeId = params.probe_id;
  const targetUrl = params.target;

  if (!probeId || !targetUrl) {
    notFound();
  }

  const [targetsResponse, historyResponse] = await Promise.all([
    getTargets({
      limit: 100,
      probe_id: probeId,
      search: targetUrl,
    }),
    getTargetHistory({
      probe_id: probeId,
      target: targetUrl,
      limit: 20,
      offset: 0,
    }),
  ]);

  const target = targetsResponse.data.find(
    (item) =>
      item.probe_id === probeId &&
      item.target === targetUrl,
  );

  if (!target) {
    notFound();
  }

  return (
    <TargetDetail
      target={target}
      initialHistory={historyResponse}
    />
  );
}

export default function TargetDetailPage({
  searchParams,
}: TargetDetailPageProps) {
  return (
    <DashboardShell>
      <Suspense fallback={<LoadingFallback />}>
        <TargetDetailContent searchParams={searchParams} />
      </Suspense>
    </DashboardShell>
  );
}
