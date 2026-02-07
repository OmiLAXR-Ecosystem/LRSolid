import { Calendar, ChevronDown, MoreHorizontal } from "lucide-react";
import { AdminShell } from "../components/AdminShell";
import { Card } from "../components/Card";

const statementRows = [
    { actor: "John Doe", verb: "completed", object: "Course A", time: "2022-01-15 14:32" },
    { actor: "Jane Smith", verb: "attempted", object: "Quiz B", time: "2022-01-14 10:20" },
    { actor: "Michael Lee", verb: "passed", object: "Lesson 1", time: "2022-01-13 09:45" },
    { actor: "Emily Clark", verb: "viewed", object: "Video X", time: "2022-01-12 16:10" },
    { actor: "Robert Brown", verb: "failed", object: "Test Y", time: "2022-01-10 11:05" },
];

function FieldLabel({ children }: { children: React.ReactNode }) {
    return <div className="text-xs font-semibold text-slate-600">{children}</div>;
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

function DateBox({ placeholder }: { placeholder: string }) {
    return (
        <div className="relative">
            <input
                type="text"
                placeholder={placeholder}
                className="w-full rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300"
            />
            <Calendar className="h-4 w-4 text-slate-400 absolute right-3 top-1/2 -translate-y-1/2" />
        </div>
    );
}

export function StatementsView() {
    return (
        <AdminShell activeKey="statements" searchPlaceholder="Search statements...">
            <div className="text-2xl font-semibold text-slate-800">Statements</div>
            <Card>
                <div className="grid grid-cols-[1.1fr_1.1fr_1.1fr_1fr_1fr_auto] gap-4 items-end">
                    <div>
                        <FieldLabel>Actor:</FieldLabel>
                        <SelectBox value="All" />
                    </div>
                    <div>
                        <FieldLabel>Verb:</FieldLabel>
                        <SelectBox value="All" />
                    </div>
                    <div>
                        <FieldLabel>Object:</FieldLabel>
                        <SelectBox value="All" />
                    </div>
                    <div>
                        <FieldLabel>Date from:</FieldLabel>
                        <DateBox placeholder="01/01/2022" />
                    </div>
                    <div>
                        <FieldLabel>Date to:</FieldLabel>
                        <DateBox placeholder="01/30/2022" />
                    </div>
                    <button className="h-10 rounded-lg bg-teal-600 px-4 text-sm font-semibold text-white hover:bg-teal-700 transition">
                        Apply Filters
                    </button>
                </div>
            </Card>

            <Card>
                <div className="overflow-hidden rounded-lg ring-1 ring-slate-200">
                    <table className="min-w-full text-sm">
                        <thead className="bg-slate-50 text-slate-600">
                            <tr>
                                <th className="px-4 py-3 text-left font-semibold">Actor</th>
                                <th className="px-4 py-3 text-left font-semibold">Verb</th>
                                <th className="px-4 py-3 text-left font-semibold">Object</th>
                                <th className="px-4 py-3 text-left font-semibold">Timestamp</th>
                                <th className="px-4 py-3 text-right font-semibold">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {statementRows.map((row) => (
                                <tr key={`${row.actor}-${row.time}`} className="bg-white">
                                    <td className="px-4 py-3 text-slate-700">{row.actor}</td>
                                    <td className="px-4 py-3 text-slate-700">{row.verb}</td>
                                    <td className="px-4 py-3 text-slate-700">{row.object}</td>
                                    <td className="px-4 py-3 text-slate-600">{row.time}</td>
                                    <td className="px-4 py-3 text-right">
                                        <div className="inline-flex items-center gap-2">
                                            <button className="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-teal-600 hover:bg-teal-50">
                                                View Details
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
                    <div>Showing 1 to 5 of 25 entries</div>
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
