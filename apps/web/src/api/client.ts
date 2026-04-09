const apiBase = import.meta.env.VITE_API_BASE ?? "";

export function apiUrl(path: string): string {
    if (!apiBase) return path;
    const base = apiBase.endsWith("/") ? apiBase.slice(0, -1) : apiBase;
    return base + (path.startsWith("/") ? path : "/" + path);
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await fetch(apiUrl(path), init);
    if (!res.ok) throw new Error(`Request failed (${res.status})`);
    return res.json() as Promise<T>;
}
