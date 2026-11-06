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
