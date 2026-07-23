import { useMemo, useState } from "react";
import { MoreHorizontal, RefreshCw, SlidersHorizontal } from "lucide-react";
import { Card } from "../ui/Card";
import { Pagination } from "../ui/Pagination";
import type { StatementRow } from "../../types/statement";

type ColumnKey = "timestamp" | "actor" | "verb" | "object" | "id" | "storedAt";

const COLUMN_DEFS: { key: ColumnKey; label: string }[] = [
    { key: "timestamp", label: "Timestamp" },
    { key: "actor", label: "Actor" },
    { key: "verb", label: "Verb" },
    { key: "object", label: "Object" },
    { key: "id", label: "Statement ID" },
    { key: "storedAt", label: "Stored At" },
];

const DEFAULT_COLUMNS: Record<ColumnKey, boolean> = {
    timestamp: true,
    actor: true,
    verb: true,
    object: true,
    id: false,
    storedAt: false,
};

type StatementTableProps = {
    items: StatementRow[];
    loading: boolean;
    error: string | null;
    limit: number;
    onLimitChange: (limit: number) => void;
    onRefresh: () => void;
    onViewDetails: (id: string) => void;
    onExport: (id: string) => void;
    onVoid: (id: string) => void;
    canExport: boolean;
    canVoid: boolean;
    pageIndex: number;
    hasNextPage: boolean;
    onNext: () => void;
    onPrev: () => void;
};

export function StatementTable({
    items,
    loading,
    error,
    limit,
    onLimitChange,
    onRefresh,
    onViewDetails,
    onExport,
    onVoid,
    canExport,
    canVoid,
    pageIndex,
    hasNextPage,
    onNext,
    onPrev,
}: StatementTableProps) {
    const [columns, setColumns] = useState<Record<ColumnKey, boolean>>(DEFAULT_COLUMNS);
    const [showColumns, setShowColumns] = useState(false);
    const [menuOpenId, setMenuOpenId] = useState<string | null>(null);

    const visibleColumns = useMemo(() => COLUMN_DEFS.filter((c) => columns[c.key]), [columns]);

    return (
        <Card>
            <div className="mb-4 flex items-center justify-between">
                <div className="text-sm text-slate-500">
                    {error ? <span className="text-red-600">Error: {error}</span> : `Showing ${items.length} entries`}
                </div>
                <div className="flex items-center gap-3">
                    <button
                        onClick={onRefresh}
                        disabled={loading}
                        className="inline-flex items-center gap-2 rounded-md border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-50 disabled:opacity-50"
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
                                {COLUMN_DEFS.map((col) => (
                                    <label key={col.key} className="flex items-center gap-2 px-2 py-1 text-xs text-slate-700 cursor-pointer">
                                        <input
                                            type="checkbox"
                                            checked={columns[col.key]}
                                            onChange={(e) =>
                                                setColumns((prev) => ({ ...prev, [col.key]: e.target.checked }))
                                            }
                                        />
                                        {col.label}
                                    </label>
                                ))}
                            </div>
                        )}
                    </div>
                </div>
            </div>

            <div className="overflow-x-auto rounded-lg ring-1 ring-slate-200">
                <table className="min-w-full text-sm">
                    <thead className="bg-slate-50 text-slate-600">
                        <tr>
                            {visibleColumns.map((col) => (
                                <th key={col.key} className="px-4 py-3 text-left font-semibold">
                                    {col.label}
                                </th>
                            ))}
                            <th className="px-4 py-3 text-right font-semibold">Actions</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                        {loading && (
                            <tr>
                                <td colSpan={visibleColumns.length + 1} className="px-4 py-6 text-center text-sm text-slate-500">
                                    Loading...
                                </td>
                            </tr>
                        )}
                        {!loading && items.length === 0 && (
                            <tr>
                                <td colSpan={visibleColumns.length + 1} className="px-4 py-6 text-center text-sm text-slate-500">
                                    No statements found.
                                </td>
                            </tr>
                        )}
                        {!loading && items.map((row) => (
                            <tr key={row.id} className="bg-white">
                                {columns.timestamp && (
                                    <td className="px-4 py-3 text-slate-600">{row.timestamp ?? row.storedAt ?? "-"}</td>
                                )}
                                {columns.actor && <td className="px-4 py-3 text-slate-700 max-w-[200px] truncate">{row.actorMbox ?? "-"}</td>}
                                {columns.verb && <td className="px-4 py-3 text-slate-700 max-w-[200px] truncate">{row.verbId ?? "-"}</td>}
                                {columns.object && <td className="px-4 py-3 text-slate-700 max-w-[200px] truncate">{row.objectId ?? "-"}</td>}
                                {columns.id && (
                                    <td className="px-4 py-3 text-slate-500 font-mono text-xs">{row.id}</td>
                                )}
                                {columns.storedAt && <td className="px-4 py-3 text-slate-500">{row.storedAt ?? "-"}</td>}
                                <td className="px-4 py-3 text-right">
                                    <div className="inline-flex items-center gap-2 relative" data-menu-root>
                                        <button
                                            onClick={() => {
                                                setMenuOpenId(null);
                                                onViewDetails(row.id);
                                            }}
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
                                                    onClick={() => { setMenuOpenId(null); onViewDetails(row.id); }}
                                                    className="w-full text-left px-3 py-2 text-xs text-slate-700 hover:bg-slate-50"
                                                >
                                                    View Details
                                                </button>
                                                <button
                                                    onClick={() => { setMenuOpenId(null); onExport(row.id); }}
                                                    disabled={!canExport}
                                                    className="w-full text-left px-3 py-2 text-xs text-slate-700 hover:bg-slate-50 disabled:opacity-50"
                                                >
                                                    Export JSON
                                                </button>
                                                <button
                                                    onClick={() => { setMenuOpenId(null); onVoid(row.id); }}
                                                    disabled={!canVoid}
                                                    className="w-full text-left px-3 py-2 text-xs text-red-600 hover:bg-red-50 disabled:opacity-50"
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

            <Pagination
                label={`Page ${pageIndex + 1}`}
                limit={limit}
                onLimitChange={onLimitChange}
                onPrev={onPrev}
                onNext={onNext}
                prevDisabled={loading || pageIndex === 0}
                nextDisabled={loading || !hasNextPage}
            />
        </Card>
    );
}
