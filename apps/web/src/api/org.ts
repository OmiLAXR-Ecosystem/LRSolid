import { apiFetch } from "./client";
import type { RestOrgListResponse } from "../types/org";

export async function fetchOrgs(): Promise<RestOrgListResponse> {
  return apiFetch<RestOrgListResponse>("/api/v1/org");
}
