import { useState } from "react";
import { GripVertical, Settings, Trash2, Plus } from "lucide-react";
import { Modal } from "../ui/Modal";
import type { WidgetPlugin, WidgetEntry, ConfigField } from "../../plugins/types";

interface Props {
  open: boolean;
  onClose: () => void;
  plugins: WidgetPlugin[];
  entries: WidgetEntry[];
  onSave: (entries: WidgetEntry[], scope: "user" | "org") => Promise<void>;
  onReset: () => Promise<void>;
}

function defaultConfig(plugin: WidgetPlugin): Record<string, unknown> {
  const cfg: Record<string, unknown> = {};
  for (const f of plugin.configSchema ?? []) {
    cfg[f.key] = f.default;
  }
  return cfg;
}

function ConfigFieldInput({ field, value, onChange }: {
  field: ConfigField;
  value: unknown;
  onChange: (v: unknown) => void;
}) {
  if (field.type === "int") {
    return (
      <div className="flex flex-wrap items-center gap-2">
        <label className="w-full text-xs text-slate-600 sm:w-32 sm:shrink-0">{field.label}</label>
        <input
          type="number"
          min={field.min}
          max={field.max}
          value={value as number ?? field.default as number}
          onChange={(e) => onChange(Number(e.target.value))}
          className="w-20 rounded border border-slate-300 px-2 py-1 text-sm focus:outline-none focus:ring-1 focus:ring-teal-400"
        />
        {field.min !== undefined && field.max !== undefined && (
          <span className="text-xs text-slate-400">{field.min}–{field.max}</span>
        )}
      </div>
    );
  }
  if (field.type === "select") {
    return (
      <div className="flex flex-wrap items-center gap-2">
        <label className="w-full text-xs text-slate-600 sm:w-32 sm:shrink-0">{field.label}</label>
        <select
          value={value as string}
          onChange={(e) => onChange(e.target.value)}
          className="rounded border border-slate-300 px-2 py-1 text-sm focus:outline-none focus:ring-1 focus:ring-teal-400"
        >
          {(field.options ?? []).map((o) => (
            <option key={String(o)} value={String(o)}>{String(o)}</option>
          ))}
        </select>
      </div>
    );
  }
  return null;
}

export function DashboardConfigModal({ open, onClose, plugins, entries, onSave, onReset }: Props) {
  const [local, setLocal] = useState<WidgetEntry[]>(() => entries);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [scope, setScope] = useState<"user" | "org">("user");
  const [saving, setSaving] = useState(false);

  // Re-sync when entries change (e.g. modal re-opened)
  const [lastEntries, setLastEntries] = useState(entries);
  if (entries !== lastEntries) {
    setLastEntries(entries);
    setLocal(entries);
  }

  const pluginMap = Object.fromEntries(plugins.map((p) => [p.id, p]));
  const enabledIds = new Set(local.filter((e) => e.enabled).map((e) => e.id));
  const allIds = new Set(plugins.map((p) => p.id));
  const disabledIds = [...allIds].filter((id) => !enabledIds.has(id));

  function setEnabled(id: string, enabled: boolean) {
    setLocal((prev) => {
      const exists = prev.find((e) => e.id === id);
      if (exists) {
        return prev.map((e) => e.id === id ? { ...e, enabled } : e);
      }
      const plugin = pluginMap[id];
      return [...prev, { id, enabled, order: prev.length, config: defaultConfig(plugin) }];
    });
  }

  function setConfig(id: string, key: string, value: unknown) {
    setLocal((prev) =>
      prev.map((e) => e.id === id ? { ...e, config: { ...e.config, [key]: value } } : e)
    );
  }

  function moveUp(idx: number) {
    if (idx === 0) return;
    setLocal((prev) => {
      const next = [...prev];
      [next[idx - 1], next[idx]] = [next[idx], next[idx - 1]];
      return next.map((e, i) => ({ ...e, order: i }));
    });
  }

  function moveDown(idx: number) {
    setLocal((prev) => {
      if (idx >= prev.length - 1) return prev;
      const next = [...prev];
      [next[idx], next[idx + 1]] = [next[idx + 1], next[idx]];
      return next.map((e, i) => ({ ...e, order: i }));
    });
  }

  async function handleSave() {
    setSaving(true);
    try {
      await onSave(local, scope);
      onClose();
    } finally {
      setSaving(false);
    }
  }

  async function handleReset() {
    setSaving(true);
    try {
      await onReset();
      onClose();
    } finally {
      setSaving(false);
    }
  }

  const enabledEntries = local.filter((e) => e.enabled);

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Dashboard konfigurieren"
      maxWidth="max-w-2xl"
      footer={
        <div className="flex w-full flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-xs text-slate-500">Speichern für:</span>
            <select
              value={scope}
              onChange={(e) => setScope(e.target.value as "user" | "org")}
              className="rounded border border-slate-300 px-2 py-1 text-sm"
            >
              <option value="user">Nur für mich</option>
              <option value="org">Gesamte Organisation</option>
            </select>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => void handleReset()}
              disabled={saving}
              className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm text-slate-600 hover:bg-slate-50 disabled:opacity-50"
            >
              Zurücksetzen
            </button>
            <button
              type="button"
              onClick={() => void handleSave()}
              disabled={saving}
              className="rounded-lg bg-teal-600 px-4 py-1.5 text-sm font-semibold text-white hover:bg-teal-700 disabled:opacity-50"
            >
              {saving ? "Speichern…" : "Speichern"}
            </button>
          </div>
        </div>
      }
    >
      <div className="space-y-5">
        {/* Active widgets */}
        <div>
          <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Aktive Widgets</div>
          <div className="space-y-2">
            {enabledEntries.map((entry, idx) => {
              const plugin = pluginMap[entry.id];
              if (!plugin) return null;
              const hasSchema = (plugin.configSchema?.length ?? 0) > 0;
              const isExpanded = expandedId === entry.id;

              return (
                <div key={entry.id} className="rounded-lg border border-slate-200 bg-white">
                  <div className="flex items-center gap-2 px-3 py-2">
                    <div className="flex flex-col gap-0.5">
                      <button type="button" onClick={() => moveUp(idx)} disabled={idx === 0}
                        className="text-slate-400 hover:text-slate-600 disabled:opacity-20 text-xs leading-none">▲</button>
                      <button type="button" onClick={() => moveDown(idx)} disabled={idx === enabledEntries.length - 1}
                        className="text-slate-400 hover:text-slate-600 disabled:opacity-20 text-xs leading-none">▼</button>
                    </div>
                    <GripVertical className="h-4 w-4 text-slate-300 shrink-0" />
                    <span className="flex-1 text-sm font-medium text-slate-800">{plugin.name}</span>
                    {hasSchema && (
                      <button type="button"
                        onClick={() => setExpandedId(isExpanded ? null : entry.id)}
                        className={`text-slate-400 hover:text-teal-600 ${isExpanded ? "text-teal-600" : ""}`}
                      >
                        <Settings className="h-4 w-4" />
                      </button>
                    )}
                    <button type="button" onClick={() => setEnabled(entry.id, false)}
                      className="text-slate-400 hover:text-red-500">
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>

                  {isExpanded && hasSchema && (
                    <div className="border-t border-slate-100 bg-slate-50 px-4 py-3 space-y-2 rounded-b-lg">
                      {plugin.configSchema!.map((field) => (
                        <ConfigFieldInput
                          key={field.key}
                          field={field}
                          value={entry.config?.[field.key] ?? field.default}
                          onChange={(v) => setConfig(entry.id, field.key, v)}
                        />
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
            {enabledEntries.length === 0 && (
              <div className="text-sm text-slate-400 py-2">Keine Widgets aktiv.</div>
            )}
          </div>
        </div>

        {/* Inactive widgets */}
        {disabledIds.length > 0 && (
          <div>
            <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Verfügbare Widgets</div>
            <div className="space-y-1">
              {disabledIds.map((id) => {
                const plugin = pluginMap[id];
                if (!plugin) return null;
                return (
                  <div key={id}
                    className="flex items-center gap-2 rounded-lg border border-dashed border-slate-200 px-3 py-2 text-slate-500">
                    <span className="flex-1 text-sm">{plugin.name}</span>
                    <button type="button" onClick={() => setEnabled(id, true)}
                      className="text-slate-400 hover:text-teal-600">
                      <Plus className="h-4 w-4" />
                    </button>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>
    </Modal>
  );
}
