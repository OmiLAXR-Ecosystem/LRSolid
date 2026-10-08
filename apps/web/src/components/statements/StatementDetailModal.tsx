import { Ban, ChevronDown, Download } from "lucide-react";
import type { ReactNode } from "react";
import { useState } from "react";
import { Modal } from "../ui/Modal";
import { JSONTree } from "./JSONTree";
import { getStatementKeyColorClass, getStatementRawHighlightClass, getStatementTabColorClass } from "./keyColor";
import type { StatementDetail } from "../../types/statement";

type StatementDetailModalProps = {
    open: boolean;
    detail: StatementDetail | null;
    loading: boolean;
    error: string | null;
    canExport: boolean;
    canVoid: boolean;
    onClose: () => void;
    onExport: () => void;
    onVoid: () => void;
};

const JSON_TOKEN_REGEX = /("(?:\\u[\dA-Fa-f]{4}|\\[^u]|[^\\"])*"(?=\s*:)?|"(?:\\u[\dA-Fa-f]{4}|\\[^u]|[^\\"])*"|\btrue\b|\bfalse\b|\bnull\b|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)/g;

type Range = {
    start: number;
    end: number;
};

function getValueAtPath(root: unknown, path: string[]): unknown {
    let current = root;
    for (const segment of path) {
        if (current === null || current === undefined || typeof current !== "object") return undefined;
        if (Array.isArray(current)) {
            const idx = Number(segment);
            if (!Number.isInteger(idx) || idx < 0 || idx >= current.length) return undefined;
            current = current[idx];
        } else {
            current = (current as Record<string, unknown>)[segment];
        }
    }
    return current;
}

function getHighlightRange(root: unknown, path: string[]): Range | null {
    if (path.length === 0) return null;
    const json = JSON.stringify(root, null, 2);
    const value = getValueAtPath(root, path);
    const key = path[path.length - 1];
    const indent = "  ".repeat(path.length);
    const valueStr = JSON.stringify(value, null, 2).replace(/\n/g, `\n${indent}`);
    const snippet = `${indent}${JSON.stringify(key)}: ${valueStr}`;
    const start = json.indexOf(snippet);
    if (start < 0) return null;
    return { start, end: start + snippet.length };
}

function renderHighlightedJSON(value: unknown, hoveredPath: string[] | null): ReactNode[] {
    const json = JSON.stringify(value, null, 2);
    const hoveredKey = hoveredPath && hoveredPath.length > 0 ? hoveredPath[hoveredPath.length - 1] : null;
    const range = hoveredPath ? getHighlightRange(value, hoveredPath) : null;
    const nodes: ReactNode[] = [];
    let lastIndex = 0;

    const pushSegment = (segment: string, start: number, baseClassName: string) => {
        if (!segment) return;
        const end = start + segment.length;
        const isInHoveredRange = !!range && start < range.end && end > range.start;
        if (isInHoveredRange && hoveredKey) {
            const highlightStart = Math.max(start, range.start);
            const highlightEnd = Math.min(end, range.end);
            const before = segment.slice(0, highlightStart - start);
            const middle = segment.slice(highlightStart - start, highlightEnd - start);
            const after = segment.slice(highlightEnd - start);

            if (before) {
                pushSegment(before, start, baseClassName);
            }
            nodes.push(
                <span
                    key={`${highlightStart}-h`}
                    className={`${baseClassName} rounded-sm px-0.5 font-bold ${getStatementRawHighlightClass(hoveredKey)}`}
                >
                    {middle}
                </span>,
            );
            if (after) {
                pushSegment(after, highlightEnd, baseClassName);
            }
            return;
        }
        if (baseClassName === "text-slate-700") {
            nodes.push(segment);
            return;
        }
        nodes.push(
            <span key={start} className={baseClassName}>
                {segment}
            </span>,
        );
    };

    for (const match of json.matchAll(JSON_TOKEN_REGEX)) {
        const token = match[0];
        const index = match.index ?? 0;
        if (index > lastIndex) {
            pushSegment(json.slice(lastIndex, index), lastIndex, "text-slate-700");
        }

        let className = "text-slate-700";
        const isKey = token.startsWith("\"") && /^\s*:/.test(json.slice(index + token.length));
        if (isKey) {
            const rawKey = token.slice(0, -1);
            let key = rawKey;
            try {
                key = JSON.parse(rawKey) as string;
            } catch {
                key = rawKey.slice(1, -1);
            }
            className = getStatementKeyColorClass(key);
        } else if (token.startsWith("\"")) {
            className = "text-emerald-700";
        } else if (token === "true" || token === "false") {
            className = "text-violet-700";
        } else if (token === "null") {
            className = "text-rose-700";
        } else {
            className = "text-amber-700";
        }

        pushSegment(token, index, className);
        lastIndex = index + token.length;
    }

    if (lastIndex < json.length) {
        pushSegment(json.slice(lastIndex), lastIndex, "text-slate-700");
    }

    return nodes;
}

export function StatementDetailModal({
    open,
    detail,
    loading,
    error,
    canExport,
    canVoid,
    onClose,
    onExport,
    onVoid,
}: StatementDetailModalProps) {
    const [hoveredPath, setHoveredPath] = useState<string[] | null>(null);
    const [metadataOpen, setMetadataOpen] = useState(false);
    const metadataRows = detail
        ? [
              { label: "Organization", value: `${detail.metadata.organizationName} (${detail.metadata.organizationSlug})` },
              { label: "Organization ID", value: detail.metadata.organizationId },
              { label: "Project", value: `${detail.metadata.projectName} (${detail.metadata.projectSlug})` },
              { label: "Project ID", value: detail.metadata.projectId },
              { label: "LRS", value: detail.metadata.lrsName },
              { label: "LRS ID", value: detail.metadata.lrsId },
              { label: "Tenant ID", value: detail.metadata.tenantId },
              { label: "Data Store", value: `${detail.metadata.dataStoreName} (${detail.metadata.dataStoreType})` },
              { label: "Data Store ID", value: detail.metadata.dataStoreId },
              { label: "Client", value: detail.metadata.clientName ?? "-" },
              { label: "Client ID", value: detail.metadata.clientId ?? "-" },
          ]
        : [];
    const topLevelKeys = (() => {
        if (!detail?.raw || typeof detail.raw !== "object" || Array.isArray(detail.raw)) return [];
        const keys = Object.keys(detail.raw as Record<string, unknown>);
        const preferredOrder = ["id", "actor", "verb", "object", "result", "context", "timestamp", "stored", "authority", "version", "attachments"];
        const ordered = preferredOrder.filter((key) => keys.includes(key));
        const rest = keys.filter((key) => !preferredOrder.includes(key)).sort((a, b) => a.localeCompare(b));
        return [...ordered, ...rest];
    })();

    const footer = (
        <>
            <div className="min-w-0 break-all text-xs text-slate-500">{detail?.id ?? ""}</div>
            <div className="flex shrink-0 flex-wrap items-center gap-2">
                <button
                    onClick={onClose}
                    className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-50"
                >
                    Close
                </button>
                <button
                    onClick={onExport}
                    disabled={!detail || !canExport}
                    className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                >
                    <Download className="h-3.5 w-3.5" />
                    Export
                </button>
                <button
                    onClick={onVoid}
                    disabled={!detail || !canVoid}
                    className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-red-600 hover:bg-red-50 disabled:opacity-50"
                >
                    <Ban className="h-3.5 w-3.5" />
                    Void
                </button>
            </div>
        </>
    );

    return (
        <Modal open={open} onClose={onClose} title="Statement Details" footer={footer} maxWidth="max-w-7xl" showCloseButton={false}>
            {loading && <div className="text-sm text-slate-500 py-4">Loading...</div>}
            {error && <div className="text-sm text-red-600 py-2">{error}</div>}
            {!loading && detail && (
                <div className="grid grid-cols-1 xl:grid-cols-[minmax(0,1fr)_minmax(0,1.35fr)] gap-6">
                    <div>
                        <div className="text-sm font-semibold text-slate-700 mb-2">Raw</div>
                        <pre className="rounded-lg bg-slate-50 p-3 text-xs text-slate-700 overflow-auto max-h-[70vh]">
                            {renderHighlightedJSON(detail.raw, hoveredPath)}
                        </pre>
                    </div>
                    <div>
                        <button
                            type="button"
                            onClick={() => setMetadataOpen((prev) => !prev)}
                            aria-expanded={metadataOpen}
                            className="mb-2 flex w-full items-center justify-between rounded-lg border border-slate-200 bg-white px-3 py-2 text-left hover:bg-slate-50"
                        >
                            <span className="text-sm font-semibold text-slate-700">Metadata</span>
                            <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-slate-600">
                                {metadataOpen ? "Collapse" : "Expand"}
                                <ChevronDown className={`h-3.5 w-3.5 transition-transform ${metadataOpen ? "rotate-180" : ""}`} />
                            </span>
                        </button>
                        {metadataOpen && (
                            <div className="mb-4 grid grid-cols-1 gap-1 rounded-lg border border-slate-200 bg-white p-3">
                                {metadataRows.map((row) => (
                                    <div key={row.label} className="grid grid-cols-1 gap-0.5 text-xs sm:grid-cols-[140px_minmax(0,1fr)] sm:gap-2">
                                        <div className="font-semibold text-slate-500">{row.label}</div>
                                        <div className="break-all font-mono text-slate-700">{row.value}</div>
                                    </div>
                                ))}
                            </div>
                        )}
                        <div className="text-sm font-semibold text-slate-700 mb-2">Tree</div>
                        {topLevelKeys.length > 0 && (
                            <div className="mb-2 flex flex-wrap gap-1.5">
                                {topLevelKeys.map((key) => (
                                    <span
                                        key={key}
                                        onMouseEnter={() => setHoveredPath([key])}
                                        onMouseLeave={() => setHoveredPath((prev) => (prev?.length === 1 && prev[0] === key ? null : prev))}
                                        className={`inline-flex cursor-default items-center rounded-md border px-2 py-0.5 text-[11px] font-semibold transition ${
                                            getStatementTabColorClass(key)
                                        } ${hoveredPath?.length === 1 && hoveredPath[0] === key ? "scale-[1.02] shadow-sm" : ""}`}
                                    >
                                        {key}
                                    </span>
                                ))}
                            </div>
                        )}
                        <div className="max-h-[70vh] overflow-auto rounded-lg border border-slate-200 bg-slate-50 p-3">
                            <JSONTree data={detail.raw} onKeyHover={setHoveredPath} />
                        </div>
                    </div>
                </div>
            )}
        </Modal>
    );
}
