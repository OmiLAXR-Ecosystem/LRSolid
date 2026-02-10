import { ChevronDown, MoreHorizontal, Pencil, ArrowUpDown } from "lucide-react";
import { AdminShell } from "../components/AdminShell";
import { Card } from "../components/Card";

const users = [
    { name: "John Doe", email: "john.doe@example.com", role: "Admin", role2: "Admin", last: "2024-04-22 15:23" },
    { name: "Jane Smith", email: "jane.smith@example.com", role: "Admin", role2: "Admin", last: "2024-04-21 14:12" },
    { name: "Michael Lee", email: "michael.lee@example.com", role: "Admin", role2: "Admin", last: "2024-04-20 13:05" },
    { name: "Emily Clark", email: "emily.clark@example.com", role: "Admin", role2: "Admin", last: "2024-04-19 12:54" },
    { name: "Robert Brown", email: "robert.brown@example.com", role: "User", role2: "User", last: "2024-04-18 11:30" },
];

function FieldLabel({ children }: { children: React.ReactNode }) {
    return <div className="text-xs font-semibold text-slate-600">{children}</div>;
}

function TextBox({ placeholder }: { placeholder: string }) {
    return (
        <input
            type="text"
            placeholder={placeholder}
            className="w-full rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300"
        />
    );
}

function SelectBox({ value }: { value: string }) {
    return (
        <div className="relative">
            <select className="w-full appearance-none rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300">
                <option>{value}</option>
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
    return (
        <AdminShell activeKey="users" searchPlaceholder="Search users...">
            <Card>
                <div className="flex items-end justify-between gap-6">
                    <div className="grid grid-cols-[1fr_1fr_0.8fr] gap-4 flex-1">
                        <div>
                            <FieldLabel>Name:</FieldLabel>
                            <TextBox placeholder="Name..." />
                        </div>
                        <div>
                            <FieldLabel>Email:</FieldLabel>
                            <TextBox placeholder="Email..." />
                        </div>
                        <div>
                            <FieldLabel>Role:</FieldLabel>
                            <SelectBox value="All" />
                        </div>
                    </div>
                    <button className="h-10 rounded-lg bg-teal-600 px-4 text-sm font-semibold text-white hover:bg-teal-700 transition">
                        Add User
                    </button>
                </div>
            </Card>

            <Card>
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
                            {users.map((row) => (
                                <tr key={`${row.email}-${row.last}`} className="bg-white">
                                    <td className="px-4 py-3 text-slate-700">{row.name}</td>
                                    <td className="px-4 py-3 text-slate-600">{row.email}</td>
                                    <td className="px-4 py-3 text-slate-700">{row.role}</td>
                                    <td className="px-4 py-3 text-slate-700">{row.role2}</td>
                                    <td className="px-4 py-3 text-slate-600">{row.last}</td>
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

                <div className="mt-4 flex items-center justify-between text-sm text-slate-600">
                    <div>Showing 1 to 5 of 45 entries</div>
                    <div className="flex items-center gap-2">
                        <button className="rounded-md border border-slate-200 px-2.5 py-1 text-slate-600 hover:bg-slate-50">Previous</button>
                        <button className="h-8 w-8 rounded-md border border-slate-200 bg-white text-slate-600">1</button>
                        <button className="h-8 w-8 rounded-md bg-teal-600 text-white">2</button>
                        <button className="h-8 w-8 rounded-md border border-slate-200 bg-white text-slate-600">3</button>
                        <button className="h-8 w-8 rounded-md border border-slate-200 bg-white text-slate-600">…</button>
                        <button className="h-8 w-8 rounded-md border border-slate-200 bg-white text-slate-600">5</button>
                        <button className="rounded-md border border-slate-200 px-2.5 py-1 text-slate-600 hover:bg-slate-50">Next</button>
                    </div>
                </div>
            </Card>
        </AdminShell>
    );
}
