import { type MouseEvent, type ReactNode } from "react";
import {
    LayoutDashboard, FileText, Users, BarChart2,
    Database, Store, KeyRound, Search, Bell,
    Calendar, ChevronDown, LogOut, FolderOpen, ShieldCheck,
    MonitorCog, Moon, Sun,
} from "lucide-react";
import Logo from "../../assets/LRSolid_Logo.png";
import { useAuth } from "../../contexts/AuthContext";
import { useOrg } from "../../contexts/OrgContext";
import { navigate } from "../../lib/navigation";

export type NavKey =
    | "dashboard"
    | "statements"
    | "users"
    | "clients"
    | "projects"
    | "reports"
    | "datasources"
    | "marketplace"
    | "settings"
    | "organizations"
    | "profile";

type AdminShellProps = {
    activeKey: NavKey;
    searchPlaceholder?: string;
    themePreference: "light" | "dark" | "system";
    onThemePreferenceChange: (theme: "light" | "dark" | "system") => void;
    children: ReactNode;
};

type NavItem = {
    key: NavKey;
    label: string;
    href: string;
    icon: ReactNode;
};

function buildNavItems(org: string): NavItem[] {
    return [
        { key: "dashboard",   label: "Dashboard",              href: `/${org}/dashboard`,   icon: <LayoutDashboard className="h-5 w-5" /> },
        { key: "statements",  label: "Statements",             href: `/${org}/statements`,  icon: <FileText className="h-5 w-5" /> },
        { key: "users",       label: "Users",                  href: `/${org}/users`,       icon: <Users className="h-5 w-5" /> },
        { key: "clients",     label: "Clients",                href: `/${org}/clients`,     icon: <KeyRound className="h-5 w-5" /> },
        { key: "projects",    label: "Projects",               href: `/${org}/projects`,    icon: <FolderOpen className="h-5 w-5" /> },
        { key: "reports",     label: "Reports",                href: `/${org}/reports`,     icon: <BarChart2 className="h-5 w-5" /> },
        { key: "datasources", label: "Learning Record Stores", href: `/${org}/datasources`, icon: <Database className="h-5 w-5" /> },
    ];
}

function SidebarItem({ item, active }: { item: NavItem; active: boolean }) {
    const onClick = (e: MouseEvent<HTMLAnchorElement>) => {
        e.preventDefault();
        navigate(item.href);
    };

    return (
        <a
            href={item.href}
            onClick={onClick}
            aria-current={active ? "page" : undefined}
            className={[
                "flex items-center gap-3 rounded-lg px-4 py-3 text-sm font-medium transition",
                active ? "bg-white/90 text-slate-900 shadow-sm" : "text-white/90 hover:bg-white/10 hover:text-white",
            ].join(" ")}
        >
            <span className="h-5 w-5 grid place-items-center">{item.icon}</span>
            <span>{item.label}</span>
        </a>
    );
}

export function AdminShell({
    activeKey,
    searchPlaceholder,
    themePreference,
    onThemePreferenceChange,
    children,
}: AdminShellProps) {
    const { state, logout } = useAuth();
    const { orgs, currentOrg, setOrg } = useOrg();
    const user = state.status === "authenticated" ? state.user : null;

    const canSeeAllOrgs = user?.role === "admin" || user?.role === "owner";
    const navItems = buildNavItems(currentOrg);
    const marketplaceHref = `/${currentOrg}/marketplace`;
    const nextTheme = themePreference === "system" ? "light" : themePreference === "light" ? "dark" : "system";
    const themeIcon = themePreference === "system" ? <MonitorCog className="h-5 w-5 text-slate-600" /> : themePreference === "dark" ? <Moon className="h-5 w-5 text-slate-600" /> : <Sun className="h-5 w-5 text-slate-600" />;
    const themeTitle = `Theme: ${themePreference} (click to switch to ${nextTheme})`;

    return (
        <div className="h-screen max-h-screen overflow-hidden bg-slate-100 text-slate-900">
            <div className="h-full p-6">
                <div className="grid h-full min-h-0 grid-cols-[260px_1fr] gap-6">
                    <aside className="min-h-0 rounded-2xl bg-gradient-to-b from-teal-600 to-cyan-700 text-white shadow-sm overflow-hidden flex flex-col">
                        <div className="px-5 py-5 border-b border-white/15 flex items-center gap-3">
                            <img src={Logo} alt="LRSolid" className="w-full" />
                        </div>

                        <div className="px-4 pt-4">
                            <div className="text-[11px] font-semibold uppercase tracking-wide text-white/70">Organization</div>
                            <div className="relative mt-2">
                                <select
                                    value={currentOrg}
                                    onChange={(e) => setOrg(e.target.value)}
                                    className="w-full appearance-none rounded-xl bg-white/10 px-3 py-2 text-sm text-white ring-1 ring-white/20 focus:outline-none focus:ring-2 focus:ring-white/50"
                                >
                                    {orgs.map((o) => (
                                        <option key={o.slug} value={o.slug} className="text-slate-900">{o.name}</option>
                                    ))}
                                </select>
                                <ChevronDown className="h-4 w-4 text-white/70 absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none" />
                            </div>
                        </div>

                        <nav className="px-4 py-4 space-y-2 overflow-y-auto">
                            {navItems.map((item) => (
                                <SidebarItem key={item.key} item={item} active={activeKey === item.key} />
                            ))}
                        </nav>

                        <div className="mt-auto p-4">
                            <a
                                href={marketplaceHref}
                                onClick={(e) => {
                                    e.preventDefault();
                                    navigate(marketplaceHref);
                                }}
                                aria-current={activeKey === "marketplace" ? "page" : undefined}
                                className="w-full flex items-center gap-3 rounded-xl bg-white/10 px-4 py-3 text-sm font-semibold hover:bg-white/15 transition"
                            >
                                <Store className="h-5 w-5" />
                                Plugin Marketplace
                            </a>
                        </div>
                    </aside>

                    <main className="min-h-0 flex flex-col gap-6">
                        <div className="shrink-0 rounded-2xl bg-white shadow-sm ring-1 ring-slate-200 px-5 py-4 flex items-center justify-between">
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
                                    <Calendar className="h-5 w-5 text-slate-600" />
                                </button>
                                <button
                                    type="button"
                                    title={themeTitle}
                                    aria-label={themeTitle}
                                    onClick={() => onThemePreferenceChange(nextTheme)}
                                    className="h-10 min-w-10 rounded-xl ring-1 ring-slate-200 bg-white hover:bg-slate-50 px-2 grid place-items-center"
                                >
                                    {themeIcon}
                                </button>
                                {canSeeAllOrgs && (
                                    <a
                                        href={`/${currentOrg}/settings`}
                                        onClick={(e) => {
                                            e.preventDefault();
                                            navigate(`/${currentOrg}/settings`);
                                        }}
                                        title="Admin Panel"
                                        aria-current={activeKey === "settings" ? "page" : undefined}
                                        className="h-10 w-10 rounded-xl ring-1 ring-slate-200 bg-white hover:bg-slate-50 grid place-items-center"
                                    >
                                        <ShieldCheck className="h-5 w-5 text-slate-600" />
                                    </a>
                                )}
                                <div className="flex items-center gap-2 pl-2">
                                    <button
                                        type="button"
                                        onClick={() => navigate("/profile")}
                                        title="Profile"
                                        className="flex min-w-0 items-center gap-2 rounded-xl px-1.5 py-1 text-left transition hover:bg-slate-50"
                                    >
                                        <div className="h-9 w-9 rounded-full bg-teal-100 ring-1 ring-teal-200 grid place-items-center text-teal-700 text-sm font-semibold select-none">
                                            {user?.email?.[0]?.toUpperCase() ?? "?"}
                                        </div>
                                        <div className="hidden min-w-0 sm:block">
                                            <div className="max-w-[160px] truncate whitespace-nowrap text-xs font-semibold leading-tight text-slate-700">
                                                {user?.email ?? ""}
                                            </div>
                                            <div className="text-[10px] text-slate-400 capitalize">{user?.role ?? ""}</div>
                                        </div>
                                    </button>
                                    <button
                                        onClick={() => void logout()}
                                        title="Sign out"
                                        className="ml-1 h-8 w-8 rounded-lg ring-1 ring-slate-200 bg-white hover:bg-red-50 hover:ring-red-200 grid place-items-center transition"
                                    >
                                        <LogOut className="h-4 w-4 text-slate-500 hover:text-red-500" />
                                    </button>
                                </div>
                            </div>
                        </div>
                        <div className="min-h-0 flex-1 overflow-y-auto pr-1">
                            {children}
                        </div>
                        <footer className="shrink-0 pb-1 text-center text-xs text-slate-400">
                            © 2026{" "}
                            <a
                                href="https://goerzen.consulting"
                                target="_blank"
                                rel="noreferrer"
                                className="hover:text-slate-600 hover:underline"
                            >
                                Sergej Görzen
                            </a>
                            . All rights reserved.
                        </footer>
                    </main>
                </div>
            </div>
        </div>
    );
}
