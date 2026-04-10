import { useEffect, useState } from "react";
import "./App.css";
import { AuthProvider, useAuth } from "./contexts/AuthContext";
import { AdminShell, type NavKey } from "./components/layout/AdminShell";
import { LRSolidDashboard } from "./LRSolidDashboard";
import { StatementsView } from "./views/StatementsView";
import { UsersView } from "./views/UsersView";
import { LoginView } from "./views/LoginView";
import { PlaceholderView } from "./views/PlaceholderView";

type RouteConfig = {
    activeKey: NavKey;
    searchPlaceholder?: string;
    content: React.ReactNode;
};

const ROUTES: Record<string, RouteConfig> = {
    dashboard:   { activeKey: "dashboard",   searchPlaceholder: "Search...",            content: <LRSolidDashboard /> },
    statements:  { activeKey: "statements",  searchPlaceholder: "Search statements...", content: <StatementsView /> },
    users:       { activeKey: "users",       searchPlaceholder: "Search users...",      content: <UsersView /> },
    clients:     { activeKey: "clients",     searchPlaceholder: "Search clients...",    content: <PlaceholderView title="Clients" /> },
    reports:     { activeKey: "reports",                                                content: <PlaceholderView title="Reports" /> },
    datasources: { activeKey: "datasources",                                            content: <PlaceholderView title="Data Stores" /> },
    marketplace: { activeKey: "marketplace",                                            content: <PlaceholderView title="Plugin Marketplace" /> },
    settings:    { activeKey: "settings",                                               content: <PlaceholderView title="Settings" /> },
};

const DEFAULT_ROUTE = ROUTES.dashboard;

function normalizeRoute(hash: string): string {
    return hash.replace(/^#\/?/, "") || "dashboard";
}

function AppShell() {
    const { state } = useAuth();
    const [route, setRoute] = useState(() => normalizeRoute(window.location.hash));

    useEffect(() => {
        if (!window.location.hash) window.location.hash = "#/dashboard";
        const handler = () => setRoute(normalizeRoute(window.location.hash));
        window.addEventListener("hashchange", handler);
        return () => window.removeEventListener("hashchange", handler);
    }, []);

    // Loading: blank screen while checking session
    if (state.status === "loading") {
        return (
            <div className="min-h-screen bg-slate-100 flex items-center justify-center">
                <div className="text-sm text-slate-400">Loading...</div>
            </div>
        );
    }

    if (state.status === "unauthenticated") {
        return <LoginView />;
    }

    const { activeKey, searchPlaceholder, content } = ROUTES[route] ?? DEFAULT_ROUTE;

    return (
        <AdminShell activeKey={activeKey} searchPlaceholder={searchPlaceholder}>
            {content}
        </AdminShell>
    );
}

export default function App() {
    return (
        <AuthProvider>
            <AppShell />
        </AuthProvider>
    );
}
