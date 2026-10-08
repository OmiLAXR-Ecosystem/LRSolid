import { useState } from "react";
import { useUsers } from "../hooks/useUsers";
import { useOrg } from "../contexts/OrgContext";
import { Card } from "../components/ui/Card";
import { FieldLabel } from "../components/ui/FieldLabel";
import { TextInput } from "../components/ui/TextInput";
import { SelectInput } from "../components/ui/SelectInput";
import { Pagination } from "../components/ui/Pagination";
import { InviteUserDialog } from "../components/users/InviteUserDialog";

const ROLE_OPTIONS = [
    { label: "All roles", value: "All" },
    { label: "Admin", value: "Admin" },
    { label: "User", value: "User" },
];

export function UsersView() {
    const u = useUsers();
    const { currentOrg } = useOrg();
    const [inviteOpen, setInviteOpen] = useState(false);

    return (
        <div className="space-y-6">
            <Card>
                <div className="flex flex-col items-stretch gap-3 sm:flex-row sm:items-end">
                    <div className="min-w-0 flex-1">
                        <FieldLabel>Search</FieldLabel>
                        <TextInput value={u.query} onChange={u.setQuery} placeholder="Name or email..." />
                    </div>
                    <div className="w-full sm:w-40">
                        <FieldLabel>Role</FieldLabel>
                        <SelectInput value={u.role} onChange={u.setRole} options={ROLE_OPTIONS} />
                    </div>
                    <button
                        onClick={() => setInviteOpen(true)}
                        className="h-10 w-full whitespace-nowrap rounded-lg bg-teal-600 px-4 text-sm font-semibold text-white transition hover:bg-teal-700 sm:w-auto"
                    >
                        Benutzer einladen
                    </button>
                </div>
            </Card>

            <Card>
                {u.error && <div className="mb-3 text-sm text-red-600">Error: {u.error}</div>}
                <div className="overflow-x-auto rounded-lg ring-1 ring-slate-200">
                    <table className="min-w-full text-sm">
                        <thead className="bg-slate-50 text-slate-600">
                            <tr>
                                <th className="px-4 py-3 text-left font-semibold">Name</th>
                                <th className="px-4 py-3 text-left font-semibold">Email</th>
                                <th className="px-4 py-3 text-left font-semibold">Role</th>
                                <th className="px-4 py-3 text-right font-semibold">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {u.loading && (
                                <tr>
                                    <td colSpan={4} className="px-4 py-6 text-center text-sm text-slate-500">
                                        Loading...
                                    </td>
                                </tr>
                            )}
                            {!u.loading && u.items.length === 0 && (
                                <tr>
                                    <td colSpan={4} className="px-4 py-6 text-center text-sm text-slate-500">
                                        No users found.
                                    </td>
                                </tr>
                            )}
                            {!u.loading && u.items.map((row) => (
                                <tr key={row.id} className="bg-white">
                                    <td className="px-4 py-3 font-medium text-slate-800">
                                        {row.displayName ?? row.email}
                                    </td>
                                    <td className="px-4 py-3 text-slate-600">{row.email}</td>
                                    <td className="px-4 py-3 text-slate-600">{row.role}</td>
                                    <td className="px-4 py-3 text-right">
                                        <button className="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-50">
                                            Edit
                                        </button>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
                <Pagination
                    label={`Showing ${u.pageStart} to ${u.pageEnd} of ${u.total} entries`}
                    limit={u.limit}
                    onLimitChange={u.setLimit}
                    onPrev={() => u.setPage((p) => Math.max(0, p - 1))}
                    onNext={() => u.setPage((p) => Math.min(u.totalPages - 1, p + 1))}
                    prevDisabled={u.page === 0 || u.loading}
                    nextDisabled={u.page >= u.totalPages - 1 || u.loading}
                    currentPage={u.page + 1}
                    totalPages={u.totalPages}
                />
            </Card>

            {currentOrg && (
                <InviteUserDialog
                    open={inviteOpen}
                    org={currentOrg}
                    onClose={() => setInviteOpen(false)}
                    onSuccess={() => setInviteOpen(false)}
                />
            )}
        </div>
    );
}
