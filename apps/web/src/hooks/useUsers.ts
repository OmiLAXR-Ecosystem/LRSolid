import { useEffect, useState } from "react";
import { fetchUsers } from "../api/users";
import type { RestUser } from "../types/user";

export function useUsers() {
    const [query, setQuery] = useState("");
    const [role, setRole] = useState("All");
    const [limit, setLimit] = useState(25);
    const [page, setPage] = useState(0);
    const [items, setItems] = useState<RestUser[]>([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        setPage(0);
    }, [query, role, limit]);

    useEffect(() => {
        let cancelled = false;
        const run = async () => {
            setLoading(true);
            setError(null);
            try {
                const data = await fetchUsers({
                    limit,
                    offset: page * limit,
                    q: query.trim() || undefined,
                    role,
                });
                if (!cancelled) {
                    setItems(data.items ?? []);
                    setTotal(data.total ?? 0);
                }
            } catch (err) {
                if (!cancelled) setError(err instanceof Error ? err.message : "Request failed");
            } finally {
                if (!cancelled) setLoading(false);
            }
        };
        void run();
        return () => { cancelled = true; };
    }, [limit, page, query, role]);

    const totalPages = Math.max(1, Math.ceil(total / Math.max(1, limit)));
    const pageStart = total === 0 ? 0 : page * limit + 1;
    const pageEnd = Math.min(total, (page + 1) * limit);

    return {
        query, setQuery,
        role, setRole,
        limit, setLimit,
        page, setPage,
        items,
        total,
        loading,
        error,
        totalPages,
        pageStart,
        pageEnd,
    };
}
