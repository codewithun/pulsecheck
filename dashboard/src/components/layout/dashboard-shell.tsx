
"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";
import {
  Activity,
  Bell,
  CircleHelp,
  LayoutDashboard,
  LogOut,
  Server,
  Settings,
  ShieldCheck,
  Target,
} from "lucide-react";

const navigation = [
  { name: "Overview", href: "/", icon: LayoutDashboard },
  { name: "Agents", href: "/agents", icon: Server },
  { name: "Targets", href: "/targets", icon: Target },
  { name: "Alerts", href: "/alerts", icon: Bell },
];

export default function DashboardShell({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();

  const [loggingOut, setLoggingOut] = useState(false);

  async function handleLogout() {
    if (loggingOut) return;

    setLoggingOut(true);

    try {
      const response = await fetch("/api/auth/logout", {
        method: "POST",
      });

      if (!response.ok) {
        throw new Error("Logout failed");
      }

      router.replace("/login");
      router.refresh();
    } catch (error) {
      console.error("Logout error:", error);
      alert("Unable to logout. Please try again.");
      setLoggingOut(false);
    }
  }

  return (
    <div className="min-h-screen bg-[#f6f8fb] text-slate-900">
      {/* Sidebar */}
      <aside className="fixed inset-y-0 left-0 z-40 hidden w-[250px] flex-col border-r border-white/10 bg-[#111c31] text-slate-300 lg:flex">
        {/* Brand */}
        <div className="flex h-[76px] items-center gap-3 border-b border-white/10 px-6">
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-blue-500 text-white">
            <Activity size={21} strokeWidth={2.5} />
          </div>

          <div>
            <p className="text-lg font-bold tracking-tight text-white">
              PulseCheck
            </p>
            <p className="text-[10px] font-medium uppercase tracking-[0.18em] text-slate-400">
              Monitoring platform
            </p>
          </div>
        </div>

        {/* Navigation */}
        <div className="px-4 pt-8">
          <p className="mb-3 px-3 text-[10px] font-semibold uppercase tracking-[0.16em] text-slate-500">
            Workspace
          </p>

          <nav className="space-y-1">
            {navigation.map((item) => {
              const active =
                item.href === "/"
                  ? pathname === "/"
                  : pathname.startsWith(item.href);

              const Icon = item.icon;

              return (
                <Link
                  key={item.name}
                  href={item.href}
                  className={`flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition ${
                    active
                      ? "bg-blue-500/15 text-blue-400"
                      : "text-slate-400 hover:bg-white/5 hover:text-white"
                  }`}
                >
                  <Icon size={18} />

                  {item.name}

                  {item.name === "Alerts" && (
                    <span className="ml-auto h-2 w-2 rounded-full bg-red-400" />
                  )}
                </Link>
              );
            })}
          </nav>
        </div>

        {/* Sidebar Footer */}
        <div className="mt-auto space-y-1 border-t border-white/10 p-4">
          <Link
            href="/help"
            className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-slate-400 hover:bg-white/5 hover:text-white"
          >
            <CircleHelp size={18} />
            Help & Support
          </Link>

          <Link
            href="/settings"
            className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-slate-400 hover:bg-white/5 hover:text-white"
          >
            <Settings size={18} />
            Settings
          </Link>

          {/* Administrator Profile */}
          <div className="mt-3 flex items-center gap-3 rounded-lg bg-white/5 p-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-blue-500/20 text-blue-300">
              <ShieldCheck size={18} />
            </div>

            <div className="min-w-0 flex-1">
              <p className="truncate text-xs font-semibold text-white">
                Administrator
              </p>
              <p className="text-[10px] text-slate-400">
                System access
              </p>
            </div>
          </div>

          {/* Logout */}
          <button
            type="button"
            onClick={handleLogout}
            disabled={loggingOut}
            className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-slate-400 transition hover:bg-red-500/10 hover:text-red-400 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <LogOut size={18} />
            {loggingOut ? "Signing out..." : "Logout"}
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <div className="min-h-screen lg:pl-[250px]">
        {/* Header */}
        <header className="sticky top-0 z-30 flex h-[76px] items-center justify-between border-b border-slate-200 bg-white/90 px-5 backdrop-blur-md sm:px-8">
          <div>
            <p className="text-sm font-semibold text-slate-800">
              Infrastructure Monitoring
            </p>

            <p className="mt-0.5 text-xs text-slate-400">
              Monitor your infrastructure in one place
            </p>
          </div>

          <div className="flex items-center gap-3">
            <div className="hidden items-center gap-2 rounded-full border border-emerald-100 bg-emerald-50 px-3 py-1.5 sm:flex">
              <span className="h-2 w-2 rounded-full bg-emerald-500" />

              <span className="text-xs font-medium text-emerald-700">
                Monitoring active
              </span>
            </div>

            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-slate-100 text-xs font-bold text-slate-600">
              AD
            </div>
          </div>
        </header>

        {/* Page Content */}
        <main className="mx-auto max-w-[1600px] p-5 sm:p-8">
          {children}
        </main>
      </div>
    </div>
  );
}
