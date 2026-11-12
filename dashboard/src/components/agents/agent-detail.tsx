
import Link from "next/link";
import {
  ArrowLeft,
  ArrowRight,
  CircleCheck,
  CircleX,
  Clock,
  Globe,
  Server,
  ShieldAlert,
  Activity,
} from "lucide-react";

import type { AgentDetailResponse } from "@/lib/api/types";

interface AgentDetailProps {
  detail: AgentDetailResponse;
}

function formatDate(value: string | null): string {
  if (!value) return "Never";

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) return "Invalid date";

  return new Intl.DateTimeFormat("id-ID", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "Asia/Jakarta",
  }).format(date);
}

function StatusBadge({ status }: { status: string }) {
  const styles: Record<string, string> = {
    ONLINE: "bg-emerald-50 text-emerald-700",
    OFFLINE: "bg-red-50 text-red-700",
    UP: "bg-emerald-50 text-emerald-700",
    DOWN: "bg-red-50 text-red-700",
    UNKNOWN: "bg-slate-100 text-slate-600",
    INVALID: "bg-amber-50 text-amber-700",
  };

  return (
    <span
      className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${
        styles[status] ?? styles.UNKNOWN
      }`}
    >
      {status}
    </span>
  );
}

export default function AgentDetail({ detail }: AgentDetailProps) {
  const { agent, summary, targets } = detail;

  const currentPage = Math.floor(targets.offset / targets.limit) + 1;
  const previousOffset = Math.max(0, targets.offset - targets.limit);
  const nextOffset = targets.offset + targets.limit;

  const start = targets.data.length
    ? targets.offset + 1
    : 0;

  const end = targets.offset + targets.data.length;

  function targetDetailUrl(target: string) {
    const params = new URLSearchParams({
      probe_id: agent.probe_id,
      target,
    });

    return `/targets/detail?${params.toString()}`;
  }

  function paginationUrl(offset: number) {
    return `/agents/${encodeURIComponent(agent.probe_id)}?offset=${offset}`;
  }

  const statistics = [
    {
      label: "Total Targets",
      value: summary.total,
      icon: Globe,
      color: "text-blue-600",
      background: "bg-blue-50",
    },
    {
      label: "Targets UP",
      value: summary.up,
      icon: CircleCheck,
      color: "text-emerald-600",
      background: "bg-emerald-50",
    },
    {
      label: "Targets DOWN",
      value: summary.down,
      icon: CircleX,
      color: "text-red-600",
      background: "bg-red-50",
    },
    {
      label: "Unknown",
      value: summary.unknown,
      icon: ShieldAlert,
      color: "text-amber-600",
      background: "bg-amber-50",
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/agents"
          className="inline-flex items-center gap-2 text-sm text-slate-500 transition hover:text-blue-600"
        >
          <ArrowLeft size={16} />
          Back to Agents
        </Link>
      </div>

      <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-col justify-between gap-5 sm:flex-row sm:items-center">
          <div className="flex items-start gap-4">
            <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-blue-50 text-blue-600">
              <Server size={27} />
            </div>

            <div>
              <h1 className="text-xl font-bold text-slate-900">
                {agent.hostname}
              </h1>

              <p className="mt-1 text-sm text-slate-500">
                Probe Agent Details
              </p>

              <code className="mt-3 inline-block rounded-md bg-slate-100 px-2.5 py-1 text-xs text-slate-600">
                {agent.probe_id}
              </code>
            </div>
          </div>

          <div>
            <StatusBadge status={agent.status} />
          </div>
        </div>

        <div className="mt-6 grid gap-4 border-t border-slate-100 pt-5 sm:grid-cols-2">
          <div className="flex items-center gap-3">
            <Clock size={18} className="text-slate-400" />
            <div>
              <p className="text-xs text-slate-400">
                Last Heartbeat
              </p>
              <p className="mt-1 text-sm font-medium text-slate-700">
                {formatDate(agent.last_seen)}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <Activity size={18} className="text-slate-400" />
            <div>
              <p className="text-xs text-slate-400">
                Agent Status
              </p>
              <p className="mt-1 text-sm font-medium text-slate-700">
                {agent.status === "ONLINE"
                  ? "Agent is currently online"
                  : "Agent is currently offline"}
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {statistics.map((item) => {
          const Icon = item.icon;

          return (
            <div
              key={item.label}
              className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm"
            >
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-slate-500">
                    {item.label}
                  </p>

                  <p className="mt-2 text-2xl font-bold text-slate-900">
                    {item.value}
                  </p>
                </div>

                <div
                  className={`flex h-11 w-11 items-center justify-center rounded-lg ${item.background} ${item.color}`}
                >
                  <Icon size={21} />
                </div>
              </div>
            </div>
          );
        })}
      </section>

      <section className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="border-b border-slate-100 px-5 py-4">
          <h2 className="font-semibold text-slate-900">
            Monitored Targets
          </h2>
          <p className="mt-1 text-sm text-slate-500">
            Targets monitored by this agent
          </p>
        </div>

        {targets.data.length === 0 ? (
          <div className="flex min-h-48 flex-col items-center justify-center px-5 text-center">
            <Globe size={30} className="text-slate-300" />
            <p className="mt-3 text-sm font-medium text-slate-600">
              No targets found
            </p>
            <p className="mt-1 text-xs text-slate-400">
              This agent has no recorded monitoring targets.
            </p>
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-left">
                <thead className="bg-slate-50">
                  <tr>
                    <th className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">
                      Target
                    </th>
                    <th className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">
                      Status
                    </th>
                    <th className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">
                      Last Check
                    </th>
                    <th className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">
                      Latency
                    </th>
                    <th className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">
                      Protocol
                    </th>
                  </tr>
                </thead>

                <tbody className="divide-y divide-slate-100">
                  {targets.data.map((target) => (
                    <tr
                      key={`${target.probe_id}-${target.target}`}
                      className="transition hover:bg-slate-50"
                    >
                      <td className="px-5 py-4">
                        <Link
                          href={targetDetailUrl(target.target)}
                          className="text-sm font-medium text-blue-600 hover:text-blue-800 hover:underline"
                        >
                          {target.target}
                        </Link>

                        {target.error && (
                          <p className="mt-1 max-w-xs truncate text-xs text-red-500">
                            {target.error}
                          </p>
                        )}
                      </td>

                      <td className="px-5 py-4">
                        <StatusBadge status={target.status} />
                      </td>

                      <td className="px-5 py-4">
                        <div className="text-sm text-slate-700">
                          {formatDate(target.last_checked_at)}
                        </div>

                        <div className="mt-1 text-xs text-slate-400">
                          Check: {target.last_check_status}
                        </div>
                      </td>

                      <td className="px-5 py-4 text-sm text-slate-700">
                        {target.latency_ms !== null
                          ? `${target.latency_ms.toFixed(2)} ms`
                          : "—"}
                      </td>

                      <td className="px-5 py-4">
                        <span className="rounded bg-slate-100 px-2 py-1 text-xs font-medium text-slate-600">
                          {target.protocol || "—"}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="flex flex-col gap-3 border-t border-slate-100 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
              <p className="text-xs text-slate-500">
                Showing {start}–{end} of {summary.total} targets
              </p>

              <div className="flex items-center gap-3">
                <Link
                  href={paginationUrl(previousOffset)}
                  aria-disabled={targets.offset === 0}
                  className={`inline-flex items-center gap-1 rounded-lg border px-3 py-2 text-xs font-medium ${
                    targets.offset === 0
                      ? "pointer-events-none border-slate-100 text-slate-300"
                      : "border-slate-200 text-slate-600 hover:bg-slate-50"
                  }`}
                >
                  <ArrowLeft size={14} />
                  Previous
                </Link>

                <span className="text-xs text-slate-500">
                  Page {currentPage}
                </span>

                <Link
                  href={paginationUrl(nextOffset)}
                  aria-disabled={!targets.has_more}
                  className={`inline-flex items-center gap-1 rounded-lg border px-3 py-2 text-xs font-medium ${
                    !targets.has_more
                      ? "pointer-events-none border-slate-100 text-slate-300"
                      : "border-slate-200 text-slate-600 hover:bg-slate-50"
                  }`}
                >
                  Next
                  <ArrowRight size={14} />
                </Link>
              </div>
            </div>
          </>
        )}
      </section>
    </div>
  );
}
