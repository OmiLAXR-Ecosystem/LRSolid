import { apiFetch, apiUrl } from "./client";
import type { ConnectionResponse, StatementDetail } from "../types/statement";

export type StatementQuery = {
    limit: number;
    actorMbox?: string;
    verbId?: string;
    objectId?: string;
    since?: string;
    until?: string;
    cursor?: string | null;
};

export async function fetchStatementList(
    query: StatementQuery,
    signal?: AbortSignal,
): Promise<ConnectionResponse> {
    const params = new URLSearchParams();
    params.set("first", String(query.limit));
    if (query.actorMbox) params.set("actor_mbox", query.actorMbox);
    if (query.verbId) params.set("verb", query.verbId);
    if (query.objectId) params.set("activity", query.objectId);
    if (query.since) params.set("since", query.since);
    if (query.until) params.set("until", query.until);
    if (query.cursor) params.set("after", query.cursor);
    return apiFetch<ConnectionResponse>(`/api/connection/statement?${params.toString()}`, { signal });
}

export async function fetchStatementById(id: string, signal?: AbortSignal): Promise<StatementDetail> {
    return apiFetch<StatementDetail>(`/api/statement/${encodeURIComponent(id)}`, { signal });
}

export async function voidStatement(statementId: string): Promise<void> {
    const voiding = {
        actor: { mbox: "mailto:system@lrsolid.local" },
        verb: { id: "http://adlnet.gov/expapi/verbs/voided" },
        object: { objectType: "StatementRef", id: statementId },
    };
    const res = await fetch(apiUrl("/xapi/statements"), {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
            "X-Experience-API-Version": "1.0.3",
        },
        body: JSON.stringify({ statements: [voiding] }),
    });
    if (!res.ok) throw new Error(`Void failed (${res.status})`);
}

export function downloadStatementJson(detail: StatementDetail): void {
    const blob = new Blob([JSON.stringify(detail.raw, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `statement-${detail.id}.json`;
    a.click();
    URL.revokeObjectURL(url);
}
