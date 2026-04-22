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
import { SettingsView } from "./views/SettingsView";
import { LoginView } from "./views/LoginView";
import { PlaceholderView } from "./views/PlaceholderView";
import { PluginsView } from "./views/PluginsView";
import { ProfileView } from "./views/ProfileView";
import { listenNavigation, navigate } from "./lib/navigation";

type RouteConfig = {
    activeKey: NavKey;
    searchPlaceholder?: string;
    content: React.ReactNode;
};

type ThemePreference = "light" | "dark" | "system";
const THEME_STORAGE_KEY = "lrsolid-theme";

const ROUTES: Record<string, RouteConfig> = {
    dashboard: { activeKey: "dashboard", searchPlaceholder: "Search...", content: <LRSolidDashboard /> },
    statements: { activeKey: "statements", searchPlaceholder: "Search statements...", content: <StatementsView /> },
    users: { activeKey: "users", searchPlaceholder: "Search users...", content: <UsersView /> },
    clients: { activeKey: "clients", content: <ClientsView /> },
    projects: { activeKey: "projects", content: <ProjectsView /> },
    reports: { activeKey: "reports", content: <PlaceholderView title="Reports" /> },
    datasources: { activeKey: "datasources", content: <LRSView /> },
    marketplace: { activeKey: "marketplace", content: <PluginsView /> },
    settings: { activeKey: "settings", content: <SettingsView /> },
    organizations: { activeKey: "organizations", content: <OrganizationsView /> },
};

/** Parse `/:org/route` — returns org slug (or "") and route name. */
function parsePath(pathname: string): { org: string; route: string } {
    const cleaned = pathname.replace(/^\/+|\/+$/g, "");
    if (!cleaned) return { org: "", route: "dashboard" };
    const parts = cleaned.split("/");
    const org = parts[0] ?? "";
    const routeWithQuery = parts.slice(1).join("/") || "dashboard";
    const route = routeWithQuery.split("?")[0] || "dashboard";
    return { org, route };
}

function AppShell() {
    const { state } = useAuth();
    const [pathname, setPathname] = useState(() => window.location.pathname);
    const [themePreference, setThemePreference] = useState<ThemePreference>(() => {
        const stored = window.localStorage.getItem(THEME_STORAGE_KEY);
        return stored === "light" || stored === "dark" || stored === "system" ? stored : "system";
    });

    useEffect(() => {
        const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
        const root = document.documentElement;
        const applyTheme = () => {
            const shouldUseDark = themePreference === "dark" || (themePreference === "system" && mediaQuery.matches);
            root.classList.toggle("dark", shouldUseDark);
        };

        applyTheme();
        window.localStorage.setItem(THEME_STORAGE_KEY, themePreference);

        const onChange = () => {
            if (themePreference === "system") {
                applyTheme();
            }
        };

        mediaQuery.addEventListener("change", onChange);
        return () => mediaQuery.removeEventListener("change", onChange);
    }, [themePreference]);

    useEffect(() => {
        // Migrate old hash URLs like `#/default/statements` to `/default/statements`.
        if (window.location.hash.startsWith("#/")) {
            navigate(window.location.hash.slice(1), { replace: true });
        }

        const syncLocation = () => setPathname(window.location.pathname);
        return listenNavigation(syncLocation);
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

    if (pathname === "/profile") {
        return (
            <AdminShell
                activeKey="profile"
                themePreference={themePreference}
                onThemePreferenceChange={setThemePreference}
            >
                <ProfileView />
            </AdminShell>
        );
    }

    const { route } = parsePath(pathname);
    const { activeKey, searchPlaceholder, content } = ROUTES[route] ?? ROUTES.dashboard;

    return (
        <AdminShell
            activeKey={activeKey}
            searchPlaceholder={searchPlaceholder}
            themePreference={themePreference}
            onThemePreferenceChange={setThemePreference}
        >
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
