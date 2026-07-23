import { apiFetch } from "./client";
import type { WidgetEntry } from "../plugins/types";
import type { ConfigField } from "../plugins/types";

export type DashboardConfigResponse = {
  widgets: WidgetEntry[];
  schema: Record<string, ConfigField[]>;
};

export function fetchDashboardConfig(org: string): Promise<DashboardConfigResponse> {
  return apiFetch<DashboardConfigResponse>(`/api/v1/${encodeURIComponent(org)}/dashboard/config`);
}

export function saveDashboardConfig(
  org: string,
  widgets: WidgetEntry[],
  scope: "user" | "org",
): Promise<void> {
  return apiFetch(`/api/v1/${encodeURIComponent(org)}/dashboard/config?scope=${scope}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ widgets }),
  });
}

export function resetDashboardConfig(org: string): Promise<void> {
  return apiFetch(`/api/v1/${encodeURIComponent(org)}/dashboard/config?scope=user`, {
    method: "DELETE",
  });
}
