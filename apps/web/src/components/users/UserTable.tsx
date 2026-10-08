import { ArrowUpDown, MoreHorizontal, Pencil } from "lucide-react";
import { Card } from "../ui/Card";
import { Pagination } from "../ui/Pagination";
import type { RestUser } from "../../types/user";

type UserTableProps = {
    items: RestUser[];
    loading: boolean;
    error: string | null;
    limit: number;
    onLimitChange: (limit: number) => void;
    page: number;
    totalPages: number;
    pageStart: number;
    pageEnd: number;
    total: number;
    onPrev: () => void;
    onNext: () => void;
};

function SortLabel({ children }: { children: React.ReactNode }) {
    return (
        <span className="inline-flex items-center gap-1">
            {children}
            <ArrowUpDown className="h-3 w-3 text-slate-400" />
        </span>
    );
}

export function UserTable({
    items,
    loading,
    error,
    limit,
    onLimitChange,
    page,
    totalPages,
    pageStart,
    pageEnd,
    total,
    onPrev,
    onNext,
}: UserTableProps) {
    return (
        <Card>
            {error && <div className="mb-3 text-sm text-red-600">Error: {error}</div>}
            <div className="overflow-x-auto rounded-lg ring-1 ring-slate-200">
                <table className="min-w-full text-sm">
                    <thead className="bg-slate-50 text-slate-600">
                        <tr>
                            <th className="px-4 py-3 text-left font-semibold">Name</th>
                            <th className="px-4 py-3 text-left font-semibold">Email</th>
                            <th className="px-4 py-3 text-left font-semibold">Role</th>
                            <th className="px-4 py-3 text-left font-semibold">
                                <SortLabel>Last Activity</SortLabel>
                            </th>
                            <th className="px-4 py-3 text-right font-semibold">Actions</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                        {loading && (
                            <tr>
                                <td colSpan={5} className="px-4 py-6 text-center text-sm text-slate-500">
                                    Loading users...
                                </td>
                            </tr>
                        )}
                        {!loading && items.length === 0 && (
                            <tr>
                                <td colSpan={5} className="px-4 py-6 text-center text-sm text-slate-500">
                                    No users found.
                                </td>
                            </tr>
                        )}
                        {!loading && items.map((row) => (
                            <tr key={row.id} className="bg-white">
                                <td className="px-4 py-3 text-slate-700">{row.displayName ?? row.email}</td>
                                <td className="px-4 py-3 text-slate-600">{row.email}</td>
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
                    </tbody>
                </table>
            </div>

            <Pagination
                label={`Showing ${pageStart} to ${pageEnd} of ${total} entries`}
                limit={limit}
                onLimitChange={onLimitChange}
                onPrev={onPrev}
                onNext={onNext}
                prevDisabled={page === 0 || loading}
                nextDisabled={page >= totalPages - 1 || loading}
                currentPage={page + 1}
                totalPages={totalPages}
            />
        </Card>
    );
}
