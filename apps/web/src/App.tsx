import { useEffect, useState } from "react";
import "./App.css";
import { AuthProvider, useAuth } from "./contexts/AuthContext";
import { ConfirmProvider } from "./contexts/ConfirmContext";
import { OrgProvider } from "./contexts/OrgContext";
import { AdminShell, type NavKey } from "./components/layout/AdminShell";
import { LRSolidDashboard } from "./LRSolidDashboard";
import { StatementsView } from "./views/StatementsView";
import { UsersView } from "./views/UsersView";
import { ClientsView } from "./views/ClientsView";
import { LRSView } from "./views/LRSView";
import { ProjectsView } from "./views/ProjectsView";
import { OrganizationsView } from "./views/OrganizationsView";
import { LoginView } from "./views/LoginView";
import { PlaceholderView } from "./views/PlaceholderView";

type RouteConfig = {
    activeKey: NavKey;
    searchPlaceholder?: string;
    content: React.ReactNode;
};

const ROUTES: Record<string, RouteConfig> = {
    dashboard:     { activeKey: "dashboard",   searchPlaceholder: "Search...",            content: <LRSolidDashboard /> },
    statements:    { activeKey: "statements",  searchPlaceholder: "Search statements...", content: <StatementsView /> },
    users:         { activeKey: "users",       searchPlaceholder: "Search users...",      content: <UsersView /> },
    clients:       { activeKey: "clients",                                                content: <ClientsView /> },
    projects:      { activeKey: "projects",                                               content: <ProjectsView /> },
    reports:       { activeKey: "reports",                                                content: <PlaceholderView title="Reports" /> },
    datasources:   { activeKey: "datasources",                                            content: <LRSView /> },
    marketplace:   { activeKey: "marketplace",                                            content: <PlaceholderView title="Plugin Marketplace" /> },
    settings:      { activeKey: "settings",                                               content: <PlaceholderView title="Settings" /> },
    organizations: { activeKey: "organizations",                                          content: <OrganizationsView /> },
};

/** Parse `#/:org/route` — returns org slug (or "") and route name. */
function parseHash(hash: string): { org: string; route: string } {
    const cleaned = hash.replace(/^#\/?/, "");
    const parts = cleaned.split("/");
    const org = parts[0] ?? "";
    const route = parts.slice(1).join("/") || "dashboard";
    return { org, route };
}

function AppShell() {
    const { state } = useAuth();
    const [hash, setHash] = useState(() => window.location.hash);

    useEffect(() => {
        if (!window.location.hash) window.location.hash = "#/dashboard";
        const handler = () => setHash(window.location.hash);
        window.addEventListener("hashchange", handler);
        return () => window.removeEventListener("hashchange", handler);
    }, []);

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

    const { route } = parseHash(hash);
    const { activeKey, searchPlaceholder, content } = ROUTES[route] ?? ROUTES.dashboard;

    return (
        <AdminShell activeKey={activeKey} searchPlaceholder={searchPlaceholder}>
            {content}
        </AdminShell>
    );
}

export default function App() {
    return (
        <AuthProvider>
            <ConfirmProvider>
                <OrgProvider>
                    <AppShell />
                </OrgProvider>
            </ConfirmProvider>
        </AuthProvider>
    );
}
