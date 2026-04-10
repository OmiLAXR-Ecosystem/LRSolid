import { apiFetch } from "./client";
import type { CreateLRSPayload, CreateProjectPayload, RestDataStoreListResponse, RestLRS, RestLRSListResponse, RestProject, RestProjectListResponse } from "../types/lrs";

export async function fetchLRSList(org: string): Promise<RestLRSListResponse> {
  return apiFetch<RestLRSListResponse>(`/api/v1/${encodeURIComponent(org)}/lrs`);
}

export async function createLRS(org: string, payload: CreateLRSPayload): Promise<RestLRS> {
  return apiFetch<RestLRS>(`/api/v1/${encodeURIComponent(org)}/lrs`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function deleteLRS(org: string, id: string): Promise<void> {
  await apiFetch<{ status: string }>(`/api/v1/${encodeURIComponent(org)}/lrs/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export async function fetchDataStores(org: string): Promise<RestDataStoreListResponse> {
  return apiFetch<RestDataStoreListResponse>(`/api/v1/${encodeURIComponent(org)}/data-store`);
}

export async function fetchProjects(org: string): Promise<RestProjectListResponse> {
  return apiFetch<RestProjectListResponse>(`/api/v1/${encodeURIComponent(org)}/project`);
}

export async function createProject(org: string, payload: CreateProjectPayload): Promise<RestProject> {
  return apiFetch<RestProject>(`/api/v1/${encodeURIComponent(org)}/project`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function deleteProject(org: string, id: string): Promise<void> {
  await apiFetch<{ status: string }>(`/api/v1/${encodeURIComponent(org)}/project/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}
