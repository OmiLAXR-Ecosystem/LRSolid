import React from "react";
import {
    Bell,
    Settings,
    Calendar,
    ChevronDown,
    LayoutDashboard,
    FileText,
    Users,
    BarChart2,
    Database,
    Store,
    Plus,
    Search,
} from "lucide-react";
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

import Logo from "./assets/LRSolid_Logo.png";

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

function Card(props: { title?: string; className?: string; children: React.ReactNode }) {
    return (
        <div className={`rounded-xl bg-white shadow-sm ring-1 ring-slate-200 ${props.className ?? ""}`}>
            {props.title && (
                <div className="px-5 pt-4 pb-2 text-sm font-semibold text-slate-800">{props.title}</div>
            )}
            <div className={props.title ? "px-5 pb-5" : "p-5"}>{props.children}</div>
        </div>
    );
}

function NavItem(props: { icon: React.ReactNode; label: string; active?: boolean; href?: string }) {
    const className = [
        "w-full flex items-center gap-3 rounded-lg px-4 py-3 text-sm font-medium transition",
        props.active
            ? "bg-white/90 text-slate-900 shadow-sm"
            : "text-white/90 hover:bg-white/10 hover:text-white",
    ].join(" ");

    if (props.href) {
        return (
            <a href={props.href} className={className} aria-current={props.active ? "page" : undefined}>
                <span className="h-5 w-5">{props.icon}</span>
                <span>{props.label}</span>
            </a>
        );
    }

    return (
        <button type="button" className={className}>
            <span className="h-5 w-5">{props.icon}</span>
            <span>{props.label}</span>
        </button>
    );
}

function ProgressRow(props: { label: string; status: "Connected" | "Disconnected" }) {
    const connected = props.status === "Connected";
    return (
        <div className="flex items-center gap-4 py-3">
            <div className={`h-3 w-3 rounded-full ${connected ? "bg-emerald-500" : "bg-slate-400"}`} />
            <div className="flex-1">
                <div className="text-sm font-medium text-slate-900">{props.label}</div>
                <div className="mt-2 h-2 w-full rounded-full bg-slate-100">
                    <div className={`h-2 rounded-full ${connected ? "bg-emerald-500" : "bg-slate-300"}`} style={{ width: connected ? "92%" : "25%" }} />
                </div>
            </div>
            <span className={`rounded-md px-3 py-1 text-xs font-semibold ${connected ? "bg-emerald-500 text-white" : "bg-slate-200 text-slate-700"}`}>
        {props.status}
      </span>
        </div>
    );
}

export function LRSolidDashboard() {
    return (
        <div className="min-h-screen bg-slate-100">
            <div className="p-6">
                <div className="grid grid-cols-[260px_1fr] gap-6 min-h-[calc(100vh-3rem)]">
                    {/* Sidebar */}
                    <aside className="rounded-2xl bg-gradient-to-b from-teal-600 to-cyan-700 text-white shadow-sm overflow-hidden">
                        <div className="flex items-center gap-3 px-5 py-5 border-b border-white/15">
                            <img src={Logo} />
                        </div>

                        <div className="p-4 space-y-2">
                            <NavItem icon={<LayoutDashboard />} label="Dashboard" active href="#/dashboard" />
                            <NavItem icon={<FileText />} label="Statements" href="#/statements" />
                            <NavItem icon={<Users />} label="Users" href="#/users" />
                            <NavItem icon={<BarChart2 />} label="Reports" href="#/reports" />
                            <NavItem icon={<Database />} label="Datasources" href="#/datasources" />
                            <NavItem icon={<Store />} label="Plugin Marketplace" href="#/marketplace" />
                        </div>

                        <div className="mt-auto p-4">
                            <a
                                href="#/settings"
                                className="w-full flex items-center gap-3 rounded-xl bg-white/10 px-4 py-3 text-sm font-semibold hover:bg-white/15 transition"
                            >
                                <Plus className="h-5 w-5" />
                                Settings
                            </a>
                        </div>
                    </aside>

                    {/* Main */}
                    <main className="space-y-6">
                        {/* Topbar */}
                        <div className="rounded-2xl bg-white shadow-sm ring-1 ring-slate-200 px-5 py-4 flex items-center justify-between">
                            <div className="flex items-center gap-3 w-[520px] max-w-full">
                                <div className="relative w-full">
                                    <Search className="h-4 w-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
                                    <input
                                        className="w-full rounded-xl bg-slate-50 ring-1 ring-slate-200 px-10 py-2.5 text-sm outline-none focus:ring-2 focus:ring-teal-300"
                                        placeholder="Search..."
                                    />
                                </div>
                            </div>

                            <div className="flex items-center gap-4">
                                <button className="h-10 w-10 rounded-xl ring-1 ring-slate-200 bg-white hover:bg-slate-50 grid place-items-center">
                                    <Bell className="h-5 w-5 text-slate-600" />
                                </button>
                                <button className="h-10 w-10 rounded-xl ring-1 ring-slate-200 bg-white hover:bg-slate-50 grid place-items-center">
                                    <Settings className="h-5 w-5 text-slate-600" />
                                </button>
                                <button className="h-10 w-10 rounded-xl ring-1 ring-slate-200 bg-white hover:bg-slate-50 grid place-items-center">
                                    <Calendar className="h-5 w-5 text-slate-600" />
                                </button>

                                <div className="flex items-center gap-2 pl-2">
                                    <div className="h-10 w-10 rounded-full bg-slate-200 ring-1 ring-slate-200" />
                                    <ChevronDown className="h-4 w-4 text-slate-500" />
                                </div>
                            </div>
                        </div>

                        {/* KPI row */}
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
                      <span className={`inline-flex items-center justify-center h-7 min-w-7 px-2 rounded-lg text-xs font-bold text-white ${k.badgeColor}`}>
                        {k.badge}
                      </span>
                                        </div>
                                    )}
                                </Card>
                            ))}
                        </div>

                        {/* Charts row */}
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
                                        {engagement.map((e, i) => (
                                            <div key={e.name} className="flex items-center justify-between gap-4">
                                                <div className="flex items-center gap-3">
                                                    <span className="h-3 w-3 rounded-sm" style={{ background: engagementColors[i] }} />
                                                    <span className="text-sm font-semibold text-slate-800">{e.name}</span>
                                                </div>
                                                <span className="text-sm font-semibold text-slate-700">{e.value}%</span>
                                            </div>
                                        ))}
                                        <div className="pt-2">
                                            <button className="text-sm font-semibold text-teal-600 hover:text-teal-700">
                                                View All <span className="ml-1">→</span>
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            </Card>
                        </div>

                        {/* Tables row */}
                        <div className="grid grid-cols-2 gap-6">
                            <Card title="Recent Statements">
                                <div className="overflow-hidden rounded-xl ring-1 ring-slate-200">
                                    <div className="grid grid-cols-[1fr_1fr_120px] bg-slate-50 px-4 py-2 text-xs font-bold text-slate-600">
                                        <div>User</div>
                                        <div className="text-center">Verb</div>
                                        <div className="text-right">Timestamp</div>
                                    </div>
                                    {recent.map((r) => (
                                        <div key={r.user} className="grid grid-cols-[1fr_1fr_120px] items-center px-4 py-3 border-t border-slate-200">
                                            <div className="flex items-center gap-3">
                                                <div className="h-7 w-7 rounded-full bg-slate-200 ring-1 ring-slate-200" />
                                                <div className="text-sm font-semibold text-slate-900">{r.user}</div>
                                            </div>
                                            <div className="text-center text-sm text-slate-700">{r.verb}</div>
                                            <div className="text-right text-sm text-slate-600">{r.time}</div>
                                        </div>
                                    ))}
                                </div>
                                <div className="pt-3">
                                    <button className="text-sm font-semibold text-teal-600 hover:text-teal-700">
                                        View All <span className="ml-1">↗</span>
                                    </button>
                                </div>
                            </Card>

                            <Card title="Datasource Status">
                                <div className="space-y-2">
                                    <ProgressRow label="PostgreSQL" status="Connected" />
                                    <ProgressRow label="MongoDB" status="Connected" />
                                    <div className="pt-2 flex justify-end">
                                        <button className="inline-flex items-center gap-2 rounded-xl px-4 py-2 text-sm font-semibold ring-1 ring-slate-200 bg-white hover:bg-slate-50">
                                            Manage Sources <ChevronDown className="h-4 w-4 text-slate-500" />
                                        </button>
                                    </div>
                                </div>
                            </Card>
                        </div>

                        {/* Marketplace */}
                        <Card title="Plugin Marketplace">
                            <div className="grid grid-cols-4 gap-4">
                                <div className="rounded-xl ring-1 ring-slate-200 bg-white p-4 flex items-center gap-3">
                                    <div className="h-10 w-10 rounded-xl bg-slate-100 grid place-items-center font-black text-slate-700">📊</div>
                                    <div>
                                        <div className="text-sm font-bold text-slate-900">Analytics Plus</div>
                                        <div className="text-xs text-slate-600">Advanced Reporting</div>
                                    </div>
                                </div>
                                <div className="rounded-xl ring-1 ring-slate-200 bg-white p-4 flex items-center gap-3">
                                    <div className="h-10 w-10 rounded-xl bg-slate-100 grid place-items-center font-black text-slate-700">🏆</div>
                                    <div>
                                        <div className="text-sm font-bold text-slate-900">Gamification Pack</div>
                                        <div className="text-xs text-slate-600">Badges & Rewards</div>
                                    </div>
                                </div>
                                <div className="rounded-xl ring-1 ring-slate-200 bg-white p-4 flex items-center gap-3">
                                    <div className="h-10 w-10 rounded-xl bg-slate-100 grid place-items-center font-black text-slate-700">🗺️</div>
                                    <div>
                                        <div className="text-sm font-bold text-slate-900">Learning Path</div>
                                        <div className="text-xs text-slate-600">Course Sequencing</div>
                                    </div>
                                </div>
                                <div className="rounded-xl ring-1 ring-slate-200 bg-white p-4 flex items-center gap-3">
                                    <div className="h-10 w-10 rounded-xl bg-slate-100 grid place-items-center font-black text-slate-700">❓</div>
                                    <div>
                                        <div className="text-sm font-bold text-slate-900">Quiz Builder</div>
                                        <div className="text-xs text-slate-600">Custom Quizzes</div>
                                    </div>
                                </div>
                            </div>

                            <div className="pt-4 flex justify-end">
                                <button className="rounded-xl bg-teal-700 text-white px-5 py-2.5 text-sm font-bold hover:bg-teal-800 transition">
                                    Browse All Plugins <span className="ml-1">›</span>
                                </button>
                            </div>
                        </Card>
                    </main>
                </div>
            </div>
        </div>
    );
}
