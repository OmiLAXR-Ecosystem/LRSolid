import { apiFetch } from "./client";

export type AdminCreateOrgPayload = {
    name: string;
    slug?: string;
};

export type AdminCreateUserPayload = {
    email: string;
    displayName?: string;
    password: string;
    role: string;
    orgSlug: string;
};

export type AdminAssignUserToOrgPayload = {
    email: string;
    orgSlug: string;
    role: string;
};

export async function adminCreateOrg(payload: AdminCreateOrgPayload): Promise<{ id: string; slug: string; name: string }> {
    return apiFetch("/api/v1/admin/org", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
    });
}

export async function adminCreateUser(payload: AdminCreateUserPayload): Promise<{ id: string; email: string; role: string }> {
    return apiFetch("/api/v1/admin/user", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
    });
}

export async function adminAssignUserToOrg(payload: AdminAssignUserToOrgPayload): Promise<{ userId: string; email: string; role: string; orgSlug: string }> {
    return apiFetch("/api/v1/admin/user/assign", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
    });
}
