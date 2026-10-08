import { useEffect, useState, useCallback } from "react";
import { Settings } from "lucide-react";
import { apiFetch } from "./api/client";
import { fetchDashboardConfig, saveDashboardConfig, resetDashboardConfig } from "./api/dashboardConfig";
import { useOrg } from "./contexts/OrgContext";
import { getWidgets } from "./plugins/registry";
import { registerBuiltinWidgets } from "./plugins/builtin";
import { DashboardConfigModal } from "./components/dashboard/DashboardConfigModal";
import type { WidgetEntry, WidgetPlugin } from "./plugins/types";

function responsiveColSpan(colSpan: WidgetPlugin["colSpan"]): string {
  if (colSpan === "col-span-4") return "col-span-1 md:col-span-2 xl:col-span-4";
  if (colSpan === "col-span-2") return "col-span-1 md:col-span-2";
  return "col-span-1";
}

registerBuiltinWidgets();

type WidgetMap = Record<string, unknown>;

export function LRSolidDashboard() {
  const { currentOrg } = useOrg();

  const [widgetData, setWidgetData] = useState<WidgetMap>({});
  const [layout, setLayout] = useState<WidgetEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [configOpen, setConfigOpen] = useState(false);

  const plugins = getWidgets();
  const pluginMap = Object.fromEntries(plugins.map((p) => [p.id, p]));

  const loadDashboard = useCallback(() => {
    if (!currentOrg) return;
    setLoading(true);
    setError(null);
    apiFetch<{ widgets: WidgetMap; layout: WidgetEntry[] }>(
      `/api/v1/${encodeURIComponent(currentOrg)}/dashboard`
    )
      .then((r) => {
        setWidgetData(r.widgets);
        setLayout(r.layout ?? []);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Dashboard request failed"))
      .finally(() => setLoading(false));
  }, [currentOrg]);

  useEffect(() => { loadDashboard(); }, [loadDashboard]);

  // Load saved config for the modal (may differ from current layout scope).
  const [configEntries, setConfigEntries] = useState<WidgetEntry[]>([]);
  useEffect(() => {
    if (!currentOrg) return;
    fetchDashboardConfig(currentOrg)
      .then((r) => setConfigEntries(r.widgets))
      .catch(() => {/* ignore, modal will show defaults */});
  }, [currentOrg]);

  async function handleSave(entries: WidgetEntry[], scope: "user" | "org") {
    if (!currentOrg) return;
    await saveDashboardConfig(currentOrg, entries, scope);
    setConfigEntries(entries);
    loadDashboard();
  }

  async function handleReset() {
    if (!currentOrg) return;
    await resetDashboardConfig(currentOrg);
    setConfigEntries([]);
    loadDashboard();
  }

  if (loading && Object.keys(widgetData).length === 0) {
    return <div className="text-sm text-slate-500">Loading dashboard...</div>;
  }
  if (error && Object.keys(widgetData).length === 0) {
    return <div className="text-sm text-red-600">{error}</div>;
  }

  // Use server-returned layout order; fall back to all plugins in registration order.
  const activeEntries: WidgetEntry[] = layout.length > 0
    ? layout.filter((e) => e.enabled && pluginMap[e.id])
    : plugins.map((p, i) => ({ id: p.id, enabled: true, order: i, config: {} }));

  const kpiEntries = activeEntries.filter((e) => pluginMap[e.id]?.colSpan === "col-span-1");
  const wideEntries = activeEntries.filter((e) => pluginMap[e.id]?.colSpan !== "col-span-1");

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        {error && (
          <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>
        )}
        <div className="w-full sm:ml-auto sm:w-auto">
          <button
            type="button"
            onClick={() => setConfigOpen(true)}
            className="flex w-full items-center justify-center gap-1.5 rounded-lg border border-slate-200 px-3 py-1.5 text-sm text-slate-600 hover:bg-slate-50 sm:w-auto sm:justify-start"
          >
            <Settings className="h-4 w-4" />
            Dashboard konfigurieren
          </button>
        </div>
      </div>

      {kpiEntries.length > 0 && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {kpiEntries.map((entry) => {
            const plugin = pluginMap[entry.id];
            if (!plugin) return null;
            return (
              <plugin.component
                key={entry.id}
                data={widgetData[entry.id]}
                config={entry.config ?? {}}
                orgSlug={currentOrg ?? ""}
              />
            );
          })}
        </div>
      )}

      {wideEntries.length > 0 && (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4 xl:gap-6">
          {wideEntries.map((entry) => {
            const plugin = pluginMap[entry.id];
            if (!plugin) return null;
            return (
              <div key={entry.id} className={responsiveColSpan(plugin.colSpan)}>
                <plugin.component
                  data={widgetData[entry.id]}
                  config={entry.config ?? {}}
                  orgSlug={currentOrg ?? ""}
                />
              </div>
            );
          })}
        </div>
      )}

      <DashboardConfigModal
        open={configOpen}
        onClose={() => setConfigOpen(false)}
        plugins={plugins}
        entries={configEntries.length > 0 ? configEntries : activeEntries}
        onSave={handleSave}
        onReset={handleReset}
      />
    </div>
  );
}
