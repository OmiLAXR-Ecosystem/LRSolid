import { useEffect, useState } from "react";
import "./App.css";
import { LRSolidDashboard } from "./LRSolidDashboard";
import { AdminShell } from "./components/AdminShell";
import { StatementsView } from "./views/StatementsView";
import { UsersView } from "./views/UsersView";
import { PlaceholderView } from "./views/PlaceholderView";

function normalizeRoute(hash: string) {
    const cleaned = hash.replace(/^#\/?/, "");
    return cleaned || "dashboard";
}

export default function App() {
    const [route, setRoute] = useState(() => normalizeRoute(window.location.hash));

    useEffect(() => {
        if (!window.location.hash) {
            window.location.hash = "#/dashboard";
        }
        const handler = () => setRoute(normalizeRoute(window.location.hash));
        window.addEventListener("hashchange", handler);
        return () => window.removeEventListener("hashchange", handler);
    }, []);

    const view = (() => {
        switch (route) {
            case "dashboard":
                return {
                    activeKey: "dashboard" as const,
                    searchPlaceholder: "Search...",
                    content: <LRSolidDashboard />,
                };
            case "statements":
                return {
                    activeKey: "statements" as const,
                    searchPlaceholder: "Search statements...",
                    content: <StatementsView />,
                };
            case "users":
                return {
                    activeKey: "users" as const,
                    searchPlaceholder: "Search users...",
                    content: <UsersView />,
                };
            case "clients":
                return {
                    activeKey: "clients" as const,
                    searchPlaceholder: "Search clients...",
                    content: <PlaceholderView title="Clients" />,
                };
            case "reports":
                return {
                    activeKey: "reports" as const,
                    searchPlaceholder: undefined,
                    content: <PlaceholderView title="Reports" />,
                };
            case "datasources":
                return {
                    activeKey: "datasources" as const,
                    searchPlaceholder: undefined,
                    content: <PlaceholderView title="Data Stores" />,
                };
            case "marketplace":
                return {
                    activeKey: "marketplace" as const,
                    searchPlaceholder: undefined,
                    content: <PlaceholderView title="Plugin Marketplace" />,
                };
            case "settings":
                return {
                    activeKey: "settings" as const,
                    searchPlaceholder: undefined,
                    content: <PlaceholderView title="Settings" />,
                };
            default:
                return {
                    activeKey: "dashboard" as const,
                    searchPlaceholder: "Search...",
                    content: <LRSolidDashboard />,
                };
        }
    })();

    return (
        <AdminShell searchPlaceholder={view.searchPlaceholder}>
            {view.content}
        </AdminShell>
    );
}
