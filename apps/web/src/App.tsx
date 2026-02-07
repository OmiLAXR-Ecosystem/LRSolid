import { useEffect, useState } from "react";
import "./App.css";
import { LRSolidDashboard } from "./LRSolidDashboard";
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

    switch (route) {
        case "dashboard":
            return <LRSolidDashboard />;
        case "statements":
            return <StatementsView />;
        case "users":
            return <UsersView />;
        case "reports":
            return <PlaceholderView title="Reports" activeKey="reports" />;
        case "datasources":
            return <PlaceholderView title="Datasources" activeKey="datasources" />;
        case "marketplace":
            return <PlaceholderView title="Plugin Marketplace" activeKey="marketplace" />;
        case "settings":
            return <PlaceholderView title="Settings" activeKey="settings" />;
        default:
            return <LRSolidDashboard />;
    }
}
