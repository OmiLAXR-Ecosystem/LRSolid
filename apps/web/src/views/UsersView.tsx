import { ChevronDown, MoreHorizontal, Pencil, ArrowUpDown } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Card } from "../components/Card";

type RestUser = {
    id: string;
    displayName?: string | null;
    email: string;
    role: string;
    disabledAt?: string | null;
};

type RestUserResponse = {
    items: RestUser[];
    total: number;
    limit: number;
    offset: number;
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

function TextBox({
    placeholder,
    value,
    onChange,
}: {
    placeholder: string;
    value: string;
    onChange: (value: string) => void;
}) {
    return (
        <input
            type="text"
            value={value}
            onChange={(event) => onChange(event.target.value)}
            placeholder={placeholder}
            className="w-full rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300"
        />
    );
}

function SelectBox({
    value,
    options,
    onChange,
}: {
    value: string;
    options: string[];
    onChange: (value: string) => void;
}) {
    return (
        <div className="relative">
            <select
                value={value}
                onChange={(event) => onChange(event.target.value)}
                className="w-full appearance-none rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300"
            >
                {options.map((option) => (
                    <option key={option} value={option}>
                        {option}
                    </option>
                ))}
            </select>
            <ChevronDown className="h-4 w-4 text-slate-400 absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none" />
        </div>
    );
}

function SortLabel({ children }: { children: React.ReactNode }) {
    return (
        <span className="inline-flex items-center gap-1">
            {children}
            <ArrowUpDown className="h-3 w-3 text-slate-400" />
        </span>
    );
}

export function UsersView() {
    const [name, setName] = useState("");
    const [email, setEmail] = useState("");
    const [role, setRole] = useState("All");
    const [limit, setLimit] = useState(25);
    const [page, setPage] = useState(0);
    const [items, setItems] = useState<RestUser[]>([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const roleOptions = ["All", "Admin", "User"];
    const totalPages = Math.max(1, Math.ceil(total / Math.max(1, limit)));
    const pageStart = total === 0 ? 0 : page * limit + 1;
    const pageEnd = Math.min(total, (page + 1) * limit);

    const query = useMemo(() => {
        const q = [name.trim(), email.trim()].filter(Boolean).join(" ");
        return q;
    }, [name, email]);

    useEffect(() => {
        const fetchUsers = async () => {
            setLoading(true);
            setError(null);
            try {
                const params = new URLSearchParams();
                params.set("limit", String(limit));
                params.set("offset", String(page * limit));
                if (query) params.set("q", query);
                if (role && role !== "All") params.set("role", role);

                const res = await fetch(`${joinApi("/api/v1/user")}?${params.toString()}`);
                if (!res.ok) {
                    throw new Error(`Request failed (${res.status})`);
                }
                const data = (await res.json()) as RestUserResponse;
                setItems(data.items ?? []);
                setTotal(data.total ?? 0);
            } catch (err) {
                setError(err instanceof Error ? err.message : "Request failed");
            } finally {
                setLoading(false);
            }
        };

        fetchUsers();
    }, [limit, page, query, role]);

    useEffect(() => {
        setPage(0);
    }, [query, role, limit]);

    return (
        <div className="space-y-6">
            <Card>
                <div className="flex items-end justify-between gap-6">
                    <div className="grid grid-cols-[1fr_1fr_0.8fr] gap-4 flex-1">
                        <div>
                            <FieldLabel>Name:</FieldLabel>
                            <TextBox placeholder="Name..." value={name} onChange={setName} />
                        </div>
                        <div>
                            <FieldLabel>Email:</FieldLabel>
                            <TextBox placeholder="Email..." value={email} onChange={setEmail} />
                        </div>
                        <div>
                            <FieldLabel>Role:</FieldLabel>
                            <SelectBox value={role} options={roleOptions} onChange={setRole} />
                        </div>
                    </div>
                    <button className="h-10 rounded-lg bg-teal-600 px-4 text-sm font-semibold text-white hover:bg-teal-700 transition">
                        Add User
                    </button>
                </div>
            </Card>

            <Card>
                {error && <div className="mb-3 text-sm text-red-600">Error: {error}</div>}
                <div className="overflow-hidden rounded-lg ring-1 ring-slate-200">
                    <table className="min-w-full text-sm">
                        <thead className="bg-slate-50 text-slate-600">
                            <tr>
                                <th className="px-4 py-3 text-left font-semibold">Name</th>
                                <th className="px-4 py-3 text-left font-semibold">Email</th>
                                <th className="px-4 py-3 text-left font-semibold">Role</th>
                                <th className="px-4 py-3 text-left font-semibold">
                                    <SortLabel>Role</SortLabel>
                                </th>
                                <th className="px-4 py-3 text-left font-semibold">
                                    <SortLabel>Last Activity</SortLabel>
                                </th>
                                <th className="px-4 py-3 text-right font-semibold">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {items.map((row) => (
                                <tr key={row.id} className="bg-white">
                                    <td className="px-4 py-3 text-slate-700">{row.displayName ?? row.email}</td>
                                    <td className="px-4 py-3 text-slate-600">{row.email}</td>
                                    <td className="px-4 py-3 text-slate-700">{row.role}</td>
                                    <td className="px-4 py-3 text-slate-700">{row.role}</td>
                                    <td className="px-4 py-3 text-slate-600">—</td>
                                    <td className="px-4 py-3 text-right">
                                        <div className="inline-flex items-center gap-2">
                                            <button className="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-teal-600 hover:bg-teal-50 inline-flex items-center gap-2">
                                                <Pencil className="h-3.5 w-3.5" />
                                                Edit
                                            </button>
                                            <button className="rounded-lg border border-slate-200 p-2 text-slate-500 hover:bg-slate-50">
                                                <MoreHorizontal className="h-4 w-4" />
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            ))}
                            {!loading && items.length === 0 && (
                                <tr>
                                    <td colSpan={6} className="px-4 py-6 text-center text-sm text-slate-500">
                                        No users found.
                                    </td>
                                </tr>
                            )}
                            {loading && (
                                <tr>
                                    <td colSpan={6} className="px-4 py-6 text-center text-sm text-slate-500">
                                        Loading users...
                                    </td>
                                </tr>
                            )}
                        </tbody>
                    </table>
                </div>

                <div className="mt-4 flex items-center justify-between text-sm text-slate-600">
                    <div>
                        Showing {pageStart} to {pageEnd} of {total} entries
                    </div>
                    <div className="flex items-center gap-2">
                        <label className="text-xs font-semibold text-slate-500">Limit</label>
                        <select
                            value={limit}
                            onChange={(event) => setLimit(Number(event.target.value))}
                            className="rounded-md border border-slate-200 bg-white px-2 py-1 text-sm"
                        >
                            {[10, 25, 50, 100].map((value) => (
                                <option key={value} value={value}>
                                    {value}
                                </option>
                            ))}
                        </select>
                        <button
                            className="rounded-md border border-slate-200 px-2.5 py-1 text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                            onClick={() => setPage((p) => Math.max(0, p - 1))}
                            disabled={page === 0 || loading}
                        >
                            Previous
                        </button>
                        <div className="flex items-center gap-2">
                            <span className="rounded-md border border-slate-200 bg-white px-2.5 py-1 text-slate-600">
                                {page + 1}
                            </span>
                            <span className="text-slate-400">/</span>
                            <span className="text-slate-600">{totalPages}</span>
                        </div>
                        <button
                            className="rounded-md border border-slate-200 px-2.5 py-1 text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                            onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                            disabled={page >= totalPages - 1 || loading}
                        >
                            Next
                        </button>
                    </div>
                </div>
            </Card>
        </div>
    );
}
