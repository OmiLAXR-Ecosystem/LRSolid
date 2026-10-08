import { apiFetch } from "./client";
import type { RestDataStore, RestDataStoreListResponse } from "../types/lrs";

export type DataStoreWrite = {
  name: string;
  databaseType: "postgres" | "mongodb";
  connectionString: string;
  description?: string;
};

export async function listDataStores(org: string, includeInactive = false): Promise<RestDataStoreListResponse> {
  const q = includeInactive ? "?includeInactive=true" : "";
  return apiFetch<RestDataStoreListResponse>(`/api/v1/${encodeURIComponent(org)}/data-store${q}`);
}

export async function createDataStore(payload: DataStoreWrite): Promise<RestDataStore> {
  return apiFetch<RestDataStore>("/api/v1/admin/data-store", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function updateDataStore(id: string, payload: { name?: string; connectionString?: string; description?: string; isActive?: boolean }): Promise<RestDataStore> {
  return apiFetch<RestDataStore>(`/api/v1/admin/data-store/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function deleteDataStore(id: string): Promise<void> {
  await apiFetch<{ status: string }>(`/api/v1/admin/data-store/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export async function testDataStore(payload: { databaseType: string; connectionString: string; name?: string }): Promise<{ ok: boolean }> {
  return apiFetch<{ ok: boolean }>("/api/v1/admin/data-store/test", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function testDataStoreByID(id: string): Promise<{ ok: boolean }> {
  return apiFetch<{ ok: boolean }>(`/api/v1/admin/data-store/${encodeURIComponent(id)}/test`, { method: "POST" });
}
