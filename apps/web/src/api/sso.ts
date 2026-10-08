import { apiFetch } from "./client";

export type LoginProvider = {
    slug: string;
    type: "oidc" | "saml" | string;
    displayName: string;
};

export type LoginProvidersResponse = {
    localLoginEnabled: boolean;
    providers: LoginProvider[];
};

export type AuthProviderConfig = Record<string, unknown>;

export type AuthProviderRecord = {
    id: string;
    slug: string;
    type: "oidc" | "saml";
    displayName: string;
    enabled: boolean;
    autoProvision: boolean;
    defaultRole: string;
    linkByEmail: boolean;
    hasSecret: boolean;
    config: AuthProviderConfig;
};

export type AuthProviderWrite = {
    slug: string;
    type: "oidc" | "saml";
    displayName: string;
    enabled: boolean;
    autoProvision: boolean;
    defaultRole: string;
    linkByEmail: boolean;
    config: AuthProviderConfig;
    clientSecret?: string;
    spPrivateKeyPem?: string;
    generateSpKey?: boolean;
};

export type InviteInfo = {
    email: string;
    orgSlug: string;
    orgName: string;
    role: string;
    expiresAt: string;
};

export async function listLoginProviders(org?: string): Promise<LoginProvidersResponse> {
    const q = org ? `?org=${encodeURIComponent(org)}` : "";
    return apiFetch<LoginProvidersResponse>(`/api/auth/providers${q}`);
}

export function ssoStartPath(slug: string, returnTo: string, org?: string, invite?: string): string {
    const params = new URLSearchParams();
    if (returnTo) params.set("return_to", returnTo);
    if (org) params.set("org", org);
    if (invite) params.set("invite", invite);
    return `/api/auth/sso/${encodeURIComponent(slug)}/start?${params.toString()}`;
}

export async function getInvite(token: string): Promise<InviteInfo> {
    return apiFetch<InviteInfo>(`/api/auth/invite?token=${encodeURIComponent(token)}`);
}

export async function acceptInvite(token: string, password: string, displayName?: string): Promise<{ id: string; email: string; role: string }> {
    return apiFetch("/api/auth/invite/accept", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token, password, displayName }),
    });
}

export async function getAuthSettings(org: string): Promise<{ localLoginEnabled: boolean }> {
    return apiFetch(`/api/v1/${encodeURIComponent(org)}/auth-settings`);
}

export async function updateAuthSettings(org: string, localLoginEnabled: boolean): Promise<{ localLoginEnabled: boolean }> {
    return apiFetch(`/api/v1/${encodeURIComponent(org)}/auth-settings`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ localLoginEnabled }),
    });
}

export async function listAuthProviders(org: string): Promise<{ items: AuthProviderRecord[] }> {
    return apiFetch(`/api/v1/${encodeURIComponent(org)}/auth-provider`);
}

export async function createAuthProvider(org: string, body: AuthProviderWrite): Promise<{ id: string }> {
    return apiFetch(`/api/v1/${encodeURIComponent(org)}/auth-provider`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
    });
}

export async function updateAuthProvider(org: string, id: string, body: AuthProviderWrite): Promise<{ id: string }> {
    return apiFetch(`/api/v1/${encodeURIComponent(org)}/auth-provider/${encodeURIComponent(id)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
    });
}

export async function deleteAuthProvider(org: string, id: string): Promise<void> {
    await apiFetch(`/api/v1/${encodeURIComponent(org)}/auth-provider/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export async function testAuthProvider(org: string, id: string): Promise<{ ok: boolean; detail: string }> {
    return apiFetch(`/api/v1/${encodeURIComponent(org)}/auth-provider/${encodeURIComponent(id)}/test`, { method: "POST" });
}

export function samlMetadataPath(slug: string): string {
    return `/api/auth/saml/${encodeURIComponent(slug)}/metadata`;
}
