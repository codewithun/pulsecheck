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

export interface AgentDetailResponse {
  agent: Agent;
  summary: {
    total: number;
    up: number;
    down: number;
    unknown: number;
  };
  targets: TargetsResponse;
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


export interface TargetCheck {
  id: number;
  probe_id: string;
  target: string;
  status: "UP" | "DOWN" | "INVALID";
  status_code: number | null;
  latency_ms: number | null;
  error?: string;
  protocol: string;
  checked_at: string;
  received_at: string;
}

export interface TargetHistoryResponse {
  data: TargetCheck[];
  limit: number;
  offset: number;
  has_more: boolean;
}

export interface TargetGroup {
  id: number;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface TargetTag {
  id: number;
  name: string;
  created_at: string;
}

export interface TargetMetadata {
  probe_id: string;
  target: string;
  group: TargetGroup | null;
  tags: TargetTag[];
}

export interface TargetGroupsResponse {
  data: TargetGroup[];
}

export interface TargetTagsResponse {
  data: TargetTag[];
}

export interface CreateTargetGroupRequest {
  name: string;
  description: string;
}

export interface CreateTargetTagRequest {
  name: string;
}

export interface UpdateTargetMetadataRequest {
  probe_id: string;
  target: string;
  group_id: number | null;
  tags: number[];
}