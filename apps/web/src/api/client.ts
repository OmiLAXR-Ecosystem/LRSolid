const apiBase = import.meta.env.VITE_API_BASE ?? "";

export class ApiError extends Error {
    status: number;

    constructor(status: number, message: string) {
        super(message);
        this.name = "ApiError";
        this.status = status;
    }
}

export function apiUrl(path: string): string {
    if (!apiBase) return path;
    const base = apiBase.endsWith("/") ? apiBase.slice(0, -1) : apiBase;
    return base + (path.startsWith("/") ? path : "/" + path);
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await fetch(apiUrl(path), {
        credentials: "include",
        ...init,
    });

    if (!res.ok) {
        let message = `Request failed (${res.status})`;
        const ct = res.headers.get("content-type") ?? "";

        if (ct.includes("application/json")) {
            const body = (await res.json().catch(() => null)) as { error?: string } | null;
            if (body?.error) message = body.error;
        } else {
            const text = await res.text().catch(() => "");
            if (text) message = text;
        }

        throw new ApiError(res.status, message);
    }

    return res.json() as Promise<T>;
}
