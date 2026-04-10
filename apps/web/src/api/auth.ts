import { apiFetch } from "./client";
import type { AuthUser } from "../types/auth";

export type LoginRequest = { email: string; password: string };

export async function login(req: LoginRequest): Promise<AuthUser> {
    return apiFetch<AuthUser>("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
    });
}

export async function logout(): Promise<void> {
    await apiFetch("/api/auth/logout", { method: "POST" });
}

export async function me(): Promise<AuthUser> {
    return apiFetch<AuthUser>("/api/auth/me");
}
