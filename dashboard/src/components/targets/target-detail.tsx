"use client";

import TargetMetadataManager from "@/components/targets/target-metadata-manager";
import { useMemo, useState } from "react";
import Link from "next/link";
import {
  Activity,
  ArrowLeft,
  CheckCircle2,
  Clock3,
  CircleHelp,
  RefreshCw,
  XCircle,
} from "lucide-react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import type {
  MonitoredTarget,
  TargetCheck,
  TargetHistoryResponse,
  TargetPeriod,
} from "@/lib/api/types";

interface TargetDetailProps {
  target: MonitoredTarget;
  initialHistory: TargetHistoryResponse;
}

const PAGE_SIZE = 20;

type HistoryPeriod = TargetPeriod | "all";

function formatDate(value: string) {
  return new Intl.DateTimeFormat("id-ID", {
    dateStyle: "medium",
    timeStyle: "medium",
    timeZone: "Asia/Jakarta",
  }).format(new Date(value));
}

function StatusBadge({ status }: { status: string }) {
  const styles: Record<string, string> = {
    UP: "bg-emerald-50 text-emerald-700",
    DOWN: "bg-red-50 text-red-700",
    INVALID: "bg-amber-50 text-amber-700",
    UNKNOWN: "bg-slate-100 text-slate-600",
  };

  const icons = {
    UP: <CheckCircle2 size={14} />,
    DOWN: <XCircle size={14} />,
    INVALID: <CircleHelp size={14} />,
    UNKNOWN: <CircleHelp size={14} />,
  };

  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-semibold ${
        styles[status] ?? styles.UNKNOWN
      }`}
    >
      {icons[status as keyof typeof icons] ?? icons.UNKNOWN}
      {status}
    </span>
  );
}

export default function TargetDetail({
  target,
  initialHistory,
}: TargetDetailProps) {
  const [history, setHistory] = useState<TargetCheck[]>(initialHistory.data);
  const [period, setPeriod] = useState<HistoryPeriod>("all");
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(initialHistory.has_more);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);

  async function fetchHistory(
    selectedPeriod: HistoryPeriod,
    selectedOffset: number,
  ) {
    setLoading(true);
    setError(false);

    try {
      const query = new URLSearchParams({
        probe_id: target.probe_id,
        target: target.target,
        limit: String(PAGE_SIZE),
        offset: String(selectedOffset),
      });

      if (selectedPeriod !== "all") {
        query.set("period", selectedPeriod);
      }

      const response = await fetch(
        `/api/targets/history?${query.toString()}`,
        { cache: "no-store" },
      );

      if (!response.ok) {
        throw new Error("Failed to load history");
      }

      const result: TargetHistoryResponse = await response.json();

      setHistory(result.data);
      setHasMore(result.has_more);
    } catch {
      setError(true);
      setHistory([]);
      setHasMore(false);
    } finally {
      setLoading(false);
    }
  }

  function handlePeriodChange(value: HistoryPeriod) {
    setPeriod(value);
    setOffset(0);
    void fetchHistory(value, 0);
  }

  function handlePrevious() {
    const nextOffset = Math.max(0, offset - PAGE_SIZE);
    setOffset(nextOffset);
    void fetchHistory(period, nextOffset);
  }

  function handleNext() {
    const nextOffset = offset + PAGE_SIZE;
    setOffset(nextOffset);
    void fetchHistory(period, nextOffset);
  }

  const chartData = useMemo(
    () =>
      [...history]
        .filter(
          (item) =>
            item.latency_ms !== null &&
            Number.isFinite(item.latency_ms),
        )
        .reverse()
        .map((item) => ({
          time: new Date(item.checked_at).toLocaleString("id-ID", {
            day: "2-digit",
            month: "short",
            hour: "2-digit",
            minute: "2-digit",
            timeZone: "Asia/Jakarta",
          }),
          latency: item.latency_ms,
          status: item.status,
        })),
    [history],
  );

  const counts = useMemo(
    () => ({
      total: history.length,
      up: history.filter((item) => item.status === "UP").length,
      down: history.filter((item) => item.status === "DOWN").length,
      invalid: history.filter((item) => item.status === "INVALID").length,
    }),
    [history],
  );

  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/targets"
          className="mb-4 inline-flex items-center gap-2 text-sm font-medium text-slate-500 transition hover:text-blue-600"
        >
          <ArrowLeft size={16} />
          Back to Targets
        </Link>

        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <div className="flex items-center gap-3">
              <Activity className="text-blue-600" size={23} />
              <h1 className="break-all text-2xl font-bold text-slate-900">
                {target.target}
              </h1>
            </div>
            <p className="mt-2 text-sm text-slate-500">
              Probe ID: <span className="font-mono">{target.probe_id}</span>
            </p>
          </div>

          <StatusBadge status={target.status} />
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {[
          {
            label: "Operational Status",
            value: target.status,
            color:
              target.status === "UP"
                ? "text-emerald-600"
                : target.status === "DOWN"
                  ? "text-red-600"
                  : "text-slate-600",
          },
          {
            label: "Last Check",
            value: target.last_check_status,
            color: "text-slate-900",
          },
          {
            label: "Latency",
            value:
              target.latency_ms === null
                ? "—"
                : `${target.latency_ms.toFixed(2)} ms`,
            color: "text-slate-900",
          },
          {
            label: "Protocol",
            value: target.protocol,
            color: "text-slate-900",
          },
        ].map((item) => (
          <div
            key={item.label}
            className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm"
          >
            <p className="text-sm text-slate-500">{item.label}</p>
            <p className={`mt-3 text-xl font-bold ${item.color}`}>
              {item.value}
            </p>
          </div>
        ))}
      </div>

      {target.error && (
        <div className="rounded-xl border border-red-200 bg-red-50 p-4">
          <p className="text-sm font-semibold text-red-700">
            Last recorded error
          </p>
          <p className="mt-1 break-all text-sm text-red-600">
            {target.error}
          </p>
        </div>
      )}

      <TargetMetadataManager
      probeId={target.probe_id}
      target={target.target}
      />

      <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
        <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="font-semibold text-slate-900">
              Latency History
            </h2>
            <p className="mt-1 text-xs text-slate-500">
              Response time across recorded checks.
            </p>
          </div>

          <select
            value={period}
            onChange={(event) =>
              handlePeriodChange(event.target.value as HistoryPeriod)
            }
            disabled={loading}
            className="h-10 rounded-lg border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none focus:border-blue-500"
          >
            <option value="all">All time</option>
            <option value="1h">Last hour</option>
            <option value="24h">Last 24 hours</option>
            <option value="7d">Last 7 days</option>
            <option value="30d">Last 30 days</option>
          </select>
        </div>

        {loading ? (
          <div className="flex h-72 items-center justify-center text-sm text-slate-500">
            Loading chart...
          </div>
        ) : chartData.length === 0 ? (
          <div className="flex h-72 flex-col items-center justify-center text-slate-400">
            <Activity size={30} />
            <p className="mt-3 text-sm">No latency data available.</p>
          </div>
        ) : (
          <div className="h-72 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart
                data={chartData}
                margin={{ top: 10, right: 20, left: 0, bottom: 5 }}
              >
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis
                  dataKey="time"
                  tick={{ fontSize: 11 }}
                  minTickGap={25}
                />
                <YAxis
                  tick={{ fontSize: 11 }}
                  width={65}
                  label={{
                    value: "Latency (ms)",
                    angle: -90,
                    position: "insideLeft",
                    style: { fontSize: 11 },
                  }}
                />
                <Tooltip
                  formatter={(value) => [
                    `${Number(value).toFixed(2)} ms`,
                    "Latency",
                  ]}
                />
                <Line
                  type="monotone"
                  dataKey="latency"
                  stroke="#2563eb"
                  strokeWidth={2}
                  dot={{ r: 3 }}
                  activeDot={{ r: 5 }}
                  connectNulls={false}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}
      </section>

      <section className="rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="flex flex-col gap-3 border-b border-slate-200 p-5 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="font-semibold text-slate-900">
              Check History
            </h2>
            <p className="mt-1 text-xs text-slate-500">
              Individual monitoring results for this target.
            </p>
          </div>

          <button
            onClick={() => void fetchHistory(period, offset)}
            disabled={loading}
            className="inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 px-3 py-2 text-xs font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50"
          >
            <RefreshCw size={14} />
            Refresh
          </button>
        </div>

        <div className="grid grid-cols-2 gap-3 border-b border-slate-100 p-5 sm:grid-cols-4">
          {[
            { label: "Loaded Checks", value: counts.total },
            { label: "UP", value: counts.up },
            { label: "DOWN", value: counts.down },
            { label: "INVALID", value: counts.invalid },
          ].map((item) => (
            <div key={item.label}>
              <p className="text-xs text-slate-500">{item.label}</p>
              <p className="mt-1 text-xl font-bold text-slate-900">
                {item.value}
              </p>
            </div>
          ))}
        </div>

        {error ? (
          <div className="p-10 text-center">
            <p className="text-sm text-red-600">
              Failed to load check history.
            </p>
            <button
              onClick={() => void fetchHistory(period, offset)}
              className="mt-3 text-sm font-semibold text-blue-600"
            >
              Try again
            </button>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[750px] text-left text-sm">
              <thead className="bg-slate-50 text-xs uppercase text-slate-500">
                <tr>
                  <th className="px-5 py-4">Status</th>
                  <th className="px-5 py-4">Checked At</th>
                  <th className="px-5 py-4">Latency</th>
                  <th className="px-5 py-4">HTTP Code</th>
                  <th className="px-5 py-4">Error</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {loading ? (
                  <tr>
                    <td colSpan={5} className="p-10 text-center text-slate-500">
                      Loading history...
                    </td>
                  </tr>
                ) : history.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="p-10 text-center text-slate-500">
                      No check history found.
                    </td>
                  </tr>
                ) : (
                  history.map((item) => (
                    <tr key={item.id} className="hover:bg-slate-50">
                      <td className="px-5 py-4">
                        <StatusBadge status={item.status} />
                      </td>
                      <td className="px-5 py-4 text-xs text-slate-600">
                        <span className="inline-flex items-center gap-1.5">
                          <Clock3 size={13} />
                          {formatDate(item.checked_at)}
                        </span>
                      </td>
                      <td className="px-5 py-4 font-mono text-xs text-slate-700">
                        {item.latency_ms === null
                          ? "—"
                          : `${item.latency_ms.toFixed(2)} ms`}
                      </td>
                      <td className="px-5 py-4 text-xs text-slate-600">
                        {item.status_code ?? "—"}
                      </td>
                      <td className="max-w-xs truncate px-5 py-4 text-xs text-red-500">
                        {item.error || "—"}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        )}

        <div className="flex items-center justify-between border-t border-slate-200 px-5 py-4">
          <p className="text-xs text-slate-500">
            Showing {history.length} checks · Offset {offset}
          </p>
          <div className="flex gap-2">
            <button
              disabled={offset === 0 || loading}
              onClick={handlePrevious}
              className="rounded-lg border border-slate-200 px-3 py-2 text-xs disabled:opacity-40"
            >
              Previous
            </button>
            <button
              disabled={!hasMore || loading}
              onClick={handleNext}
              className="rounded-lg border border-slate-200 px-3 py-2 text-xs disabled:opacity-40"
            >
              Next
            </button>
          </div>
        </div>
      </section>
    </div>
  );
}
