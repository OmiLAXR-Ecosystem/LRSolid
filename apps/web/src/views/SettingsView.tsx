import { useEffect, useState } from "react";
import { Card } from "../components/Card";
import { FieldLabel } from "../components/ui/FieldLabel";
import { TextInput } from "../components/ui/TextInput";
import { useOrg } from "../contexts/OrgContext";
import { fetchExportSettings, updateExportSettings } from "../api/export";

function toGB(n: number): string {
  return String(Math.round((n / (1024 * 1024 * 1024)) * 100) / 100);
}

function fromGB(s: string): number | null {
  const v = Number(s);
  if (!Number.isFinite(v) || v < 0) return null;
  return Math.round(v * 1024 * 1024 * 1024);
}

export function SettingsView() {
  const { currentOrg } = useOrg();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [remoteThresholdGB, setRemoteThresholdGB] = useState("2");
  const [localMaxGB, setLocalMaxGB] = useState("2");

  useEffect(() => {
    if (!currentOrg) return;
    setLoading(true);
    setError(null);
    fetchExportSettings(currentOrg)
      .then((s) => {
        setRemoteThresholdGB(toGB(s.requireRemoteTargetAboveBytes));
        setLocalMaxGB(toGB(s.localMaxBytes));
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load settings"))
      .finally(() => setLoading(false));
  }, [currentOrg]);

  const save = async () => {
    if (!currentOrg) return;
    setSaving(true);
    setError(null);
    setMessage(null);
    const requireRemoteTargetAboveBytes = fromGB(remoteThresholdGB);
    const localMaxBytes = fromGB(localMaxGB);
    if (requireRemoteTargetAboveBytes === null || localMaxBytes === null) {
      setSaving(false);
      setError("Please enter valid non-negative GB values.");
      return;
    }
    try {
      await updateExportSettings(currentOrg, { requireRemoteTargetAboveBytes, localMaxBytes });
      setMessage("Export settings saved.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save settings");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <div className="space-y-4">
          <div>
            <div className="text-lg font-semibold text-slate-800">Export Settings</div>
            <div className="text-sm text-slate-500">Threshold policy for large statement exports.</div>
          </div>
          {loading && <div className="text-sm text-slate-500">Loading...</div>}
          {error && <div className="text-sm text-red-600">{error}</div>}
          {message && <div className="text-sm text-emerald-700">{message}</div>}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-1">
              <FieldLabel>Require Remote Target Above (GB)</FieldLabel>
              <TextInput value={remoteThresholdGB} onChange={setRemoteThresholdGB} type="number" />
            </div>
            <div className="space-y-1">
              <FieldLabel>Local Export Max (GB)</FieldLabel>
              <TextInput value={localMaxGB} onChange={setLocalMaxGB} type="number" />
            </div>
          </div>
          <div className="flex justify-end">
            <button
              onClick={save}
              disabled={saving || loading}
              className="rounded-lg bg-teal-600 px-4 py-2 text-sm font-semibold text-white hover:bg-teal-700 disabled:opacity-60"
            >
              {saving ? "Saving..." : "Save"}
            </button>
          </div>
        </div>
      </Card>
    </div>
  );
}
