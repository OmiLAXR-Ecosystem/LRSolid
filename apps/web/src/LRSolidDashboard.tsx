import { BarChart2, Users, Calendar } from "lucide-react";
import {
    LineChart,
    Line,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    ResponsiveContainer,
    PieChart,
    Pie,
    Cell,
} from "recharts";

import { Card } from "./components/Card";

const kpis = [
    { label: "Total Statements", value: "12,450", icon: <BarChart2 className="h-6 w-6" /> },
    { label: "Active Users", value: "3,200", icon: <Users className="h-6 w-6" /> },
    { label: "Statements Today", value: "245", icon: <Calendar className="h-6 w-6" /> },
    { label: "Error Logs", value: "8", badge: "0", badgeColor: "bg-red-500" },
];

const activity = [
    { day: "Mon", a: 1200, b: 900 },
    { day: "Tue", a: 1800, b: 1200 },
    { day: "Wed", a: 1400, b: 1000 },
    { day: "Thu", a: 1900, b: 1500 },
    { day: "Fri", a: 1700, b: 1100 },
    { day: "Sat", a: 2400, b: 1700 },
    { day: "Sun", a: 2100, b: 1600 },
];

const engagement = [
    { name: "Sessions", value: 55 },
    { name: "Completions", value: 30 },
    { name: "Failures", value: 15 },
];

const engagementColors = ["#2F5DA8", "#2CBCC7", "#F26B5E"];

const recent = [
    { user: "john.doe", verb: "completed", time: "5 mins ago" },
    { user: "emma.smith", verb: "attempted", time: "12 mins ago" },
    { user: "alex.brown", verb: "viewed", time: "25 mins ago" },
];

function ProgressRow(props: { label: string; status: "Connected" | "Disconnected" }) {
    const connected = props.status === "Connected";
    return (
        <div className="flex items-center gap-4 py-3">
            <div className={`h-3 w-3 rounded-full ${connected ? "bg-emerald-500" : "bg-slate-400"}`} />
            <div className="flex-1">
                <div className="text-sm font-medium text-slate-900">{props.label}</div>
                <div className="mt-2 h-2 w-full rounded-full bg-slate-100">
                    <div
                        className={`h-2 rounded-full ${connected ? "bg-emerald-500" : "bg-slate-300"}`}
                        style={{ width: connected ? "92%" : "25%" }}
                    />
                </div>
            </div>
            <span
                className={`rounded-md px-3 py-1 text-xs font-semibold ${
                    connected ? "bg-emerald-500 text-white" : "bg-slate-200 text-slate-700"
                }`}
            >
                {props.status}
            </span>
        </div>
    );
}

export function LRSolidDashboard() {
    return (
        <div className="space-y-4">
            <div className="grid grid-cols-4 gap-4">
                {kpis.map((k) => (
                    <Card key={k.label} className="p-5">
                        <div className="flex items-start justify-between">
                            <div>
                                <div className="text-sm font-semibold text-slate-700">{k.label}</div>
                                <div className="mt-2 text-3xl font-extrabold text-slate-900">{k.value}</div>
                            </div>
                            <div className="text-teal-500">{k.icon}</div>
                        </div>

                        {"badge" in k && (
                            <div className="mt-3 flex justify-end">
                                <span
                                    className={`inline-flex items-center justify-center h-7 min-w-7 px-2 rounded-lg text-xs font-bold text-white ${k.badgeColor}`}
                                >
                                    {k.badge}
                                </span>
                            </div>
                        )}
                    </Card>
                ))}
            </div>

            <div className="grid grid-cols-2 gap-6">
                <Card title="Activity Overview">
                    <div className="h-[260px]">
                        <ResponsiveContainer width="100%" height="100%">
                            <LineChart data={activity} margin={{ top: 10, right: 20, bottom: 0, left: 0 }}>
                                <CartesianGrid strokeDasharray="4 4" />
                                <XAxis dataKey="day" />
                                <YAxis />
                                <Tooltip />
                                <Line type="monotone" dataKey="a" stroke="#2CBCC7" strokeWidth={3} dot={false} />
                                <Line type="monotone" dataKey="b" stroke="#2F5DA8" strokeWidth={3} dot={false} />
                            </LineChart>
                        </ResponsiveContainer>
                    </div>
                </Card>

                <Card title="User Engagement">
                    <div className="grid grid-cols-[260px_1fr] gap-6 items-center">
                        <div className="h-[240px]">
                            <ResponsiveContainer width="100%" height="100%">
                                <PieChart>
                                    <Pie
                                        data={engagement}
                                        dataKey="value"
                                        nameKey="name"
                                        cx="50%"
                                        cy="50%"
                                        innerRadius={55}
                                        outerRadius={90}
                                        paddingAngle={2}
                                    >
                                        {engagement.map((_, i) => (
                                            <Cell key={i} fill={engagementColors[i]} />
                                        ))}
                                    </Pie>
                                    <Tooltip />
                                </PieChart>
                            </ResponsiveContainer>
                        </div>

                        <div className="space-y-4">
                            {engagement.map((item, i) => (
                                <div
                                    key={item.name}
                                    className="flex items-center justify-between text-sm font-semibold text-slate-700"
                                >
                                    <div className="flex items-center gap-3">
                                        <span className="h-3 w-3 rounded-full" style={{ backgroundColor: engagementColors[i] }} />
                                        {item.name}
                                    </div>
                                    <span className="text-slate-600">{item.value}%</span>
                                </div>
                            ))}
                        </div>
                    </div>
                </Card>
            </div>

            <div className="grid grid-cols-2 gap-6">
                <Card title="Recent Statements">
                    <div className="space-y-4">
                        {recent.map((row) => (
                            <div key={row.user} className="flex items-center justify-between text-sm">
                                <div className="flex items-center gap-3">
                                    <div className="h-9 w-9 rounded-full bg-slate-100 ring-1 ring-slate-200" />
                                    <div>
                                        <div className="font-semibold text-slate-800">{row.user}</div>
                                        <div className="text-xs text-slate-500">{row.verb}</div>
                                    </div>
                                </div>
                                <div className="text-slate-500">{row.time}</div>
                            </div>
                        ))}
                        <div className="text-right">
                            <a href="#/statements" className="text-sm font-semibold text-teal-600 hover:text-teal-700">
                                View All
                            </a>
                        </div>
                    </div>
                </Card>

                <Card title="Data Store Status">
                    <ProgressRow label="PostgreSQL" status="Connected" />
                    <ProgressRow label="MongoDB" status="Connected" />
                    <div className="mt-3 text-right">
                        <button className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50">
                            Manage Sources
                        </button>
                    </div>
                </Card>
            </div>
        </div>
    );
}
