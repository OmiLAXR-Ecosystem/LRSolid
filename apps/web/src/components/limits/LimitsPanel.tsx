import { useCallback, useEffect, useState } from "react";
import { ApiError } from "../../api/client";
import { deleteLimit, fetchUsage, saveLimit, type UsageLimit, type UsageMetric, type UsageSnapshot, type UsageSubject } from "../../api/limits";
import { Modal } from "../ui/Modal";

const METRICS: { id: UsageMetric; label: string; used: (u: UsageSnapshot["usage"]) => number; bytes?: boolean }[] = [
  { id: "statements_total", label: "Statements, total", used: (u) => u.statementsTotal },
  { id: "statements_per_hour", label: "Statements per hour", used: (u) => u.statementsThisHour },
  { id: "statements_per_day", label: "Statements per day", used: (u) => u.statementsToday },
  { id: "storage_bytes", label: "Storage (MB)", used: (u) => u.storageBytes, bytes: true },
];

function formatCount(n: number): string {
  return new Intl.NumberFormat().format(n);
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let v = n / 1024;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v >= 10 ? v.toFixed(0) : v.toFixed(1)} ${units[i]}`;
}

function toInput(limit: UsageLimit | undefined, bytes: boolean): string {
  if (!limit) return "";
  if (!bytes) return String(limit.maxValue);
  return String(Math.round((limit.maxValue / (1024 * 1024)) * 100) / 100);
}

export function LimitsEditor({ org, subjectType, subjectId }: { org: string; subjectType: UsageSubject; subjectId: string }) {
  const [snapshot, setSnapshot] = useState<UsageSnapshot | null>(null);
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [enabled, setEnabled] = useState<Record<string, boolean>>({});
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    fetchUsage(org, subjectType, subjectId)
      .then((data) => {
        setSnapshot(data);
        const nextDrafts: Record<string, string> = {};
        const nextEnabled: Record<string, boolean> = {};
        for (const metric of METRICS) {
          const rule = data.limits.find((l) => l.metric === metric.id);
          nextDrafts[metric.id] = toInput(rule, !!metric.bytes);
          nextEnabled[metric.id] = rule ? rule.enabled : true;
        }
        setDrafts(nextDrafts);
        setEnabled(nextEnabled);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load limits"))
      .finally(() => setLoading(false));
  }, [org, subjectType, subjectId]);

  useEffect(() => {
    load();
  }, [load]);

  const ruleFor = (metric: UsageMetric) => snapshot?.limits.find((l) => l.metric === metric);

  const save = async (metric: UsageMetric, bytes: boolean) => {
    const raw = (drafts[metric] ?? "").trim();
    const existing = ruleFor(metric);
    setBusy(metric);
    setError(null);
    try {
      if (raw === "") {
        if (existing) await deleteLimit(org, existing.id);
      } else {
        const n = Number(raw);
        if (!Number.isFinite(n) || n < 0) {
          setError("Limit must be a number ≥ 0");
          return;
        }
        const maxValue = bytes ? Math.round(n * 1024 * 1024) : Math.round(n);
        await saveLimit(org, {
          subjectType,
          subjectId,
          metric,
          maxValue,
          enabled: enabled[metric] !== false,
        });
      }
      load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Save failed");
    } finally {
      setBusy(null);
    }
  };

  const remove = async (metric: UsageMetric) => {
    const existing = ruleFor(metric);
    if (!existing) return;
    setBusy(metric);
    setError(null);
    try {
      await deleteLimit(org, existing.id);
      load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Remove failed");
    } finally {
      setBusy(null);
    }
  };

  if (loading && !snapshot) return <div className="text-sm text-slate-500">Loading limits…</div>;

  return (
    <div className="space-y-3">
      <p className="text-sm text-slate-500">
        A limit counts this level and everything under it. Lower levels can add stricter limits. Clear the value and save to remove a rule.
      </p>
      {error && <div className="text-sm text-red-600">{error}</div>}
      <div className="divide-y divide-slate-100 rounded-lg ring-1 ring-slate-200">
        {METRICS.map((metric) => {
          const rule = ruleFor(metric.id);
          const used = snapshot ? metric.used(snapshot.usage) : 0;
          const max = rule?.enabled ? rule.maxValue : null;
          const ratio = max && max > 0 ? Math.min(used / max, 1) : 0;
          return (
            <div key={metric.id} className="space-y-2 px-3 py-3">
              <div className="flex items-baseline justify-between gap-3">
                <div className="text-sm font-medium text-slate-800">{metric.label}</div>
                <div className="text-xs text-slate-500">
                  used {metric.bytes ? formatBytes(used) : formatCount(used)}
                  {rule ? ` / ${metric.bytes ? formatBytes(rule.maxValue) : formatCount(rule.maxValue)}` : ""}
                  {rule && !rule.enabled ? " (off)" : ""}
                </div>
              </div>
              {max !== null && (
                <div className="h-1.5 overflow-hidden rounded-full bg-slate-100">
                  <div
                    className={`h-full ${used > max ? "bg-red-500" : "bg-teal-500"}`}
                    style={{ width: `${ratio * 100}%` }}
                  />
                </div>
              )}
              <div className="flex flex-wrap items-center gap-2">
                <input
                  value={drafts[metric.id] ?? ""}
                  onChange={(e) => setDrafts((d) => ({ ...d, [metric.id]: e.target.value }))}
                  placeholder="No limit"
                  inputMode="decimal"
                  className="w-36 rounded-lg bg-white px-3 py-1.5 text-sm text-slate-700 ring-1 ring-slate-200 focus:outline-none focus:ring-2 focus:ring-teal-300"
                />
                <label className="flex items-center gap-1.5 text-xs text-slate-600">
                  <input
                    type="checkbox"
                    checked={enabled[metric.id] !== false}
                    onChange={(e) => setEnabled((s) => ({ ...s, [metric.id]: e.target.checked }))}
                  />
                  Enforce
                </label>
                <button
                  onClick={() => void save(metric.id, !!metric.bytes)}
                  disabled={busy === metric.id}
                  className="rounded-lg bg-teal-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-teal-700 disabled:opacity-60"
                >
                  {busy === metric.id ? "Saving…" : "Save"}
                </button>
                {rule && (
                  <button
                    onClick={() => void remove(metric.id)}
                    disabled={busy === metric.id}
                    className="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-50 disabled:opacity-60"
                  >
                    Remove
                  </button>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export function LimitsButton({
  org,
  subjectType,
  subjectId,
  subjectName,
}: {
  org: string;
  subjectType: UsageSubject;
  subjectId: string;
  subjectName: string;
}) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-50"
      >
        Limits
      </button>
      <Modal open={open} onClose={() => setOpen(false)} title={`Limits · ${subjectName}`} maxWidth="max-w-xl">
        {open && <LimitsEditor org={org} subjectType={subjectType} subjectId={subjectId} />}
      </Modal>
    </>
  );
}
