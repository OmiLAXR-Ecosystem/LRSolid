import { useEffect, useMemo, useState } from "react";
import { Card } from "../components/ui/Card";
import { FieldLabel } from "../components/ui/FieldLabel";
import { TextInput } from "../components/ui/TextInput";
import { Modal } from "../components/ui/Modal";
import { ApiError } from "../api/client";
import { createProject, deleteProject, fetchProjects } from "../api/lrs";
import type { RestProject } from "../types/lrs";
import { useConfirm } from "../contexts/ConfirmContext";
import { useOrg } from "../contexts/OrgContext";

export function ProjectsView() {
  const { currentOrg } = useOrg();
  const [allItems, setAllItems] = useState<RestProject[]>([]);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reloadTick, setReloadTick] = useState(0);
  const confirm = useConfirm();

  const [createOpen, setCreateOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  useEffect(() => {
    if (!currentOrg) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    fetchProjects(currentOrg)
      .then((data) => { if (!cancelled) setAllItems(data.items ?? []); })
      .catch((err) => { if (!cancelled) setError(err instanceof Error ? err.message : "Request failed"); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [reloadTick, currentOrg]);

  const items = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return allItems;
    return allItems.filter(
      (p) =>
        p.name.toLowerCase().includes(q) ||
        p.slug.toLowerCase().includes(q) ||
        (p.description ?? "").toLowerCase().includes(q),
    );
  }, [allItems, query]);

  const resetForm = () => {
    setName("");
    setDescription("");
    setCreateError(null);
  };

  const handleCreate = async () => {
    if (!name.trim()) { setCreateError("Name is required"); return; }
    setSubmitting(true);
    setCreateError(null);
    try {
      await createProject(currentOrg, { name: name.trim(), description: description.trim() || undefined });
      setCreateOpen(false);
      setReloadTick((v) => v + 1);
    } catch (err) {
      setCreateError(err instanceof ApiError ? err.message : err instanceof Error ? err.message : "Create failed");
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (project: RestProject) => {
    const ok = await confirm({
      title: "Delete Project",
      message: `Delete "${project.name}"? This will also delete all associated LRS and clients.`,
      confirmLabel: "Delete",
      destructive: true,
    });
    if (!ok) return;
    try {
      await deleteProject(currentOrg, project.id);
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
            <TextInput value={query} onChange={setQuery} placeholder="Name, slug, description..." />
          </div>
          <button
            onClick={() => { resetForm(); setCreateOpen(true); }}
            className="h-10 rounded-lg bg-teal-600 px-4 text-sm font-semibold text-white hover:bg-teal-700 transition"
          >
            New Project
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
                <th className="px-4 py-3 text-left font-semibold">Slug</th>
                <th className="px-4 py-3 text-left font-semibold">Description</th>
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
                    {query ? "No projects match your search." : "No projects found."}
                  </td>
                </tr>
              )}
              {!loading && items.map((row) => (
                <tr key={row.id} className="bg-white">
                  <td className="px-4 py-3 font-medium text-slate-800">{row.name}</td>
                  <td className="px-4 py-3 font-mono text-xs text-slate-500">{row.slug}</td>
                  <td className="px-4 py-3 text-slate-600 max-w-xs truncate" title={row.description ?? ""}>
                    {row.description || "—"}
                  </td>
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
        title="Create Project"
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
              disabled={submitting}
              className="rounded-lg bg-teal-600 px-4 py-2 text-sm font-semibold text-white hover:bg-teal-700 disabled:opacity-60"
            >
              {submitting ? "Creating..." : "Create Project"}
            </button>
          </>
        }
      >
        <div className="space-y-4">
          {createError && <div className="text-sm text-red-600">{createError}</div>}
          <div>
            <FieldLabel>Name</FieldLabel>
            <TextInput value={name} onChange={setName} placeholder="e.g. My Project" />
          </div>
          <div>
            <FieldLabel>Description (optional)</FieldLabel>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What is this project used for?"
              className="w-full min-h-[80px] rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300 focus:outline-none"
            />
          </div>
        </div>
      </Modal>
    </div>
  );
}
