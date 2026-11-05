
import DashboardShell from "@/components/layout/dashboard-shell";
import OverviewDashboard from "@/components/overview/overview-dashboard";

export default function Home() {
  return (
    <DashboardShell>
      <OverviewDashboard />
    </DashboardShell>
  );
}
