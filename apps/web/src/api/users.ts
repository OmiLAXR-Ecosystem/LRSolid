import { apiFetch } from "./client";
import type { RestUserResponse } from "../types/user";

export type UserQuery = {
    limit: number;
    offset: number;
    q?: string;
    role?: string;
};

export type InviteUserPayload = {
    email: string;
    displayName?: string;
    role: string;
};

export async function inviteUser(org: string, payload: InviteUserPayload): Promise<{ token: string; inviteURL: string }> {
    return apiFetch(`/api/v1/${encodeURIComponent(org)}/user/invite`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
    });
}

export async function fetchUsers(org: string, query: UserQuery): Promise<RestUserResponse> {
    const params = new URLSearchParams();
    params.set("limit", String(query.limit));
    params.set("offset", String(query.offset));
    if (query.q) params.set("q", query.q);
    if (query.role && query.role !== "All") params.set("role", query.role);
    return apiFetch<RestUserResponse>(`/api/v1/${encodeURIComponent(org)}/user?${params.toString()}`);
}
