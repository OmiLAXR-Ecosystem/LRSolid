import { useEffect, useMemo, useState } from "react";
import { BarChart2, Calendar, Users } from "lucide-react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
} from "recharts";

import { Card } from "./components/Card";
import { fetchDashboard } from "./api/dashboard";
import { downloadStatementJson, fetchStatementById } from "./api/statements";
import { useOrg } from "./contexts/OrgContext";
import type { DashboardResponse } from "./types/dashboard";
import type { StatementDetail } from "./types/statement";
import { StatementDetailModal } from "./components/statements/StatementDetailModal";

function compact(n: number): string {
  return new Intl.NumberFormat("de-DE", { notation: "compact", maximumFractionDigits: 1 }).format(n);
}

function dayLabel(isoDate: string): string {
  const d = new Date(`${isoDate}T00:00:00Z`);
  return d.toLocaleDateString("de-DE", { weekday: "short" });
}

function relativeTime(iso: string): string {
  const date = new Date(iso);
  const deltaMs = Date.now() - date.getTime();
  const mins = Math.max(1, Math.floor(deltaMs / 60000));
  if (mins < 60) return `vor ${mins} min`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `vor ${hours} h`;
  const days = Math.floor(hours / 24);
  return `vor ${days} d`;
}

function shortVerb(verbId?: string | null): string {
  if (!verbId) return "-";
  const parts = verbId.split("/");
  return parts[parts.length - 1] || verbId;
}

function shortActor(actorName?: string | null, actorMbox?: string | null): string {
  if (actorName && actorName.trim() !== "") return actorName.trim();
  if (!actorMbox) return "anonymous";
  return actorMbox.replace(/^mailto:/i, "");
}

function shortObject(objectId?: string | null): string {
  if (!objectId) return "-";
  const parts = objectId.split("/");
  return parts[parts.length - 1] || objectId;
}

export function LRSolidDashboard() {
  const { currentOrg } = useOrg();
  const [data, setData] = useState<DashboardResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [detail, setDetail] = useState<StatementDetail | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);

  useEffect(() => {
    if (!currentOrg) return;
    setLoading(true);
    setError(null);
    fetchDashboard(currentOrg)
      .then(setData)
      .catch((err) => setError(err instanceof Error ? err.message : "Dashboard request failed"))
      .finally(() => setLoading(false));
  }, [currentOrg]);

  const activity = useMemo(
    () => (data?.activityDaily ?? []).map((d) => ({ day: dayLabel(d.day), count: d.count })),
    [data?.activityDaily],
  );

  const coverage = useMemo(() => {
    const k = data?.kpis;
    if (!k) return [];
    const toPct = (have: number, total: number) => (total <= 0 ? 0 : Math.round((have / total) * 100));
    return [
      { name: "Projects", value: toPct(k.projectsWithStatements, k.projectsCount) },
      { name: "LRS", value: toPct(k.lrsWithStatements, k.lrsCount) },
      { name: "Clients", value: toPct(k.clientsWithStatements, k.clientsCount) },
    ];
  }, [data?.kpis]);

  const kpis = data?.kpis;
  const cards = [
    { label: "Total Statements", value: compact(kpis?.totalStatements ?? 0), icon: <BarChart2 className="h-6 w-6" /> },
    { label: "Active Users (30d)", value: compact(kpis?.activeUsers ?? 0), icon: <Users className="h-6 w-6" /> },
    { label: "Statements Today", value: compact(kpis?.statementsToday ?? 0), icon: <Calendar className="h-6 w-6" /> },
    { label: "Tracked Clients", value: `${kpis?.clientsWithStatements ?? 0}/${kpis?.clientsCount ?? 0}`, icon: <Users className="h-6 w-6" /> },
  ];

  const openDetail = async (id: string) => {
    setDetailOpen(true);
    setDetailLoading(true);
    setDetailError(null);
    try {
      const d = await fetchStatementById(id);
      setDetail(d);
    } catch (err) {
      setDetailError(err instanceof Error ? err.message : "Failed to load statement detail");
    } finally {
      setDetailLoading(false);
    }
  };

  const closeDetail = () => {
    setDetailOpen(false);
    setDetail(null);
    setDetailError(null);
  };

  if (loading && !data) {
    return <div className="text-sm text-slate-500">Loading dashboard...</div>;
  }
  if (error && !data) {
    return <div className="text-sm text-red-600">{error}</div>;
  }

  return (
    <div className="space-y-4">
      {error && <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>}

      <div className="grid grid-cols-4 gap-4">
        {cards.map((k) => (
          <Card key={k.label} className="p-5">
            <div className="flex items-start justify-between">
              <div>
                <div className="text-sm font-semibold text-slate-700">{k.label}</div>
                <div className="mt-2 text-3xl font-extrabold text-slate-900">{k.value}</div>
              </div>
              <div className="text-teal-500">{k.icon}</div>
            </div>
          </Card>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-6">
        <Card title="Statement Throughput (7d)">
          <div className="h-[260px]">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={activity} margin={{ top: 10, right: 20, bottom: 0, left: 0 }}>
                <CartesianGrid strokeDasharray="4 4" />
                <XAxis dataKey="day" />
                <YAxis allowDecimals={false} />
                <Tooltip />
                <Line type="monotone" dataKey="count" stroke="#2CBCC7" strokeWidth={3} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </Card>

        <Card title="Coverage">
          <div className="grid grid-cols-[240px_1fr] gap-6 items-center">
            <div className="h-[220px]">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie data={coverage} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={45} outerRadius={82} paddingAngle={2}>
                    {coverage.map((_, i) => (
                      <Cell key={i} fill={["#2F5DA8", "#2CBCC7", "#F26B5E"][i % 3]} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </div>
            <div className="space-y-3 text-sm">
              {coverage.map((item, i) => (
                <div key={item.name} className="flex items-center justify-between text-slate-700">
                  <div className="flex items-center gap-2">
                    <span className="h-3 w-3 rounded-full" style={{ backgroundColor: ["#2F5DA8", "#2CBCC7", "#F26B5E"][i % 3] }} />
                    <span className="font-semibold">{item.name}</span>
                  </div>
                  <span>{item.value}%</span>
                </div>
              ))}
              <div className="pt-2 text-xs text-slate-500">
                Projects: {kpis?.projectsWithStatements ?? 0}/{kpis?.projectsCount ?? 0} • LRS: {kpis?.lrsWithStatements ?? 0}/{kpis?.lrsCount ?? 0}
              </div>
            </div>
          </div>
        </Card>
      </div>

      <Card title="Recent Statements">
        <div className="space-y-4">
          {(data?.recentStatements ?? []).length === 0 && (
            <div className="text-sm text-slate-500">No statements yet.</div>
          )}
          {(data?.recentStatements ?? []).map((row) => (
            <button
              key={row.id}
              type="button"
              onClick={() => void openDetail(row.id)}
              className="w-full text-left flex items-center justify-between text-sm rounded-lg px-2 py-1 hover:bg-slate-50 transition"
            >
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
            </button>
          ))}
          <div className="text-right">
            <a href={`#/${currentOrg}/statements`} className="text-sm font-semibold text-teal-600 hover:text-teal-700">
              View All
            </a>
          </div>
        </div>
      </Card>

      <StatementDetailModal
        open={detailOpen}
        detail={detail}
        loading={detailLoading}
        error={detailError}
        canExport={true}
        canVoid={false}
        onClose={closeDetail}
        onExport={() => {
          if (detail) downloadStatementJson(detail);
        }}
        onVoid={() => {}}
      />
    </div>
  );
}
