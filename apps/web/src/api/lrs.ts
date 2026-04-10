import { apiFetch } from "./client";
import type { CreateLRSPayload, CreateProjectPayload, RestDataStoreListResponse, RestLRS, RestLRSListResponse, RestProject, RestProjectListResponse } from "../types/lrs";

export async function fetchLRSList(): Promise<RestLRSListResponse> {
  return apiFetch<RestLRSListResponse>("/api/v1/lrs");
}

export async function createLRS(payload: CreateLRSPayload): Promise<RestLRS> {
  return apiFetch<RestLRS>("/api/v1/lrs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function deleteLRS(id: string): Promise<void> {
  await apiFetch<{ status: string }>(`/api/v1/lrs/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export async function fetchDataStores(): Promise<RestDataStoreListResponse> {
  return apiFetch<RestDataStoreListResponse>("/api/v1/data-store");
}

export async function fetchProjects(): Promise<RestProjectListResponse> {
  return apiFetch<RestProjectListResponse>("/api/v1/project");
}

export async function createProject(payload: CreateProjectPayload): Promise<RestProject> {
  return apiFetch<RestProject>("/api/v1/project", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function deleteProject(id: string): Promise<void> {
  await apiFetch<{ status: string }>(`/api/v1/project/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}
