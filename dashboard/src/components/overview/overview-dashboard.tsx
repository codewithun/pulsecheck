
"use client";

import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Server,
  Target,
} from "lucide-react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

const stats = [
  {
    label: "Total Agents",
    value: "12",
    detail: "10 online · 2 offline",
    icon: Server,
    color: "blue",
  },
  {
    label: "Active Targets",
    value: "48",
    detail: "45 UP · 3 DOWN",
    icon: Target,
    color: "violet",
  },
  {
    label: "Availability",
    value: "99.82%",
    detail: "Across monitored checks",
    icon: Activity,
    color: "emerald",
  },
  {
    label: "Pending Alerts",
    value: "3",
    detail: "Require attention",
    icon: AlertTriangle,
    color: "amber",
  },
];

const chartData = [
  { time: "00:00", availability: 99.9 },
  { time: "02:00", availability: 100 },
  { time: "04:00", availability: 99.8 },
  { time: "06:00", availability: 99.9 },
  { time: "08:00", availability: 99.7 },
  { time: "10:00", availability: 100 },
  { time: "12:00", availability: 99.8 },
  { time: "14:00", availability: 99.9 },
  { time: "16:00", availability: 99.6 },
  { time: "18:00", availability: 99.9 },
  { time: "20:00", availability: 100 },
  { time: "22:00", availability: 99.9 },
];

const targetStatus = [
  { label: "Operational", count: 45, color: "bg-emerald-500" },
  { label: "Down", count: 3, color: "bg-red-500" },
  { label: "Unknown", count: 0, color: "bg-slate-300" },
];

const colorStyles: Record<string, string> = {
  blue: "bg-blue-50 text-blue-600",
  violet: "bg-violet-50 text-violet-600",
  emerald: "bg-emerald-50 text-emerald-600",
  amber: "bg-amber-50 text-amber-600",
};

export default function OverviewDashboard() {
  return (
    <div className="space-y-6">
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

        <span className="rounded-full border border-amber-200 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-700">
          DEMO DATA
        </span>
      </div>

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
              <p className="mt-2 text-xs text-slate-400">{stat.detail}</p>
            </div>
          );
        })}
      </div>

      <div className="grid gap-6 xl:grid-cols-3">
        <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm xl:col-span-2">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="font-semibold text-slate-900">
                Availability Trend
              </h2>
              <p className="mt-1 text-xs text-slate-400">
                Observed availability over the last 24 hours
              </p>
            </div>
            <span className="rounded-lg bg-slate-50 px-3 py-1.5 text-xs font-medium text-slate-500">
              Last 24 hours
            </span>
          </div>

          <div className="mt-6 h-[280px] w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart
                data={chartData}
                margin={{ top: 10, right: 10, left: -20, bottom: 0 }}
              >
                <defs>
                  <linearGradient
                    id="availabilityGradient"
                    x1="0"
                    y1="0"
                    x2="0"
                    y2="1"
                  >
                    <stop
                      offset="0%"
                      stopColor="#3b82f6"
                      stopOpacity={0.2}
                    />
                    <stop
                      offset="95%"
                      stopColor="#3b82f6"
                      stopOpacity={0}
                    />
                  </linearGradient>
                </defs>

                <CartesianGrid
                  strokeDasharray="3 3"
                  vertical={false}
                  stroke="#e9eef5"
                />
                <XAxis
                  dataKey="time"
                  tickLine={false}
                  axisLine={false}
                  tick={{ fill: "#94a3b8", fontSize: 11 }}
                  dy={10}
                />
                <YAxis
                  domain={[99, 100]}
                  tickLine={false}
                  axisLine={false}
                  tick={{ fill: "#94a3b8", fontSize: 11 }}
                  tickFormatter={(value) => `${value}%`}
                />
                <Tooltip
                  formatter={(value) => [`${value}%`, "Availability"]}
                  contentStyle={{
                    borderRadius: 10,
                    border: "1px solid #e2e8f0",
                    fontSize: 12,
                  }}
                />
                <Area
                  type="monotone"
                  dataKey="availability"
                  stroke="#3b82f6"
                  strokeWidth={2.5}
                  fill="url(#availabilityGradient)"
                  activeDot={{ r: 5 }}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </section>

        <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <h2 className="font-semibold text-slate-900">Target Status</h2>
          <p className="mt-1 text-xs text-slate-400">
            Current operational distribution
          </p>

          <div className="mt-7 flex items-center justify-center">
            <div className="relative flex h-40 w-40 items-center justify-center rounded-full border-[14px] border-emerald-100">
              <div className="absolute inset-0 rounded-full border-[14px] border-transparent border-t-red-400 rotate-[35deg]" />
              <div className="text-center">
                <p className="text-3xl font-bold text-slate-900">48</p>
                <p className="text-xs text-slate-400">Total targets</p>
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
                  <span className={`h-2.5 w-2.5 rounded-full ${item.color}`} />
                  <span className="text-sm text-slate-600">{item.label}</span>
                </div>
                <span className="text-sm font-semibold text-slate-900">
                  {item.count}
                </span>
              </div>
            ))}
          </div>
        </section>
      </div>

      <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="font-semibold text-slate-900">
              System Health Summary
            </h2>
            <p className="mt-1 text-xs text-slate-400">
              Example operational indicators
            </p>
          </div>
          <CheckCircle2 className="text-emerald-500" size={21} />
        </div>

        <div className="mt-5 grid gap-4 sm:grid-cols-3">
          <div className="rounded-lg bg-emerald-50 p-4">
            <p className="text-xs text-emerald-700">Healthy targets</p>
            <p className="mt-1 text-xl font-bold text-emerald-800">45</p>
          </div>
          <div className="rounded-lg bg-red-50 p-4">
            <p className="text-xs text-red-700">Failed targets</p>
            <p className="mt-1 text-xl font-bold text-red-800">3</p>
          </div>
          <div className="rounded-lg bg-slate-50 p-4">
            <p className="text-xs text-slate-600">Unknown targets</p>
            <p className="mt-1 text-xl font-bold text-slate-800">0</p>
          </div>
        </div>
      </section>
    </div>
  );
}
