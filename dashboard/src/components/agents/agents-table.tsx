
import {
  CircleCheck,
  CircleX,
  Server,
} from "lucide-react";

import type { Agent } from "@/lib/api/types";

interface AgentsTableProps {
  agents: Agent[];
  apiError: boolean;
}

function formatLastSeen(value: string): string {
  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "Invalid date";
  }

  return new Intl.DateTimeFormat("id-ID", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "Asia/Jakarta",
  }).format(date);
}

function getTimeAgo(value: string): string {
  const date = new Date(value);
  const timestamp = date.getTime();

  if (Number.isNaN(timestamp)) {
    return "Unknown";
  }

  const difference = Date.now() - timestamp;
  const minutes = Math.floor(difference / 60000);

  if (minutes < 1) return "Baru saja";
  if (minutes < 60) return `${minutes} menit lalu`;

  const hours = Math.floor(minutes / 60);

  if (hours < 24) return `${hours} jam lalu`;

  const days = Math.floor(hours / 24);

  return `${days} hari lalu`;
}

export default function AgentsTable({
  agents,
  apiError,
}: AgentsTableProps) {
  const onlineCount = agents.filter(
    (agent) => agent.status === "ONLINE",
  ).length;

  const offlineCount = agents.length - onlineCount;

  return (
    <div className="space-y-6">
      {/* Page heading */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-blue-600">
            Workspace / Agents
          </p>

          <h1 className="mt-1 text-2xl font-bold tracking-tight text-slate-900">
            Agents
          </h1>

          <p className="mt-1 text-sm text-slate-500">
            Monitor registered probe agents and their heartbeat status.
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
        <div className="rounded-xl border border-red-200 bg-red-50 p-4">
          <p className="text-sm font-semibold text-red-800">
            Unable to retrieve agent data
          </p>

          <p className="mt-1 text-sm text-red-700">
            Check the Central API service and try refreshing this page.
          </p>
        </div>
      )}

      {/* Summary cards */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium text-slate-500">
              Total Agents
            </p>
            <Server className="text-blue-600" size={20} />
          </div>

          <p className="mt-4 text-3xl font-bold text-slate-900">
            {apiError ? "—" : agents.length}
          </p>

          <p className="mt-1 text-xs text-slate-400">
            Registered probe agents
          </p>
        </div>

        <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium text-slate-500">
              Online
            </p>
            <CircleCheck className="text-emerald-600" size={20} />
          </div>

          <p className="mt-4 text-3xl font-bold text-emerald-600">
            {apiError ? "—" : onlineCount}
          </p>

          <p className="mt-1 text-xs text-slate-400">
            Active agents
          </p>
        </div>

        <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium text-slate-500">
              Offline
            </p>
            <CircleX className="text-red-600" size={20} />
          </div>

          <p className="mt-4 text-3xl font-bold text-red-600">
            {apiError ? "—" : offlineCount}
          </p>

          <p className="mt-1 text-xs text-slate-400">
            Inactive agents
          </p>
        </div>
      </div>

      {/* Agents table */}
      <section className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="border-b border-slate-200 px-5 py-4">
          <h2 className="font-semibold text-slate-900">
            Registered Agents
          </h2>

          <p className="mt-1 text-xs text-slate-400">
            Agent identity and latest heartbeat information
          </p>
        </div>

        {apiError ? (
          <div className="p-10 text-center">
            <p className="text-sm text-slate-500">
              Agent data is currently unavailable.
            </p>
          </div>
        ) : agents.length === 0 ? (
          <div className="p-10 text-center">
            <Server className="mx-auto text-slate-300" size={32} />

            <p className="mt-3 text-sm font-medium text-slate-700">
              No agents registered
            </p>

            <p className="mt-1 text-xs text-slate-400">
              Registered probe agents will appear here.
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[700px] text-left">
              <thead className="bg-slate-50">
                <tr>
                  <th className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Agent
                  </th>

                  <th className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Probe ID
                  </th>

                  <th className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Last Heartbeat
                  </th>

                  <th className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Status
                  </th>
                </tr>
              </thead>

              <tbody className="divide-y divide-slate-100">
                {agents.map((agent) => {
                  const isOnline = agent.status === "ONLINE";

                  return (
                    <tr
                      key={agent.probe_id}
                      className="transition-colors hover:bg-slate-50"
                    >
                      <td className="px-5 py-4">
                        <div className="flex items-center gap-3">
                          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-blue-50 text-blue-600">
                            <Server size={17} />
                          </div>

                          <div>
                            <p className="text-sm font-semibold text-slate-800">
                              {agent.hostname}
                            </p>

                            <p className="text-xs text-slate-400">
                              Probe Agent
                            </p>
                          </div>
                        </div>
                      </td>

                      <td className="px-5 py-4">
                        <code className="rounded bg-slate-100 px-2 py-1 text-xs text-slate-600">
                          {agent.probe_id}
                        </code>
                      </td>

                      <td className="px-5 py-4">
                        <p className="text-sm text-slate-700">
                          {formatLastSeen(agent.last_seen)}
                        </p>

                        <p className="mt-1 text-xs text-slate-400">
                          {getTimeAgo(agent.last_seen)}
                        </p>
                      </td>

                      <td className="px-5 py-4">
                        <span
                          className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-semibold ${
                            isOnline
                              ? "bg-emerald-50 text-emerald-700"
                              : "bg-red-50 text-red-700"
                          }`}
                        >
                          <span
                            className={`h-1.5 w-1.5 rounded-full ${
                              isOnline
                                ? "bg-emerald-500"
                                : "bg-red-500"
                            }`}
                          />

                          {agent.status}
                        </span>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
