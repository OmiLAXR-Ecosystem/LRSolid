import { useEffect, useState } from "react";
import { Package, Code2 } from "lucide-react";
import { Card } from "../components/ui/Card";
import { useOrg } from "../contexts/OrgContext";
import { apiFetch } from "../api/client";

type PluginInfo = { id: string; schema: { key: string; label: string; type: string; default: unknown }[] };

const EXAMPLE = `// plugins/my_widget.js
function schema() {
  return [
    { key: "limit", label: "Top N", type: "int", default: 5 }
  ];
}

function fetchData(orgID, config) {
  var rows = db.query(
    "SELECT COUNT(*) AS n FROM statements st " +
    "JOIN lrs l ON l.id = st.tenant_id " +
    "JOIN projects p ON p.id = l.project_id " +
    "WHERE p.org_id = $1",
    orgID
  );
  return { value: Number(rows[0].n) };
}`;

export function PluginsView() {
  const { currentOrg } = useOrg();
  const [items, setItems] = useState<PluginInfo[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!currentOrg) return;
    setLoading(true);
    apiFetch<{ items: PluginInfo[] }>(`/api/v1/${encodeURIComponent(currentOrg)}/plugins`)
      .then((r) => setItems(r.items))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [currentOrg]);

  return (
    <div className="space-y-6">
      <Card>
        <div className="flex items-center gap-3 mb-4">
          <Package className="h-5 w-5 text-teal-600" />
          <h2 className="font-semibold text-slate-800">Geladene Plugins</h2>
          <span className="ml-auto text-xs text-slate-400">{items.length} geladen</span>
        </div>

        {loading && <p className="text-sm text-slate-400">Laden…</p>}
        {!loading && items.length === 0 && (
          <p className="text-sm text-slate-400 py-2">
            Keine Plugins geladen. Lege <code className="font-mono bg-slate-100 px-1 rounded">.js</code>-Dateien
            in den <code className="font-mono bg-slate-100 px-1 rounded">plugins/</code>-Ordner und starte die
            Binary neu.
          </p>
        )}

        <div className="space-y-2">
          {items.map((p) => (
            <div key={p.id} className="flex items-start gap-4 rounded-lg border border-slate-200 bg-white px-4 py-3">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-mono text-sm font-semibold text-slate-800">{p.id}</span>
                  {p.schema?.length > 0 && (
                    <span className="text-xs text-slate-400">{p.schema.length} Setting{p.schema.length !== 1 ? "s" : ""}</span>
                  )}
                </div>
                {p.schema?.length > 0 && (
                  <div className="mt-1 flex flex-wrap gap-2">
                    {p.schema.map((f) => (
                      <span key={f.key} className="text-xs bg-slate-50 border border-slate-200 rounded px-2 py-0.5 font-mono">
                        {f.key}: {f.type} = {String(f.default)}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      </Card>

      <Card>
        <div className="flex items-center gap-3 mb-4">
          <Code2 className="h-5 w-5 text-teal-600" />
          <h2 className="font-semibold text-slate-800">Plugin schreiben</h2>
        </div>
        <p className="text-sm text-slate-600 mb-3">
          Lege eine <code className="font-mono bg-slate-100 px-1 rounded">.js</code>-Datei
          im Ordner <code className="font-mono bg-slate-100 px-1 rounded">plugins/</code> ab und starte die Binary neu.
          Der Dateiname wird zur Widget-ID.
        </p>
        <pre className="rounded-lg bg-slate-900 text-slate-100 text-xs p-4 overflow-x-auto leading-relaxed">{EXAMPLE}</pre>
        <div className="mt-4 space-y-2 text-sm text-slate-600">
          <p className="font-semibold text-slate-700">Verfügbare Host-APIs:</p>
          <div className="rounded-lg border border-slate-200 overflow-hidden text-xs font-mono">
            <div className="flex border-b border-slate-100 bg-slate-50">
              <span className="px-3 py-2 w-64 font-semibold text-slate-700">API</span>
              <span className="px-3 py-2 text-slate-500">Beschreibung</span>
            </div>
            <div className="flex border-b border-slate-100">
              <span className="px-3 py-2 w-64 text-teal-700">db.query(sql, ...args)</span>
              <span className="px-3 py-2 text-slate-500 font-sans">SELECT-Abfragen · gibt Array von Objekten zurück</span>
            </div>
          </div>
          <p className="text-xs text-slate-400 pt-1">
            Die UI-Komponente für das Widget wird separat in
            <code className="mx-1 bg-slate-100 px-1 rounded">apps/web/src/plugins/builtin/</code>
            als TSX-Datei registriert. Siehe <code className="bg-slate-100 px-1 rounded">plugins/README.md</code>.
          </p>
        </div>
      </Card>
    </div>
  );
}
