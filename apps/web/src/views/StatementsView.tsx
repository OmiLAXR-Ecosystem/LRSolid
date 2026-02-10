import { Calendar, ChevronDown, MoreHorizontal, RefreshCw, SlidersHorizontal, Download, Ban } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Card } from "../components/Card";

type StatementRow = {
    id: string;
    actorMbox?: string | null;
    verbId?: string | null;
    objectId?: string | null;
    timestamp?: string | null;
    storedAt?: string | null;
};

type ConnectionEdge = {
    cursor: string;
    node: StatementRow;
};

type ConnectionResponse = {
    edges: ConnectionEdge[];
    pageInfo: { hasNextPage: boolean; endCursor?: string };
};

type StatementDetail = {
    id: string;
    storedAt: string;
    timestamp?: string | null;
    actorMbox?: string | null;
    verbId?: string | null;
    objectId?: string | null;
    raw: any;
};

const apiBase = import.meta.env.VITE_API_BASE ?? "";

function joinApi(path: string) {
    if (!apiBase) return path;
    const base = apiBase.endsWith("/") ? apiBase.slice(0, -1) : apiBase;
    return base + (path.startsWith("/") ? path : "/" + path);
}

function FieldLabel({ children }: { children: React.ReactNode }) {
    return <div className="text-xs font-semibold text-slate-600">{children}</div>;
}

function SelectBox({
    value,
    onChange,
    placeholder,
}: {
    value: string;
    onChange: (value: string) => void;
    placeholder: string;
}) {
    return (
        <div className="relative">
            <input
                value={value}
                onChange={(e) => onChange(e.target.value)}
                placeholder={placeholder}
                className="w-full rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300"
            />
            <ChevronDown className="h-4 w-4 text-slate-400 absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none" />
        </div>
    );
}

function DateBox({
    value,
    onChange,
    placeholder,
}: {
    value: string;
    onChange: (value: string) => void;
    placeholder: string;
}) {
    return (
        <div className="relative">
            <input
                type="text"
                value={value}
                onChange={(e) => onChange(e.target.value)}
                placeholder={placeholder}
                className="w-full rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300"
            />
            <Calendar className="h-4 w-4 text-slate-400 absolute right-3 top-1/2 -translate-y-1/2" />
        </div>
    );
}

function JSONTree({ data, depth = 0 }: { data: any; depth?: number }) {
    if (data === null || data === undefined) {
        return <div className="text-slate-500">null</div>;
    }
    if (typeof data !== "object") {
        return <div className="text-slate-700">{String(data)}</div>;
    }

    const entries = Array.isArray(data) ? data.map((v, i) => [String(i), v]) : Object.entries(data);

    return (
        <div className="space-y-1">
            {entries.map(([key, value]) => (
                <details key={`${depth}-${key}`} className="rounded-md border border-slate-200 bg-white/70 px-3 py-2">
                    <summary className="cursor-pointer text-sm font-semibold text-slate-700">
                        {Array.isArray(data) ? `[#${key}]` : key}
                    </summary>
                    <div className="mt-2 pl-3 border-l border-slate-200">
                        <JSONTree data={value} depth={depth + 1} />
                    </div>
                </details>
            ))}
        </div>
    );
}

export function StatementsView() {
    const [statementId, setStatementId] = useState("");
    const [actorMbox, setActorMbox] = useState("");
    const [verbId, setVerbId] = useState("");
    const [objectId, setObjectId] = useState("");
    const [dateFrom, setDateFrom] = useState("");
    const [dateTo, setDateTo] = useState("");
    const [limit, setLimit] = useState(25);
    const [items, setItems] = useState<StatementRow[]>([]);
    const [nextCursor, setNextCursor] = useState<string | null>(null);
    const [cursorHistory, setCursorHistory] = useState<string[]>([]);
    const [pageIndex, setPageIndex] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const [showColumns, setShowColumns] = useState(false);
    const [selected, setSelected] = useState<StatementDetail | null>(null);
    const [detailOpen, setDetailOpen] = useState(false);
    const [detailLoading, setDetailLoading] = useState(false);
    const [detailError, setDetailError] = useState<string | null>(null);
    const [menuOpenId, setMenuOpenId] = useState<string | null>(null);

    const [columns, setColumns] = useState<Record<string, boolean>>({
        timestamp: true,
        actor: true,
        verb: true,
        object: true,
        id: false,
        storedAt: false,
    });

    const columnDefs = useMemo(
        () => [
            { key: "timestamp", label: "Timestamp" },
            { key: "actor", label: "Actor" },
            { key: "verb", label: "Verb" },
            { key: "object", label: "Object" },
            { key: "id", label: "Statement ID" },
            { key: "storedAt", label: "Stored At" },
        ],
        []
    );

    const visibleColumns = columnDefs.filter((c) => columns[c.key]);

    const fetchStatements = async (cursor?: string | null, resetHistory?: boolean) => {
        setLoading(true);
        setError(null);
        const controller = new AbortController();

        try {
            if (statementId.trim()) {
                const res = await fetch(joinApi(`/api/statement/${encodeURIComponent(statementId.trim())}`), {
                    signal: controller.signal,
                });
                if (!res.ok) {
                    throw new Error(`Request failed (${res.status})`);
                }
                const detail = (await res.json()) as StatementDetail;
                setItems([
                    {
                        id: detail.id,
                        actorMbox: detail.actorMbox,
                        verbId: detail.verbId,
                        objectId: detail.objectId,
                        timestamp: detail.timestamp ?? null,
                        storedAt: detail.storedAt,
                    },
                ]);
                setNextCursor(null);
                if (resetHistory) {
                    setCursorHistory([]);
                    setPageIndex(0);
                }
                return;
            }

            const params = new URLSearchParams();
            params.set("first", String(limit));
            if (actorMbox.trim()) params.set("actor_mbox", actorMbox.trim());
            if (verbId.trim()) params.set("verb", verbId.trim());
            if (objectId.trim()) params.set("activity", objectId.trim());
            if (dateFrom.trim()) params.set("since", dateFrom.trim());
            if (dateTo.trim()) params.set("until", dateTo.trim());
            if (cursor) params.set("after", cursor);

            const res = await fetch(`${joinApi("/api/connection/statement")}?${params.toString()}`, {
                signal: controller.signal,
            });
            if (!res.ok) {
                throw new Error(`Request failed (${res.status})`);
            }
            const data = (await res.json()) as ConnectionResponse;
            setItems((data.edges ?? []).map((edge) => edge.node));
            setNextCursor(data.pageInfo?.endCursor ?? null);
            if (resetHistory) {
                setCursorHistory([]);
                setPageIndex(0);
            }
        } catch (err) {
            const message = err instanceof Error ? err.message : "Unknown error";
            setError(message);
        } finally {
            setLoading(false);
        }
        return () => controller.abort();
    };

    useEffect(() => {
        void fetchStatements(null, true);
    }, []);

    const applyFilters = () => {
        void fetchStatements(null, true);
    };

    const goNext = () => {
        if (!nextCursor) return;
        setCursorHistory((prev) => {
            const next = prev.slice(0, pageIndex + 1);
            next.push(nextCursor);
            return next;
        });
        setPageIndex((p) => p + 1);
        void fetchStatements(nextCursor);
    };

    const goPrev = () => {
        if (pageIndex === 0) return;
        const prevCursor = pageIndex - 2 >= 0 ? cursorHistory[pageIndex - 2] : null;
        setPageIndex((p) => Math.max(0, p - 1));
        void fetchStatements(prevCursor);
    };

    const openDetails = async (id: string) => {
        setDetailOpen(true);
        setDetailLoading(true);
        setDetailError(null);
        try {
            const res = await fetch(joinApi(`/api/statement/${encodeURIComponent(id)}`));
            if (!res.ok) throw new Error(`Request failed (${res.status})`);
            const data = (await res.json()) as StatementDetail;
            setSelected(data);
            return data;
        } catch (err) {
            const message = err instanceof Error ? err.message : "Unknown error";
            setDetailError(message);
            return null;
        } finally {
            setDetailLoading(false);
        }
    };

    const closeDetails = () => {
        setDetailOpen(false);
        setSelected(null);
        setDetailError(null);
        setDetailLoading(false);
    };

    const exportStatement = (detail: StatementDetail) => {
        const blob = new Blob([JSON.stringify(detail.raw, null, 2)], { type: "application/json" });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `statement-${detail.id}.json`;
        a.click();
        URL.revokeObjectURL(url);
    };

    const voidStatement = async (detail: StatementDetail) => {
        const ok = window.confirm(`Void statement ${detail.id}? This cannot be undone.`);
        if (!ok) return;
        const voiding = {
            actor: { mbox: "mailto:system@lrsolid.local" },
            verb: { id: "http://adlnet.gov/expapi/verbs/voided" },
            object: { objectType: "StatementRef", id: detail.id },
        };
        await fetch(joinApi("/xapi/statements"), {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "X-Experience-API-Version": "1.0.3",
            },
            body: JSON.stringify({ statements: [voiding] }),
        });
        void fetchStatements(null, true);
    };

    useEffect(() => {
        const onKey = (e: KeyboardEvent) => {
            if (e.key === "Escape") {
                setMenuOpenId(null);
                if (detailOpen) closeDetails();
            }
        };
        const onClick = (e: MouseEvent) => {
            const target = e.target as HTMLElement | null;
            if (!target) return;
            if (!target.closest("[data-menu-root]")) {
                setMenuOpenId(null);
            }
        };
        window.addEventListener("keydown", onKey);
        window.addEventListener("click", onClick);
        return () => {
            window.removeEventListener("keydown", onKey);
            window.removeEventListener("click", onClick);
        };
    }, [detailOpen]);

    const canExport = import.meta.env.VITE_CAN_EXPORT !== "false";
    const canVoid = import.meta.env.VITE_CAN_VOID !== "false";

    return (
        <div className="space-y-6">
            <Card>
                <div className="grid grid-cols-[1.1fr_1.1fr_1.1fr_1fr_1fr_1fr_auto] gap-4 items-end">
                    <div>
                        <FieldLabel>Statement ID:</FieldLabel>
                        <SelectBox value={statementId} onChange={setStatementId} placeholder="Exact ID" />
                    </div>
                    <div>
                        <FieldLabel>Actor:</FieldLabel>
                        <SelectBox value={actorMbox} onChange={setActorMbox} placeholder="All" />
                    </div>
                    <div>
                        <FieldLabel>Verb:</FieldLabel>
                        <SelectBox value={verbId} onChange={setVerbId} placeholder="All" />
                    </div>
                    <div>
                        <FieldLabel>Object:</FieldLabel>
                        <SelectBox value={objectId} onChange={setObjectId} placeholder="All" />
                    </div>
                    <div>
                        <FieldLabel>Date from:</FieldLabel>
                        <DateBox value={dateFrom} onChange={setDateFrom} placeholder="2022-01-01" />
                    </div>
                    <div>
                        <FieldLabel>Date to:</FieldLabel>
                        <DateBox value={dateTo} onChange={setDateTo} placeholder="2022-01-30" />
                    </div>
                    <button
                        onClick={applyFilters}
                        className="h-10 rounded-lg bg-teal-600 px-4 text-sm font-semibold text-white hover:bg-teal-700 transition disabled:opacity-60"
                        disabled={loading}
                    >
                        {loading ? "Loading..." : "Apply Filters"}
                    </button>
                </div>
            </Card>

            <Card>
                <div className="mb-4 flex items-center justify-between">
                    <div className="text-sm text-slate-500">
                        {error ? `Error: ${error}` : `Showing ${items.length} entries`}
                    </div>
                    <div className="flex items-center gap-3">
                        <label className="text-xs font-semibold text-slate-500">Limit</label>
                        <select
                            value={limit}
                            onChange={(e) => setLimit(Number(e.target.value))}
                            className="rounded-md border border-slate-200 bg-white px-2 py-1 text-sm"
                        >
                            <option value={10}>10</option>
                            <option value={25}>25</option>
                            <option value={50}>50</option>
                            <option value={100}>100</option>
                        </select>
                        <button
                            onClick={() => void fetchStatements(null, true)}
                            className="inline-flex items-center gap-2 rounded-md border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-50"
                            disabled={loading}
                        >
                            <RefreshCw className="h-3.5 w-3.5" />
                            Refresh
                        </button>
                        <div className="relative">
                            <button
                                onClick={() => setShowColumns((s) => !s)}
                                className="inline-flex items-center gap-2 rounded-md border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-50"
                            >
                                <SlidersHorizontal className="h-3.5 w-3.5" />
                                Columns
                            </button>
                            {showColumns && (
                                <div className="absolute right-0 mt-2 w-52 rounded-lg border border-slate-200 bg-white shadow-md p-2 z-10">
                                    {columnDefs.map((col) => (
                                        <label key={col.key} className="flex items-center gap-2 px-2 py-1 text-xs text-slate-700">
                                            <input
                                                type="checkbox"
                                                checked={columns[col.key]}
                                                onChange={(e) => setColumns((prev) => ({ ...prev, [col.key]: e.target.checked }))}
                                            />
                                            {col.label}
                                        </label>
                                    ))}
                                </div>
                            )}
                        </div>
                    </div>
                </div>
                <div className="overflow-hidden rounded-lg ring-1 ring-slate-200">
                    <table className="min-w-full text-sm">
                        <thead className="bg-slate-50 text-slate-600">
                            <tr>
                                {visibleColumns.map((col) => (
                                    <th key={col.key} className="px-4 py-3 text-left font-semibold">{col.label}</th>
                                ))}
                                <th className="px-4 py-3 text-right font-semibold">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {items.map((row) => (
                                <tr key={row.id} className="bg-white">
                                    {columns.timestamp && (
                                        <td className="px-4 py-3 text-slate-600">{row.timestamp ?? row.storedAt ?? "-"}</td>
                                    )}
                                    {columns.actor && <td className="px-4 py-3 text-slate-700">{row.actorMbox ?? "-"}</td>}
                                    {columns.verb && <td className="px-4 py-3 text-slate-700">{row.verbId ?? "-"}</td>}
                                    {columns.object && <td className="px-4 py-3 text-slate-700">{row.objectId ?? "-"}</td>}
                                    {columns.id && <td className="px-4 py-3 text-slate-500 font-mono text-xs">{row.id}</td>}
                                    {columns.storedAt && <td className="px-4 py-3 text-slate-500">{row.storedAt ?? "-"}</td>}
                                    <td className="px-4 py-3 text-right">
                                        <div className="inline-flex items-center gap-2 relative" data-menu-root>
                                            <button
                                                onClick={() => void openDetails(row.id)}
                                                className="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-teal-600 hover:bg-teal-50"
                                            >
                                                View Details
                                            </button>
                                            <button
                                                onClick={() => setMenuOpenId((prev) => (prev === row.id ? null : row.id))}
                                                className="rounded-lg border border-slate-200 p-2 text-slate-500 hover:bg-slate-50"
                                            >
                                                <MoreHorizontal className="h-4 w-4" />
                                            </button>
                                            {menuOpenId === row.id && (
                                                <div className="absolute right-0 top-9 w-44 rounded-lg border border-slate-200 bg-white shadow-md z-10">
                                                    <button
                                                        onClick={() => {
                                                            setMenuOpenId(null);
                                                            void openDetails(row.id);
                                                        }}
                                                        className="w-full text-left px-3 py-2 text-xs text-slate-700 hover:bg-slate-50"
                                                    >
                                                        View Details
                                                    </button>
                                                    <button
                                                        onClick={() => {
                                                            setMenuOpenId(null);
                                                            void (async () => {
                                                                const detail = selected?.id === row.id ? selected : await openDetails(row.id);
                                                                if (detail && canExport) exportStatement(detail);
                                                            })();
                                                        }}
                                                        className="w-full text-left px-3 py-2 text-xs text-slate-700 hover:bg-slate-50 disabled:opacity-50"
                                                        disabled={!canExport}
                                                    >
                                                        Export JSON
                                                    </button>
                                                    <button
                                                        onClick={() => {
                                                            setMenuOpenId(null);
                                                            void (async () => {
                                                                const detail = selected?.id === row.id ? selected : await openDetails(row.id);
                                                                if (detail && canVoid) await voidStatement(detail);
                                                            })();
                                                        }}
                                                        className="w-full text-left px-3 py-2 text-xs text-red-600 hover:bg-red-50 disabled:opacity-50"
                                                        disabled={!canVoid}
                                                    >
                                                        Void Statement
                                                    </button>
                                                </div>
                                            )}
                                        </div>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>

                <div className="mt-4 flex items-center justify-between text-sm text-slate-600">
                    <div>Page {pageIndex + 1}</div>
                    <div className="flex items-center gap-2">
                        <button
                            onClick={goPrev}
                            className="rounded-md border border-slate-200 px-2.5 py-1 text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                            disabled={loading || pageIndex === 0}
                        >
                            Previous
                        </button>
                        <button
                            onClick={goNext}
                            className="rounded-md border border-slate-200 px-2.5 py-1 text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                            disabled={loading || !nextCursor || statementId.trim().length > 0}
                        >
                            Next
                        </button>
                    </div>
                </div>
            </Card>

            {detailOpen && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
                    <div className="w-[90vw] max-w-5xl rounded-2xl bg-white shadow-lg ring-1 ring-slate-200">
                        <div className="flex items-center justify-between border-b border-slate-200 px-6 py-4">
                            <div className="text-lg font-semibold text-slate-800">Statement Details</div>
                            <button className="text-slate-500 hover:text-slate-700" onClick={closeDetails}>
                                ✕
                            </button>
                        </div>
                        <div className="px-6 py-4 grid grid-cols-[1fr_1fr] gap-6">
                            <div>
                                <div className="text-sm font-semibold text-slate-700 mb-2">Raw</div>
                                <pre className="rounded-lg bg-slate-50 p-3 text-xs text-slate-700 overflow-auto max-h-[60vh]">
                                    {selected ? JSON.stringify(selected.raw, null, 2) : "Loading..."}
                                </pre>
                            </div>
                            <div>
                                <div className="text-sm font-semibold text-slate-700 mb-2">Tree</div>
                                <div className="max-h-[60vh] overflow-auto">
                                    {selected ? <JSONTree data={selected.raw} /> : <div className="text-slate-500">Loading...</div>}
                                </div>
                            </div>
                        </div>
                        <div className="flex items-center justify-between border-t border-slate-200 px-6 py-4">
                            <div className="text-xs text-slate-500">{selected?.id ?? ""}</div>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={() => selected && canExport && exportStatement(selected)}
                                    className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                                    disabled={!selected || !canExport}
                                >
                                    <Download className="h-3.5 w-3.5" /> Export
                                </button>
                                <button
                                    onClick={() => selected && canVoid && void voidStatement(selected)}
                                    className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-red-600 hover:bg-red-50 disabled:opacity-50"
                                    disabled={!selected || !canVoid}
                                >
                                    <Ban className="h-3.5 w-3.5" /> Void
                                </button>
                            </div>
                        </div>
                        {detailLoading && <div className="px-6 pb-4 text-sm text-slate-500">Loading...</div>}
                        {detailError && <div className="px-6 pb-4 text-sm text-red-600">{detailError}</div>}
                    </div>
                </div>
            )}
        </div>
    );
}
