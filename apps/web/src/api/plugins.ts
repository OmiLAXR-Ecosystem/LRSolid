import { apiFetch } from "./client";

export type InstalledPlugin = {
  id: string;
  slug: string;
  name: string;
  description: string;
  s3Key: string;
  enabled: boolean;
  installedAt: string;
};

export function fetchPlugins(org: string): Promise<{ items: InstalledPlugin[] }> {
  return apiFetch(`/api/v1/${encodeURIComponent(org)}/plugins`);
}

export function installPlugin(org: string, file: File, name: string, description: string): Promise<InstalledPlugin> {
  const form = new FormData();
  form.append("file", file);
  form.append("name", name);
  form.append("description", description);
  return apiFetch(`/api/v1/${encodeURIComponent(org)}/plugins`, { method: "POST", body: form });
}

export function togglePlugin(org: string, id: string, enabled: boolean): Promise<void> {
  return apiFetch(`/api/v1/${encodeURIComponent(org)}/plugins/${id}/toggle`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled }),
  });
}

export function deletePlugin(org: string, id: string): Promise<void> {
  return apiFetch(`/api/v1/${encodeURIComponent(org)}/plugins/${id}`, { method: "DELETE" });
}
