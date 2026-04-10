import { apiFetch } from "./client";
import type { CreateClientPayload, CreateClientResponse, RestClientResponse, RestLRSResponse } from "../types/client";

export type ClientQuery = {
  limit: number;
  offset: number;
  q?: string;
};

export async function fetchClients(org: string, query: ClientQuery): Promise<RestClientResponse> {
  const params = new URLSearchParams();
  params.set("limit", String(query.limit));
  params.set("offset", String(query.offset));
  if (query.q) params.set("q", query.q);
  return apiFetch<RestClientResponse>(`/api/v1/${encodeURIComponent(org)}/client?${params.toString()}`);
}

export async function createClient(org: string, payload: CreateClientPayload): Promise<CreateClientResponse> {
  return apiFetch<CreateClientResponse>(`/api/v1/${encodeURIComponent(org)}/client`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function deleteClient(org: string, id: string): Promise<void> {
  await apiFetch<{ status: string }>(`/api/v1/${encodeURIComponent(org)}/client/${id}`, {
    method: "DELETE",
  });
}

export async function fetchLRS(org: string): Promise<RestLRSResponse> {
  return apiFetch<RestLRSResponse>(`/api/v1/${encodeURIComponent(org)}/lrs`);
}
