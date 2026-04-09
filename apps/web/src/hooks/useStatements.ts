import { useEffect, useState } from "react";
import {
    fetchStatementById,
    fetchStatementList,
    voidStatement as apiVoidStatement,
    downloadStatementJson,
} from "../api/statements";
import type { StatementDetail, StatementRow } from "../types/statement";

export type StatementFilters = {
    statementId: string;
    actorMbox: string;
    verbId: string;
    objectId: string;
    dateFrom: string;
    dateTo: string;
    limit: number;
};

const DEFAULT_FILTERS: StatementFilters = {
    statementId: "",
    actorMbox: "",
    verbId: "",
    objectId: "",
    dateFrom: "",
    dateTo: "",
    limit: 25,
};

export function useStatements() {
    const [filters, setFiltersState] = useState<StatementFilters>(DEFAULT_FILTERS);
    const [items, setItems] = useState<StatementRow[]>([]);
    const [nextCursor, setNextCursor] = useState<string | null>(null);
    const [cursorHistory, setCursorHistory] = useState<string[]>([]);
    const [pageIndex, setPageIndex] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Detail modal state
    const [detail, setDetail] = useState<StatementDetail | null>(null);
    const [detailOpen, setDetailOpen] = useState(false);
    const [detailLoading, setDetailLoading] = useState(false);
    const [detailError, setDetailError] = useState<string | null>(null);

    const setFilter = <K extends keyof StatementFilters>(field: K, value: StatementFilters[K]) => {
        setFiltersState((prev) => ({ ...prev, [field]: value }));
    };

    const load = async (cursor: string | null, resetHistory: boolean, currentFilters: StatementFilters) => {
        setLoading(true);
        setError(null);
        const controller = new AbortController();
        try {
            if (currentFilters.statementId.trim()) {
                const d = await fetchStatementById(currentFilters.statementId.trim(), controller.signal);
                setItems([
                    {
                        id: d.id,
                        actorMbox: d.actorMbox,
                        verbId: d.verbId,
                        objectId: d.objectId,
                        timestamp: d.timestamp ?? null,
                        storedAt: d.storedAt,
                    },
                ]);
                setNextCursor(null);
            } else {
                const data = await fetchStatementList(
                    {
                        limit: currentFilters.limit,
                        actorMbox: currentFilters.actorMbox.trim() || undefined,
                        verbId: currentFilters.verbId.trim() || undefined,
                        objectId: currentFilters.objectId.trim() || undefined,
                        since: currentFilters.dateFrom.trim() || undefined,
                        until: currentFilters.dateTo.trim() || undefined,
                        cursor,
                    },
                    controller.signal,
                );
                setItems((data.edges ?? []).map((e) => e.node));
                setNextCursor(data.pageInfo?.endCursor ?? null);
            }
            if (resetHistory) {
                setCursorHistory([]);
                setPageIndex(0);
            }
        } catch (err) {
            if ((err as Error).name !== "AbortError") {
                setError(err instanceof Error ? err.message : "Unknown error");
            }
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        void load(null, true, filters);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const applyFilters = () => void load(null, true, filters);
    const refresh = () => void load(null, true, filters);

    const goNext = () => {
        if (!nextCursor) return;
        setCursorHistory((prev) => {
            const next = prev.slice(0, pageIndex + 1);
            next.push(nextCursor);
            return next;
        });
        setPageIndex((p) => p + 1);
        void load(nextCursor, false, filters);
    };

    const goPrev = () => {
        if (pageIndex === 0) return;
        const prevCursor = pageIndex - 2 >= 0 ? cursorHistory[pageIndex - 2] : null;
        setPageIndex((p) => Math.max(0, p - 1));
        void load(prevCursor, false, filters);
    };

    const openDetail = async (id: string) => {
        setDetailOpen(true);
        setDetailLoading(true);
        setDetailError(null);
        try {
            const d = await fetchStatementById(id);
            setDetail(d);
        } catch (err) {
            setDetailError(err instanceof Error ? err.message : "Unknown error");
        } finally {
            setDetailLoading(false);
        }
    };

    const closeDetail = () => {
        setDetailOpen(false);
        setDetail(null);
        setDetailError(null);
    };

    const exportDetail = () => {
        if (detail) downloadStatementJson(detail);
    };

    const voidDetail = async () => {
        if (!detail) return;
        const ok = window.confirm(`Void statement ${detail.id}? This cannot be undone.`);
        if (!ok) return;
        await apiVoidStatement(detail.id);
        closeDetail();
        refresh();
    };

    return {
        filters,
        setFilter,
        items,
        loading,
        error,
        pageIndex,
        hasNextPage: !!nextCursor,
        applyFilters,
        refresh,
        goNext,
        goPrev,
        // detail
        detail,
        detailOpen,
        detailLoading,
        detailError,
        openDetail,
        closeDetail,
        exportDetail,
        voidDetail,
    };
}
