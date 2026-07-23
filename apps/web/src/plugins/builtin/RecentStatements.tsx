import { Card } from "../../components/Card";
import { navigate } from "../../lib/navigation";
import type { WidgetPlugin } from "../types";

type Statement = {
  id: string;
  actorName?: string | null;
  actorMbox?: string | null;
  verbId?: string | null;
  objectId?: string | null;
  tenantId: string;
  lrsName?: string | null;
  projectName: string;
  clientName?: string | null;
  storedAt: string;
};

function shortVerb(v?: string | null) {
  if (!v) return "-";
  const parts = v.split("/");
  return parts[parts.length - 1] || v;
}
function shortActor(name?: string | null, mbox?: string | null) {
  if (name?.trim()) return name.trim();
  return mbox ? mbox.replace(/^mailto:/i, "") : "anonymous";
}
function shortObject(id?: string | null) {
  if (!id) return "-";
  const parts = id.split("/");
  return parts[parts.length - 1] || id;
}
function relativeTime(iso: string) {
  const mins = Math.max(1, Math.floor((Date.now() - new Date(iso).getTime()) / 60000));
  if (mins < 60) return `vor ${mins} min`;
  const h = Math.floor(mins / 60);
  return h < 24 ? `vor ${h} h` : `vor ${Math.floor(h / 24)} d`;
}

export const RecentStatementsWidget: WidgetPlugin<Statement[]> = {
  id: "recent_statements",
  name: "Recent Statements",
  colSpan: "col-span-4",
  configSchema: [
    { key: "limit", label: "Number of rows", type: "int", default: 5, min: 1, max: 25 },
  ],
  component: ({ data, orgSlug }) => {
    const rows = data ?? [];
    return (
      <Card title="Recent Statements">
        <div className="space-y-4">
          {rows.length === 0 && <div className="text-sm text-slate-500">No statements yet.</div>}
          {rows.map((row) => (
            <div key={row.id} className="flex items-center justify-between text-sm rounded-lg px-2 py-1">
              <div className="min-w-0">
                <div className="font-semibold text-slate-800 truncate">
                  {(row.clientName ?? row.lrsName ?? "Client")}: {shortActor(row.actorName, row.actorMbox)} {shortVerb(row.verbId)} {shortObject(row.objectId)}
                </div>
                <div className="text-xs text-slate-500 truncate">
                  {shortVerb(row.verbId)} • {row.projectName} • {row.lrsName ?? row.tenantId}
                  {row.clientName ? ` • ${row.clientName}` : ""}
                </div>
              </div>
              <div className="ml-4 shrink-0 text-slate-500">{relativeTime(row.storedAt)}</div>
            </div>
          ))}
          <div className="text-right">
            <a
              href={`/${orgSlug}/statements`}
              onClick={(e) => {
                e.preventDefault();
                navigate(`/${orgSlug}/statements`);
              }}
              className="text-sm font-semibold text-teal-600 hover:text-teal-700"
            >
              View All
            </a>
          </div>
        </div>
      </Card>
    );
  },
};
