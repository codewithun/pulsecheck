
"use client";

import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Server,
  Target,
  XCircle,
  CircleHelp,
} from "lucide-react";

import type { OverviewResponse } from "@/lib/api/types";

interface OverviewDashboardProps {
  overview: OverviewResponse | null;
  apiError: boolean;
}

const colorStyles = {
  blue: "bg-blue-50 text-blue-600",
  violet: "bg-violet-50 text-violet-600",
  emerald: "bg-emerald-50 text-emerald-600",
  amber: "bg-amber-50 text-amber-600",
};

export default function OverviewDashboard({
  overview,
  apiError,
}: OverviewDashboardProps) {
  const stats = [
    {
      label: "Total Agents",
      value: overview?.agents.total ?? "—",
      detail: overview
        ? `${overview.agents.online} online · ${overview.agents.offline} offline`
        : "Data unavailable",
      icon: Server,
      color: "blue" as const,
    },
    {
      label: "Total Targets",
      value: overview?.targets.total ?? "—",
      detail: overview
        ? `${overview.targets.up} UP · ${overview.targets.down} DOWN`
        : "Data unavailable",
      icon: Target,
      color: "violet" as const,
    },
    {
      label: "Availability",
      value:
        overview?.availability != null
          ? `${overview.availability.toFixed(2)}%`
          : "N/A",
      detail: overview
        ? `${overview.checks.up + overview.checks.down} valid checks`
        : "Data unavailable",
      icon: Activity,
      color: "emerald" as const,
    },
    {
      label: "Pending Alerts",
      value: overview?.alerts.pending ?? "—",
      detail: "Awaiting delivery",
      icon: AlertTriangle,
      color: "amber" as const,
    },
  ];

  const targetStatus = [
    {
      label: "Operational",
      count: overview?.targets.up ?? 0,
      color: "bg-emerald-500",
    },
    {
      label: "Down",
      count: overview?.targets.down ?? 0,
      color: "bg-red-500",
    },
    {
      label: "Unknown",
      count: overview?.targets.unknown ?? 0,
      color: "bg-slate-300",
    },
  ];

  return (
    <div className="space-y-6">
      {/* Page heading */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-blue-600">
            Workspace / Overview
          </p>

          <h1 className="mt-1 text-2xl font-bold tracking-tight text-slate-900">
            Overview
          </h1>

          <p className="mt-1 text-sm text-slate-500">
            Monitor the health and performance of your infrastructure.
          </p>
        </div>

        <span
          className={`rounded-full border px-3 py-1.5 text-xs font-semibold ${
            apiError
              ? "border-red-200 bg-red-50 text-red-700"
              : "border-emerald-200 bg-emerald-50 text-emerald-700"
          }`}
        >
          {apiError ? "API UNAVAILABLE" : "LIVE DATA"}
        </span>
      </div>

      {/* API error */}
      {apiError && (
        <div className="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4">
          <AlertTriangle className="mt-0.5 text-red-600" size={20} />

          <div>
            <p className="text-sm font-semibold text-red-800">
              Unable to retrieve monitoring data
            </p>

            <p className="mt-1 text-sm text-red-700">
              Central API could not be reached. Check the backend service and
              try refreshing the page.
            </p>
          </div>
        </div>
      )}

      {/* Statistics */}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon;

          return (
            <div
              key={stat.label}
              className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm"
            >
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium text-slate-500">
                  {stat.label}
                </p>

                <div
                  className={`flex h-10 w-10 items-center justify-center rounded-xl ${colorStyles[stat.color]}`}
                >
                  <Icon size={19} />
                </div>
              </div>

              <p className="mt-4 text-3xl font-bold tracking-tight text-slate-900">
                {stat.value}
              </p>

              <p className="mt-2 text-xs text-slate-400">
                {stat.detail}
              </p>
            </div>
          );
        })}
      </div>

      {/* Monitoring overview */}
      <div className="grid gap-6 xl:grid-cols-3">
        {/* Check statistics */}
        <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm xl:col-span-2">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="font-semibold text-slate-900">
                Monitoring Activity
              </h2>

              <p className="mt-1 text-xs text-slate-400">
                Check statistics for the selected monitoring period
              </p>
            </div>

            <span className="rounded-lg bg-slate-50 px-3 py-1.5 text-xs font-medium text-slate-500">
              Last {overview?.period ?? "24h"}
            </span>
          </div>

          <div className="mt-6">
            {overview ? (
              <>
                <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                  <div className="rounded-lg bg-slate-50 p-4">
                    <p className="text-xs text-slate-500">Total Checks</p>
                    <p className="mt-2 text-2xl font-bold text-slate-900">
                      {overview.checks.total}
                    </p>
                  </div>

                  <div className="rounded-lg bg-emerald-50 p-4">
                    <p className="text-xs text-emerald-700">Successful</p>
                    <p className="mt-2 text-2xl font-bold text-emerald-800">
                      {overview.checks.up}
                    </p>
                  </div>

                  <div className="rounded-lg bg-red-50 p-4">
                    <p className="text-xs text-red-700">Failed</p>
                    <p className="mt-2 text-2xl font-bold text-red-800">
                      {overview.checks.down}
                    </p>
                  </div>

                  <div className="rounded-lg bg-amber-50 p-4">
                    <p className="text-xs text-amber-700">Invalid</p>
                    <p className="mt-2 text-2xl font-bold text-amber-800">
                      {overview.checks.invalid}
                    </p>
                  </div>
                </div>

                <div className="mt-6 rounded-lg border border-slate-100 p-4">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="text-sm font-semibold text-slate-800">
                        Observed Availability
                      </p>

                      <p className="mt-1 text-xs text-slate-400">
                        Based on UP and DOWN checks only
                      </p>
                    </div>

                    <Activity className="text-blue-500" size={22} />
                  </div>

                  <p className="mt-3 text-3xl font-bold text-slate-900">
                    {overview.availability != null
                      ? `${overview.availability.toFixed(2)}%`
                      : "N/A"}
                  </p>

                  {overview.availability == null && (
                    <p className="mt-2 text-xs text-slate-500">
                      Not enough valid monitoring checks to calculate
                      availability.
                    </p>
                  )}
                </div>
              </>
            ) : (
              <div className="flex h-56 flex-col items-center justify-center text-center">
                <CircleHelp className="text-slate-300" size={32} />

                <p className="mt-3 text-sm font-medium text-slate-600">
                  Monitoring statistics unavailable
                </p>
              </div>
            )}
          </div>
        </section>

        {/* Target status */}
        <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <h2 className="font-semibold text-slate-900">
            Target Status
          </h2>

          <p className="mt-1 text-xs text-slate-400">
            Current operational distribution
          </p>

          <div className="mt-7 flex items-center justify-center">
            <div className="relative flex h-40 w-40 items-center justify-center rounded-full border-[14px] border-slate-100">
              <div className="text-center">
                <p className="text-3xl font-bold text-slate-900">
                  {overview?.targets.total ?? "—"}
                </p>

                <p className="text-xs text-slate-400">
                  Total targets
                </p>
              </div>
            </div>
          </div>

          <div className="mt-7 space-y-4">
            {targetStatus.map((item) => (
              <div
                key={item.label}
                className="flex items-center justify-between"
              >
                <div className="flex items-center gap-2">
                  <span
                    className={`h-2.5 w-2.5 rounded-full ${item.color}`}
                  />

                  <span className="text-sm text-slate-600">
                    {item.label}
                  </span>
                </div>

                <span className="text-sm font-semibold text-slate-900">
                  {overview ? item.count : "—"}
                </span>
              </div>
            ))}
          </div>
        </section>
      </div>

      {/* System health */}
      <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="font-semibold text-slate-900">
              System Health Summary
            </h2>

            <p className="mt-1 text-xs text-slate-400">
              Current monitoring and alert status
            </p>
          </div>

          {apiError ? (
            <XCircle className="text-red-500" size={21} />
          ) : (
            <CheckCircle2 className="text-emerald-500" size={21} />
          )}
        </div>

        <div className="mt-5 grid gap-4 sm:grid-cols-3">
          <div className="rounded-lg bg-emerald-50 p-4">
            <p className="text-xs text-emerald-700">
              Healthy targets
            </p>

            <p className="mt-1 text-xl font-bold text-emerald-800">
              {overview?.targets.up ?? "—"}
            </p>
          </div>

          <div className="rounded-lg bg-red-50 p-4">
            <p className="text-xs text-red-700">
              Failed targets
            </p>

            <p className="mt-1 text-xl font-bold text-red-800">
              {overview?.targets.down ?? "—"}
            </p>
          </div>

          <div className="rounded-lg bg-slate-50 p-4">
            <p className="text-xs text-slate-600">
              Unknown targets
            </p>

            <p className="mt-1 text-xl font-bold text-slate-800">
              {overview?.targets.unknown ?? "—"}
            </p>
          </div>
        </div>

        <div className="mt-5 flex flex-wrap gap-6 border-t border-slate-100 pt-4">
          <div className="flex items-center gap-2">
            <CheckCircle2 size={16} className="text-emerald-500" />

            <span className="text-xs text-slate-500">
              {overview?.alerts.delivered ?? "—"} alerts delivered
            </span>
          </div>

          <div className="flex items-center gap-2">
            <AlertTriangle size={16} className="text-amber-500" />

            <span className="text-xs text-slate-500">
              {overview?.alerts.pending ?? "—"} alerts pending
            </span>
          </div>
        </div>
      </section>
    </div>
  );
}
