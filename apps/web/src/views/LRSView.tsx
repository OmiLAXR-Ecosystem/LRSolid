import { useEffect, useMemo, useState } from "react";
import { Card } from "../components/ui/Card";
import { FieldLabel } from "../components/ui/FieldLabel";
import { TextInput } from "../components/ui/TextInput";
import { SelectInput } from "../components/ui/SelectInput";
import { Modal } from "../components/ui/Modal";
import { ApiError } from "../api/client";
import { createLRS, deleteLRS, fetchDataStores, fetchLRSList, fetchProjects } from "../api/lrs";
import type { RestDataStore, RestLRS, RestProject } from "../types/lrs";
import { useConfirm } from "../contexts/ConfirmContext";
import { useOrg } from "../contexts/OrgContext";

export function LRSView() {
  const { currentOrg } = useOrg();
  const [allItems, setAllItems] = useState<RestLRS[]>([]);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reloadTick, setReloadTick] = useState(0);
  const confirm = useConfirm();

  const [createOpen, setCreateOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [formLoading, setFormLoading] = useState(false);
  const [name, setName] = useState("");
  const [projectId, setProjectId] = useState("");
  const [dataStoreId, setDataStoreId] = useState("");
  const [projects, setProjects] = useState<RestProject[]>([]);
  const [dataStores, setDataStores] = useState<RestDataStore[]>([]);

  useEffect(() => {
    if (!currentOrg) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    fetchLRSList(currentOrg)
      .then((data) => { if (!cancelled) setAllItems(data.items ?? []); })
      .catch((err) => { if (!cancelled) setError(err instanceof Error ? err.message : "Request failed"); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [reloadTick, currentOrg]);

  useEffect(() => {
    if (!createOpen || !currentOrg) return;
    let cancelled = false;
    setFormLoading(true);
    Promise.all([fetchProjects(currentOrg), fetchDataStores(currentOrg)])
      .then(([pRes, dsRes]) => {
        if (cancelled) return;
        setProjects(pRes.items ?? []);
        setDataStores(dsRes.items ?? []);
        if (!projectId && (pRes.items?.length ?? 0) > 0) setProjectId(pRes.items[0].id);
        if (!dataStoreId && (dsRes.items?.length ?? 0) > 0) setDataStoreId(dsRes.items[0].id);
      })
      .catch((err) => { if (!cancelled) setCreateError(err instanceof Error ? err.message : "Failed to load form data"); })
      .finally(() => { if (!cancelled) setFormLoading(false); });
    return () => { cancelled = true; };
  }, [createOpen, currentOrg]);

  const items = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return allItems;
    return allItems.filter(
      (l) =>
        l.name.toLowerCase().includes(q) ||
        l.id.toLowerCase().includes(q) ||
        l.dataStoreName.toLowerCase().includes(q),
    );
  }, [allItems, query]);

  const projectOptions = useMemo(() => projects.map((p) => ({ label: p.name, value: p.id })), [projects]);
  const dataStoreOptions = useMemo(() => dataStores.map((ds) => ({ label: `${ds.name} (${ds.databaseType})`, value: ds.id })), [dataStores]);

  const resetForm = () => {
    setName(""); setProjectId(""); setDataStoreId(""); setCreateError(null);
  };

  const handleCreate = async () => {
    if (!name.trim()) { setCreateError("Name is required"); return; }
    if (!projectId) { setCreateError("Project is required"); return; }
    if (!dataStoreId) { setCreateError("Data store is required"); return; }
    setSubmitting(true);
    setCreateError(null);
    try {
      await createLRS(currentOrg, { name: name.trim(), projectId, dataStoreId });
      setCreateOpen(false);
      setReloadTick((v) => v + 1);
    } catch (err) {
      setCreateError(err instanceof ApiError ? err.message : err instanceof Error ? err.message : "Create failed");
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (lrs: RestLRS) => {
    const ok = await confirm({
      title: "Delete LRS",
      message: `Delete "${lrs.name}"? This cannot be undone and will also delete all associated clients.`,
      confirmLabel: "Delete",
      destructive: true,
    });
    if (!ok) return;
    try {
      await deleteLRS(currentOrg, lrs.id);
      setReloadTick((v) => v + 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Delete failed");
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <div className="flex items-end gap-4">
          <div className="flex-1">
            <FieldLabel>Search</FieldLabel>
            <TextInput value={query} onChange={setQuery} placeholder="Name, ID, data store..." />
          </div>
          <button
            onClick={() => { resetForm(); setCreateOpen(true); }}
            className="h-10 rounded-lg bg-teal-600 px-4 text-sm font-semibold text-white hover:bg-teal-700 transition"
          >
            New LRS
          </button>
        </div>
      </Card>

      <Card>
        {error && <div className="mb-3 text-sm text-red-600">Error: {error}</div>}
        <div className="overflow-hidden rounded-lg ring-1 ring-slate-200">
          <table className="min-w-full text-sm">
            <thead className="bg-slate-50 text-slate-600">
              <tr>
                <th className="px-4 py-3 text-left font-semibold">Name</th>
                <th className="px-4 py-3 text-left font-semibold">ID</th>
                <th className="px-4 py-3 text-left font-semibold">Data Store</th>
                <th className="px-4 py-3 text-left font-semibold">Created</th>
                <th className="px-4 py-3 text-right font-semibold">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {loading && (
                <tr>
                  <td colSpan={5} className="px-4 py-6 text-center text-sm text-slate-500">Loading...</td>
                </tr>
              )}
              {!loading && items.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-6 text-center text-sm text-slate-500">
                    {query ? "No LRS match your search." : "No learning record stores found."}
                  </td>
                </tr>
              )}
              {!loading && items.map((row) => (
                <tr key={row.id} className="bg-white">
                  <td className="px-4 py-3 font-medium text-slate-800">{row.name}</td>
                  <td className="px-4 py-3 font-mono text-xs text-slate-500">{row.id}</td>
                  <td className="px-4 py-3 text-slate-600">{row.dataStoreName}</td>
                  <td className="px-4 py-3 text-slate-600">{new Date(row.createdAt).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-right">
                    <button
                      onClick={() => void handleDelete(row)}
                      className="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-semibold text-red-600 hover:bg-red-50"
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Create Learning Record Store"
        footer={
          <>
            <button
              onClick={() => setCreateOpen(false)}
              className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-50"
            >
              Cancel
            </button>
            <button
              onClick={() => void handleCreate()}
              disabled={submitting || formLoading}
              className="rounded-lg bg-teal-600 px-4 py-2 text-sm font-semibold text-white hover:bg-teal-700 disabled:opacity-60"
            >
              {submitting ? "Creating..." : "Create LRS"}
            </button>
          </>
        }
      >
        <div className="space-y-4">
          {createError && <div className="text-sm text-red-600">{createError}</div>}
          {formLoading && <div className="text-sm text-slate-500">Loading form data...</div>}
          {!formLoading && (
            <>
              <div>
                <FieldLabel>Name</FieldLabel>
                <TextInput value={name} onChange={setName} placeholder="e.g. Production LRS" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <FieldLabel>Project</FieldLabel>
                  {projects.length === 0
                    ? <div className="text-sm text-amber-600">No projects found. Create a project first.</div>
                    : <SelectInput value={projectId} onChange={setProjectId} options={projectOptions} />
                  }
                </div>
                <div>
                  <FieldLabel>Data Store</FieldLabel>
                  {dataStores.length === 0
                    ? <div className="text-sm text-amber-600">No data stores available.</div>
                    : <SelectInput value={dataStoreId} onChange={setDataStoreId} options={dataStoreOptions} />
                  }
                  <div className="mt-1 text-xs text-slate-400">The database backend where xAPI statements are stored.</div>
                </div>
              </div>
            </>
          )}
        </div>
      </Modal>
    </div>
  );
}
