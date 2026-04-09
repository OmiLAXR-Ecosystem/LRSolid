import { apiFetch } from "./client";
import type { RestUserResponse } from "../types/user";

export type UserQuery = {
    limit: number;
    offset: number;
    q?: string;
    role?: string;
};

export async function fetchUsers(query: UserQuery): Promise<RestUserResponse> {
    const params = new URLSearchParams();
    params.set("limit", String(query.limit));
    params.set("offset", String(query.offset));
    if (query.q) params.set("q", query.q);
    if (query.role && query.role !== "All") params.set("role", query.role);
    return apiFetch<RestUserResponse>(`/api/v1/user?${params.toString()}`);
}
