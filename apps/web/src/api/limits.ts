import { apiFetch } from "./client";

export type UsageMetric = "statements_total" | "statements_per_hour" | "statements_per_day" | "storage_bytes";
export type UsageSubject = "organisation" | "project" | "lrs" | "client";

export type UsageLimit = {
  id: string;
  subjectType: UsageSubject;
  subjectId: string;
  metric: UsageMetric;
  maxValue: number;
  enabled: boolean;
  updatedAt: string;
};

export type UsageSnapshot = {
  subjectType: UsageSubject;
  subjectId: string;
  usage: {
    statementsTotal: number;
    statementsThisHour: number;
    statementsToday: number;
    storageBytes: number;
  };
  limits: UsageLimit[];
};

export function fetchUsage(org: string, subjectType: UsageSubject, subjectId: string): Promise<UsageSnapshot> {
  const q = new URLSearchParams({ subjectType, subjectId });
  return apiFetch<UsageSnapshot>(`/api/v1/${encodeURIComponent(org)}/limits?${q}`);
}

export function saveLimit(
  org: string,
  payload: { subjectType: UsageSubject; subjectId: string; metric: UsageMetric; maxValue: number; enabled: boolean },
): Promise<UsageLimit> {
  return apiFetch<UsageLimit>(`/api/v1/${encodeURIComponent(org)}/limits`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export function deleteLimit(org: string, id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/${encodeURIComponent(org)}/limits/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}
