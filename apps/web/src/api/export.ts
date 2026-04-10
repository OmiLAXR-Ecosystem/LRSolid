import { apiFetch, apiUrl } from "./client";
import type {
  CreateStatementExportPayload,
  ExportJob,
  ExportJobListResponse,
  ExportSettings,
  UpdateExportSettingsPayload,
} from "../types/export";

export async function fetchExportSettings(org: string): Promise<ExportSettings> {
  return apiFetch<ExportSettings>(`/api/v1/${encodeURIComponent(org)}/export/settings`);
}

export async function updateExportSettings(org: string, payload: UpdateExportSettingsPayload): Promise<ExportSettings> {
  return apiFetch<ExportSettings>(`/api/v1/${encodeURIComponent(org)}/export/settings`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function createStatementExport(org: string, payload: CreateStatementExportPayload): Promise<ExportJob> {
  return apiFetch<ExportJob>(`/api/v1/${encodeURIComponent(org)}/export/statements`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function fetchExportJobs(org: string, limit = 20): Promise<ExportJobListResponse> {
  return apiFetch<ExportJobListResponse>(`/api/v1/${encodeURIComponent(org)}/export/jobs?limit=${limit}`);
}

export async function downloadExportJob(org: string, jobId: string): Promise<void> {
  const url = apiUrl(`/api/v1/${encodeURIComponent(org)}/export/jobs/${encodeURIComponent(jobId)}/download`);
  const res = await fetch(url, { credentials: "include" });
  if (!res.ok) {
    let message = `Download failed (${res.status})`;
    const ct = res.headers.get("content-type") ?? "";
    if (ct.includes("application/json")) {
      const body = (await res.json().catch(() => null)) as { error?: string } | null;
      if (body?.error) message = body.error;
    }
    throw new Error(message);
  }
  const blob = await res.blob();
  const blobUrl = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = blobUrl;
  a.download = `export-${jobId}.ndjson.gz`;
  a.click();
  URL.revokeObjectURL(blobUrl);
}
