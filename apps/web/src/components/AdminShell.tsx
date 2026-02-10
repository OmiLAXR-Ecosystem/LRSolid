import type { ReactNode } from "react";
import {
    LayoutDashboard,
    FileText,
    Users,
    BarChart2,
    Database,
    Store,
    Plus,
    Search,
    Bell,
    Settings,
    Calendar,
    ChevronDown,
} from "lucide-react";

type NavKey = "dashboard" | "statements" | "users" | "reports" | "datasources" | "marketplace" | "settings";

type AdminShellProps = {
    activeKey: NavKey;
    searchPlaceholder?: string;
    children: ReactNode;
};

type NavItem = {
    key: NavKey;
    label: string;
    href: string;
    icon: ReactNode;
};

const navItems: NavItem[] = [
    { key: "dashboard", label: "Dashboard", href: "#/dashboard", icon: <LayoutDashboard className="h-5 w-5" /> },
    { key: "statements", label: "Statements", href: "#/statements", icon: <FileText className="h-5 w-5" /> },
    { key: "users", label: "Users", href: "#/users", icon: <Users className="h-5 w-5" /> },
    { key: "reports", label: "Reports", href: "#/reports", icon: <BarChart2 className="h-5 w-5" /> },
    { key: "datasources", label: "Datasources", href: "#/datasources", icon: <Database className="h-5 w-5" /> },
    { key: "marketplace", label: "Plugin Marketplace", href: "#/marketplace", icon: <Store className="h-5 w-5" /> },
];

function SidebarItem({ item, active }: { item: NavItem; active: boolean }) {
    return (
        <a
            href={item.href}
            className={[
                "flex items-center gap-3 rounded-lg px-4 py-3 text-sm font-medium transition",
                active
                    ? "bg-white/90 text-slate-900 shadow-sm"
                    : "text-white/90 hover:bg-white/10 hover:text-white",
            ].join(" ")}
            aria-current={active ? "page" : undefined}
        >
            <span className="h-5 w-5 grid place-items-center">{item.icon}</span>
            <span>{item.label}</span>
        </a>
    );
}

export function AdminShell({ activeKey, searchPlaceholder, children }: AdminShellProps) {
    return (
        <div className="min-h-screen bg-slate-100 text-slate-900">
            <div className="p-6">
                <div className="grid grid-cols-[260px_1fr] gap-6 min-h-[calc(100vh-3rem)]">
                    <aside className="rounded-2xl bg-gradient-to-b from-teal-600 to-cyan-700 text-white shadow-sm overflow-hidden flex flex-col">
                        <div className="px-5 py-5 border-b border-white/15 flex items-center gap-3">
                            <div className="h-9 w-9 rounded-xl bg-white/15 flex items-center justify-center">
                                <span className="text-lg font-black">L</span>
                            </div>
                            <div className="text-xl font-extrabold tracking-tight">LRSolid</div>
                        </div>

                        <nav className="px-4 py-4 space-y-2">
                            {navItems.map((item) => (
                                <SidebarItem key={item.key} item={item} active={activeKey === item.key} />
                            ))}
                        </nav>

                        <div className="mt-auto p-4">
                            <a
                                href="#/settings"
                                className="w-full flex items-center gap-3 rounded-xl bg-white/10 px-4 py-3 text-sm font-semibold hover:bg-white/15 transition"
                                aria-current={activeKey === "settings" ? "page" : undefined}
                            >
                                <Plus className="h-5 w-5" />
                                Settings
                            </a>
                        </div>
                    </aside>

                    <main className="space-y-6">
                        <div className="rounded-2xl bg-white shadow-sm ring-1 ring-slate-200 px-5 py-4 flex items-center justify-between">
                            <div className="flex items-center gap-3 w-[520px] max-w-full">
                                {searchPlaceholder && (
                                    <div className="relative w-full">
                                        <Search className="h-4 w-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
                                        <input
                                            className="w-full rounded-xl bg-slate-50 ring-1 ring-slate-200 px-10 py-2.5 text-sm outline-none focus:ring-2 focus:ring-teal-300"
                                            placeholder={searchPlaceholder}
                                        />
                                    </div>
                                )}
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

                        {children}
                    </main>
                </div>
            </div>
        </div>
    );
}
