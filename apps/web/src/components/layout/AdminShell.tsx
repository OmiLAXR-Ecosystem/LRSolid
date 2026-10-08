import { useEffect, useState, type MouseEvent, type ReactNode } from "react";
import {
    LayoutDashboard, FileText, Users, BarChart2,
    Database, Puzzle, KeyRound, Search, Bell,
    Calendar, ChevronDown, LogOut, FolderOpen, ShieldCheck,
    MonitorCog, Moon, Sun, Menu, X,
} from "lucide-react";
import Logo from "../../assets/LRSolid_Logo.png";
import { useAuth } from "../../contexts/AuthContext";
import { useOrg } from "../../contexts/OrgContext";
import { navigate } from "../../lib/navigation";
import { APP_VERSION } from "../../version";

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

function SidebarItem({ item, active, onNavigate }: { item: NavItem; active: boolean; onNavigate?: () => void }) {
    const onClick = (e: MouseEvent<HTMLAnchorElement>) => {
        e.preventDefault();
        navigate(item.href);
        onNavigate?.();
    };

    return (
        <a
            href={item.href}
            onClick={onClick}
            aria-current={active ? "page" : undefined}
            className={[
                "flex items-center gap-3 rounded-lg px-4 py-3 text-sm font-medium transition",
                active ? "bg-white/90 text-slate-950 shadow-sm" : "text-white/90 hover:bg-white/10 hover:text-white",
            ].join(" ")}
        >
            <span className="h-5 w-5 grid place-items-center">{item.icon}</span>
            <span>{item.label}</span>
        </a>
    );
}

function SidebarPanel({
    currentOrg,
    orgs,
    setOrg,
    navItems,
    activeKey,
    marketplaceHref,
    onNavigate,
    headerAction,
}: {
    currentOrg: string;
    orgs: { slug: string; name: string }[];
    setOrg: (slug: string) => void;
    navItems: NavItem[];
    activeKey: NavKey;
    marketplaceHref: string;
    onNavigate?: () => void;
    headerAction?: ReactNode;
}) {
    return (
        <>
            <div className="flex items-center gap-3 border-b border-white/15 px-5 py-5">
                <img src={Logo} alt="LRSolid" className={headerAction ? "min-w-0 flex-1" : "w-full"} />
                {headerAction}
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
                    <ChevronDown className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-white/70" />
                </div>
            </div>

            <nav className="space-y-2 overflow-y-auto px-4 py-4">
                {navItems.map((item) => (
                    <SidebarItem key={item.key} item={item} active={activeKey === item.key} onNavigate={onNavigate} />
                ))}
            </nav>

            <div className="mt-auto p-4">
                <a
                    href={marketplaceHref}
                    onClick={(e) => {
                        e.preventDefault();
                        navigate(marketplaceHref);
                        onNavigate?.();
                    }}
                    aria-current={activeKey === "marketplace" ? "page" : undefined}
                    className="flex w-full items-center gap-3 rounded-xl bg-white/10 px-4 py-3 text-sm font-semibold transition hover:bg-white/15"
                >
                    <Puzzle className="h-5 w-5" />
                    Plugins & Extensions
                </a>
            </div>
        </>
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
    const [navOpen, setNavOpen] = useState(false);

    const canSeeAllOrgs = user?.role === "admin" || user?.role === "owner";
    const navItems = buildNavItems(currentOrg);
    const marketplaceHref = `/${currentOrg}/marketplace`;
    const nextTheme = themePreference === "system" ? "light" : themePreference === "light" ? "dark" : "system";
    const themeIcon = themePreference === "system" ? <MonitorCog className="h-5 w-5 text-slate-600" /> : themePreference === "dark" ? <Moon className="h-5 w-5 text-slate-600" /> : <Sun className="h-5 w-5 text-slate-600" />;
    const themeTitle = `Theme: ${themePreference} (click to switch to ${nextTheme})`;

    useEffect(() => {
        if (!navOpen) return;
        const onKey = (e: KeyboardEvent) => {
            if (e.key === "Escape") setNavOpen(false);
        };
        window.addEventListener("keydown", onKey);
        return () => window.removeEventListener("keydown", onKey);
    }, [navOpen]);

    const sidebarProps = {
        currentOrg,
        orgs,
        setOrg,
        navItems,
        activeKey,
        marketplaceHref,
    };

    return (
        <div className="h-screen max-h-screen overflow-hidden bg-slate-100 text-slate-900">
            <div className="h-full p-3 sm:p-4 lg:p-6">
                <div className="grid h-full min-h-0 grid-cols-1 gap-3 lg:grid-cols-[260px_minmax(0,1fr)] lg:gap-6">
                    <aside className="hidden min-h-0 flex-col overflow-hidden rounded-2xl bg-gradient-to-b from-teal-600 to-cyan-700 text-white shadow-sm lg:flex">
                        <SidebarPanel {...sidebarProps} />
                    </aside>

                    {navOpen && (
                        <div className="fixed inset-0 z-50 lg:hidden">
                            <button
                                type="button"
                                aria-label="Close navigation"
                                className="absolute inset-0 bg-slate-900/40"
                                onClick={() => setNavOpen(false)}
                            />
                            <aside className="relative flex h-full w-[min(18rem,88vw)] flex-col overflow-hidden bg-gradient-to-b from-teal-600 to-cyan-700 text-white shadow-xl">
                                <SidebarPanel
                                    {...sidebarProps}
                                    onNavigate={() => setNavOpen(false)}
                                    headerAction={
                                        <button
                                            type="button"
                                            aria-label="Close navigation"
                                            onClick={() => setNavOpen(false)}
                                            className="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-white/10 text-white hover:bg-white/20"
                                        >
                                            <X className="h-4 w-4" />
                                        </button>
                                    }
                                />
                            </aside>
                        </div>
                    )}

                    <main className="flex min-h-0 min-w-0 flex-col gap-3 sm:gap-6">
                        <div className="flex shrink-0 flex-wrap items-center gap-2 rounded-2xl bg-white px-3 py-3 shadow-sm ring-1 ring-slate-200 sm:gap-3 sm:px-5 sm:py-4">
                            <button
                                type="button"
                                aria-label="Open navigation"
                                onClick={() => setNavOpen(true)}
                                className="order-1 grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-white ring-1 ring-slate-200 hover:bg-slate-50 lg:hidden"
                            >
                                <Menu className="h-5 w-5 text-slate-600" />
                            </button>
                            {searchPlaceholder && (
                                <div className="relative order-3 min-w-0 basis-full sm:order-2 sm:flex-1 sm:basis-auto">
                                    <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                                    <input
                                        className="w-full rounded-xl bg-slate-50 px-10 py-2.5 text-sm outline-none ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300"
                                        placeholder={searchPlaceholder}
                                    />
                                </div>
                            )}
                            {!searchPlaceholder && <div className="hidden flex-1 sm:block" />}
                            <div className="order-2 ml-auto flex shrink-0 items-center gap-1.5 sm:order-3 sm:ml-0 sm:gap-3">
                                <button className="hidden h-10 w-10 place-items-center rounded-xl bg-white ring-1 ring-slate-200 hover:bg-slate-50 md:grid">
                                    <Bell className="h-5 w-5 text-slate-600" />
                                </button>
                                <button className="hidden h-10 w-10 place-items-center rounded-xl bg-white ring-1 ring-slate-200 hover:bg-slate-50 md:grid">
                                    <Calendar className="h-5 w-5 text-slate-600" />
                                </button>
                                <button
                                    type="button"
                                    title={themeTitle}
                                    aria-label={themeTitle}
                                    onClick={() => onThemePreferenceChange(nextTheme)}
                                    className="grid h-10 min-w-10 place-items-center rounded-xl bg-white px-2 ring-1 ring-slate-200 hover:bg-slate-50"
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
                                        className="grid h-10 w-10 place-items-center rounded-xl bg-white ring-1 ring-slate-200 hover:bg-slate-50"
                                    >
                                        <ShieldCheck className="h-5 w-5 text-slate-600" />
                                    </a>
                                )}
                                <div className="flex items-center gap-1 pl-1 sm:gap-2 sm:pl-2">
                                    <button
                                        type="button"
                                        onClick={() => navigate("/profile")}
                                        title="Profile"
                                        className="flex min-w-0 items-center gap-2 rounded-xl px-1.5 py-1 text-left transition hover:bg-slate-50"
                                    >
                                        <div className="grid h-9 w-9 select-none place-items-center rounded-full bg-teal-100 text-sm font-semibold text-teal-700 ring-1 ring-teal-200">
                                            {user?.email?.[0]?.toUpperCase() ?? "?"}
                                        </div>
                                        <div className="hidden min-w-0 xl:block">
                                            <div className="max-w-[160px] truncate whitespace-nowrap text-xs font-semibold leading-tight text-slate-700">
                                                {user?.email ?? ""}
                                            </div>
                                            <div className="text-[10px] capitalize text-slate-400">{user?.role ?? ""}</div>
                                        </div>
                                    </button>
                                    <button
                                        onClick={() => void logout()}
                                        title="Sign out"
                                        className="grid h-8 w-8 place-items-center rounded-lg bg-white ring-1 ring-slate-200 transition hover:bg-red-50 hover:ring-red-200"
                                    >
                                        <LogOut className="h-4 w-4 text-slate-500 hover:text-red-500" />
                                    </button>
                                </div>
                            </div>
                        </div>
                        <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden pr-1">
                            {children}
                        </div>
                        <footer className="shrink-0 space-y-0.5 px-1 pb-1 text-center text-[11px] leading-relaxed text-slate-400 sm:text-xs">
                            <div>
                                © 2026{" "}
                                <a
                                    href="https://goerzen.consulting"
                                    target="_blank"
                                    rel="noreferrer"
                                    className="hover:text-slate-600 hover:underline"
                                >
                                    Dr.-Ing. Sergej Görzen
                                </a>
                                . All rights reserved.
                                <span className="mx-1.5" aria-hidden="true">·</span>
                                <a
                                    href="https://github.com/SGoerzen"
                                    target="_blank"
                                    rel="noreferrer"
                                    className="hover:text-slate-600 hover:underline"
                                >
                                    GitHub
                                </a>
                                <span className="mx-1.5" aria-hidden="true">·</span>
                                v{APP_VERSION} (Beta)
                            </div>
                            <div>
                                In cooperation with the{" "}
                                <a
                                    href="https://elearn.rwth-aachen.de"
                                    target="_blank"
                                    rel="noreferrer"
                                    className="hover:text-slate-600 hover:underline"
                                >
                                    Learning Technologies Research Group @ RWTH Aachen University
                                </a>
                            </div>
                        </footer>
                    </main>
                </div>
            </div>
        </div>
    );
}
