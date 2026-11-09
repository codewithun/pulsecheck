
"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import {
  Activity,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  CircleHelp,
  Clock3,
  Search,
  Server,
  XCircle,
} from "lucide-react";

import type {
  MonitoredTarget,
  TargetPeriod,
  TargetStatus,
  TargetsResponse,
} from "@/lib/api/types";

const PAGE_SIZE = 10;

type FilterStatus = "ALL" | TargetStatus;

interface TargetsTableProps {
  initialTargets: TargetsResponse;
  apiError: boolean;
}

function formatDate(value: string | null) {
  if (!value) return "Never";

  return new Intl.DateTimeFormat("id-ID", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "Asia/Jakarta",
  }).format(new Date(value));
}

function formatLatency(value: number | null) {
  if (value === null) return "—";
  return `${value.toFixed(2)} ms`;
}

function StatusBadge({ status }: { status: TargetStatus }) {
  const styles = {
    UP: "bg-emerald-50 text-emerald-700 ring-emerald-600/20",
    DOWN: "bg-red-50 text-red-700 ring-red-600/20",
    UNKNOWN: "bg-slate-100 text-slate-600 ring-slate-500/20",
  };

  const icons = {
    UP: <CheckCircle2 size={14} />,
    DOWN: <XCircle size={14} />,
    UNKNOWN: <CircleHelp size={14} />,
  };

  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-semibold ring-1 ring-inset ${styles[status]}`}
    >
      {icons[status]}
      {status}
    </span>
  );
}

export default function TargetsTable({
  initialTargets,
  apiError,
}: TargetsTableProps) {
  const [targets, setTargets] = useState<MonitoredTarget[]>(
    initialTargets.data,
  );
  const [status, setStatus] = useState<FilterStatus>("ALL");
  const [probeId, setProbeId] = useState("ALL");
  const [period, setPeriod] = useState<TargetPeriod | "all">("all");
  const [search, setSearch] = useState("");
  const [offset, setOffset] = useState(initialTargets.offset);
  const [hasMore, setHasMore] = useState(initialTargets.has_more);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(apiError);

  const fetchTargets = useCallback(async () => {
    setLoading(true);
    setError(false);

    try {
      const query = new URLSearchParams({
        limit: String(PAGE_SIZE),
        offset: String(offset),
      });

      if (status !== "ALL") query.set("status", status);
      if (probeId !== "ALL") query.set("probe_id", probeId);
      if (period !== "all") query.set("period", period);
      if (search.trim()) query.set("search", search.trim());

      const response = await fetch(`/api/targets?${query.toString()}`, {
        cache: "no-store",
      });

      if (!response.ok) {
        throw new Error("Failed to load targets");
      }

      const result: TargetsResponse = await response.json();

      setTargets(result.data);
      setHasMore(result.has_more);
    } catch {
      setError(true);
      setTargets([]);
      setHasMore(false);
    } finally {
      setLoading(false);
    }
  }, [offset, status, probeId, period, search]);

  useEffect(() => {
    const timer = setTimeout(() => {
      void fetchTargets();
    }, search ? 350 : 0);

    return () => clearTimeout(timer);
  }, [fetchTargets, search]);

  const counts = useMemo(
    () => ({
      total: targets.length,
      up: targets.filter((target) => target.status === "UP").length,
      down: targets.filter((target) => target.status === "DOWN").length,
      unknown: targets.filter((target) => target.status === "UNKNOWN").length,
    }),
    [targets],
  );

  const probeOptions = useMemo(
    () => [...new Set(initialTargets.data.map((target) => target.probe_id))],
    [initialTargets.data],
  );

  function changeFilter(callback: () => void) {
    callback();
    setOffset(0);
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <Server className="text-blue-600" size={22} />
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">
              Targets
            </h1>
          </div>
          <p className="mt-1 text-sm text-slate-500">
            Monitor and manage the health of your monitored endpoints.
          </p>
        </div>

        <span className="inline-flex w-fit items-center gap-2 rounded-full bg-blue-50 px-3 py-1.5 text-xs font-semibold text-blue-700">
          <Activity size={14} />
          LIVE DATA
        </span>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {[
          {
            label: "Loaded Targets",
            value: counts.total,
            icon: <Server size={19} />,
            color: "text-blue-600 bg-blue-50",
          },
          {
            label: "UP",
            value: counts.up,
            icon: <CheckCircle2 size={19} />,
            color: "text-emerald-600 bg-emerald-50",
          },
          {
            label: "DOWN",
            value: counts.down,
            icon: <XCircle size={19} />,
            color: "text-red-600 bg-red-50",
          },
          {
            label: "UNKNOWN",
            value: counts.unknown,
            icon: <CircleHelp size={19} />,
            color: "text-slate-600 bg-slate-100",
          },
        ].map((item) => (
          <div
            key={item.label}
            className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm"
          >
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium text-slate-500">
                {item.label}
              </p>
              <span className={`rounded-lg p-2 ${item.color}`}>
                {item.icon}
              </span>
            </div>
            <p className="mt-3 text-3xl font-bold text-slate-900">
              {item.value}
            </p>
          </div>
        ))}
      </div>

      <div className="rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="flex flex-col gap-4 border-b border-slate-200 p-5 xl:flex-row xl:items-center xl:justify-between">
          <div>
            <h2 className="font-semibold text-slate-900">
              Monitored Endpoints
            </h2>
            <p className="mt-1 text-xs text-slate-500">
              Search, filter, and inspect your monitored targets.
            </p>
          </div>

          <div className="grid gap-3 sm:grid-cols-2 xl:flex">
            <div className="relative">
              <Search
                size={16}
                className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
              />
              <input
                value={search}
                onChange={(event) =>
                  changeFilter(() => {
                    setSearch(event.target.value);
                  })
                }
                placeholder="Search target..."
                className="h-10 w-full rounded-lg border border-slate-200 bg-white pl-9 pr-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100 xl:w-56"
              />
            </div>

            <select
              value={status}
              onChange={(event) =>
                changeFilter(() =>
                  setStatus(event.target.value as FilterStatus),
                )
              }
              className="h-10 rounded-lg border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none focus:border-blue-500"
            >
              <option value="ALL">All statuses</option>
              <option value="UP">UP</option>
              <option value="DOWN">DOWN</option>
              <option value="UNKNOWN">UNKNOWN</option>
            </select>

            <select
              value={probeId}
              onChange={(event) =>
                changeFilter(() => setProbeId(event.target.value))
              }
              className="h-10 rounded-lg border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none focus:border-blue-500"
            >
              <option value="ALL">All probes</option>
              {probeOptions.map((id) => (
                <option key={id} value={id}>
                  {id}
                </option>
              ))}
            </select>

            <select
              value={period}
              onChange={(event) =>
                changeFilter(() =>
                  setPeriod(event.target.value as TargetPeriod | "all"),
                )
              }
              className="h-10 rounded-lg border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none focus:border-blue-500"
            >
              <option value="all">All time</option>
              <option value="1h">Last hour</option>
              <option value="24h">Last 24 hours</option>
              <option value="7d">Last 7 days</option>
              <option value="30d">Last 30 days</option>
            </select>
          </div>
        </div>

        {error ? (
          <div className="p-12 text-center">
            <XCircle className="mx-auto text-red-500" size={30} />
            <p className="mt-3 font-medium text-slate-900">
              Failed to load targets
            </p>
            <p className="mt-1 text-sm text-slate-500">
              Please check the Central API connection and try again.
            </p>
            <button
              onClick={() => void fetchTargets()}
              className="mt-4 rounded-lg bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700"
            >
              Retry
            </button>
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[850px] text-left text-sm">
                <thead className="bg-slate-50 text-xs uppercase tracking-wide text-slate-500">
                  <tr>
                    <th className="px-5 py-4 font-semibold">Target</th>
                    <th className="px-5 py-4 font-semibold">Probe ID</th>
                    <th className="px-5 py-4 font-semibold">Status</th>
                    <th className="px-5 py-4 font-semibold">Last Check</th>
                    <th className="px-5 py-4 font-semibold">Latency</th>
                    <th className="px-5 py-4 font-semibold">Protocol</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {loading ? (
                    <tr>
                      <td
                        colSpan={6}
                        className="px-5 py-12 text-center text-slate-500"
                      >
                        Loading targets...
                      </td>
                    </tr>
                  ) : targets.length === 0 ? (
                    <tr>
                      <td
                        colSpan={6}
                        className="px-5 py-12 text-center text-slate-500"
                      >
                        No targets found.
                      </td>
                    </tr>
                  ) : (
                    targets.map((target) => (
                      <tr
                        key={`${target.probe_id}-${target.target}`}
                        className="transition hover:bg-slate-50"
                      >
                        <td className="px-5 py-4">
                         <Link
                              href={`/targets/detail?probe_id=${encodeURIComponent(target.probe_id)}&target=${encodeURIComponent(target.target)}`}
                              className="font-medium text-blue-600 transition hover:text-blue-800 hover:underline"
                            >
                              {target.target}
                            </Link>
                          {target.error && (
                            <p className="mt-1 max-w-xs truncate text-xs text-red-500">
                              {target.error}
                            </p>
                          )}
                        </td>
                        <td className="px-5 py-4 font-mono text-xs text-slate-600">
                          {target.probe_id}
                        </td>
                        <td className="px-5 py-4">
                          <StatusBadge status={target.status} />
                          {target.last_check_status !== target.status && (
                            <p className="mt-1 text-[11px] text-slate-400">
                              Last check: {target.last_check_status}
                            </p>
                          )}
                        </td>
                        <td className="px-5 py-4 text-xs text-slate-600">
                          <span className="inline-flex items-center gap-1.5">
                            <Clock3 size={13} />
                            {formatDate(target.last_checked_at)}
                          </span>
                        </td>
                        <td className="px-5 py-4 font-mono text-xs text-slate-700">
                          {formatLatency(target.latency_ms)}
                        </td>
                        <td className="px-5 py-4">
                          <span className="rounded-md bg-slate-100 px-2 py-1 text-xs font-medium text-slate-600">
                            {target.protocol}
                          </span>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>

            <div className="flex flex-col gap-3 border-t border-slate-200 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
              <p className="text-xs text-slate-500">
                Showing {targets.length} targets · Offset {offset}
              </p>

              <div className="flex items-center gap-2">
                <button
                  disabled={offset === 0 || loading}
                  onClick={() =>
                    setOffset((current) =>
                      Math.max(0, current - PAGE_SIZE),
                    )
                  }
                  className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-2 text-xs font-medium text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40"
                >
                  <ChevronLeft size={15} />
                  Previous
                </button>

                <button
                  disabled={!hasMore || loading}
                  onClick={() =>
                    setOffset((current) => current + PAGE_SIZE)
                  }
                  className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-2 text-xs font-medium text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40"
                >
                  Next
                  <ChevronRight size={15} />
                </button>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
