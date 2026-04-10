import { apiFetch } from "./client";
import type { DashboardResponse } from "../types/dashboard";

export async function fetchDashboard(org: string): Promise<DashboardResponse> {
  return apiFetch<DashboardResponse>(`/api/v1/${encodeURIComponent(org)}/dashboard`);
}
