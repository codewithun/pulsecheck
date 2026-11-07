export type OverviewPeriod = "1h" | "24h" | "7d" | "30d";

export interface Agent {
  probe_id: string;
  hostname: string;
  last_seen: string;
  status: "ONLINE" | "OFFLINE";
}

export interface AgentsResponse {
  data: Agent[];
}

export interface OverviewResponse {
  period: OverviewPeriod;

  agents: {
    total: number;
    online: number;
    offline: number;
  };

  targets: {
    total: number;
    up: number;
    down: number;
    unknown: number;
  };

  checks: {
    total: number;
    up: number;
    down: number;
    invalid: number;
  };

  availability: number | null;

  alerts: {
    pending: number;
    delivered: number;
  };
}

export type TargetStatus = "UP" | "DOWN" | "UNKNOWN";

export type CheckStatus = "UP" | "DOWN" | "INVALID";

export type TargetPeriod = "1h" | "24h" | "7d" | "30d";

export interface MonitoredTarget {
  probe_id: string;
  target: string;
  status: TargetStatus;
  last_check_status: CheckStatus;
  last_checked_at: string | null;
  latency_ms: number | null;
  protocol: string;
  error?: string;
}

export interface TargetsResponse {
  data: MonitoredTarget[];
  limit: number;
  offset: number;
  has_more: boolean;
}
