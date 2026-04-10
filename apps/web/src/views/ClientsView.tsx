import { useEffect, useMemo, useState } from "react";
import { Card } from "../components/ui/Card";
import { Pagination } from "../components/ui/Pagination";
import { FieldLabel } from "../components/ui/FieldLabel";
import { TextInput } from "../components/ui/TextInput";
import { SelectInput } from "../components/ui/SelectInput";
import { Modal } from "../components/ui/Modal";
import { ApiError } from "../api/client";
import { createClient, deleteClient, fetchClients, fetchLRS } from "../api/clients";
import { fetchUsers } from "../api/users";
import type { CreateClientResponse, RestClient, RestLRS } from "../types/client";
import type { RestUser } from "../types/user";

type ScopeState = {
  statementsRead: boolean;
  statementsWrite: boolean;
};

const emptyScopes: ScopeState = {
  statementsRead: true,
  statementsWrite: true,
};

function toLocalDateString(value?: string | null): string {
  if (!value) return "";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleDateString();
}

export function ClientsView() {
  const [query, setQuery] = useState("");
  const [limit, setLimit] = useState(25);
  const [page, setPage] = useState(0);
  const [items, setItems] = useState<RestClient[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reloadTick, setReloadTick] = useState(0);

  const [createOpen, setCreateOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [created, setCreated] = useState<CreateClientResponse | null>(null);

  const [name, setName] = useState("");
  const [lrsId, setLrsId] = useState("");
  const [note, setNote] = useState("");
  const [expiresAt, setExpiresAt] = useState("");
  const [scopes, setScopes] = useState<ScopeState>(emptyScopes);

  const [lrsItems, setLrsItems] = useState<RestLRS[]>([]);
  const [users, setUsers] = useState<RestUser[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [userSearch, setUserSearch] = useState("");
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([]);

  const fetchList = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchClients({
        limit,
        offset: page * limit,
        q: query.trim() || undefined,
      });
      setItems(data.items ?? []);
      setTotal(data.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void fetchList();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [limit, page, reloadTick]);

  useEffect(() => {
    setPage(0);
  }, [limit, query]);

  useEffect(() => {
    let cancelled = false;
    const run = async () => {
      if (!createOpen) return;
      setUsersLoading(true);
      try {
        const [lrsRes, usersRes] = await Promise.all([
          fetchLRS(),
          fetchUsers({ limit: 200, offset: 0, q: userSearch.trim() || undefined, role: "All" }),
        ]);
        if (cancelled) return;
        setLrsItems(lrsRes.items ?? []);
        setUsers(usersRes.items ?? []);
        if (!lrsId && (lrsRes.items?.length ?? 0) > 0) {
          setLrsId(lrsRes.items[0].id);
        }
      } catch (err) {
        if (!cancelled) {
          setCreateError(err instanceof Error ? err.message : "Failed to load form data");
        }
      } finally {
        if (!cancelled) setUsersLoading(false);
      }
    };
    void run();
    return () => {
      cancelled = true;
    };
  }, [createOpen, userSearch, lrsId]);

  const totalPages = Math.max(1, Math.ceil(total / Math.max(1, limit)));
  const pageStart = total === 0 ? 0 : page * limit + 1;
  const pageEnd = Math.min(total, (page + 1) * limit);

  const lrsOptions = useMemo(
    () => lrsItems.map((lrs) => ({ label: `${lrs.name} (${lrs.id})`, value: lrs.id })),
    [lrsItems],
  );

  const selectedUserSet = useMemo(() => new Set(selectedUserIds), [selectedUserIds]);

  const resetCreateForm = () => {
    setName("");
    setNote("");
    setExpiresAt("");
    setScopes(emptyScopes);
    setUserSearch("");
    setSelectedUserIds([]);
    setCreateError(null);
    setCreated(null);
  };

  const openCreateDialog = () => {
    resetCreateForm();
    setCreateOpen(true);
  };

  const toggleUser = (id: string) => {
    setSelectedUserIds((prev) => {
      if (prev.includes(id)) return prev.filter((x) => x !== id);
      return [...prev, id];
    });
  };

  const handleCreate = async () => {
    if (!name.trim()) {
      setCreateError("Label is required");
      return;
    }
    if (!lrsId) {
      setCreateError("Learning Record Store is required");
      return;
    }

    setSubmitting(true);
    setCreateError(null);
    try {
      const result = await createClient({
        name: name.trim(),
        lrsId,
        note: note.trim() || undefined,
        expiresAt: expiresAt || undefined,
        userIds: selectedUserIds,
        scopes: {
          "statements:read": scopes.statementsRead,
          "statements:write": scopes.statementsWrite,
        },
      });
      setCreated(result);
      setReloadTick((v) => v + 1);
    } catch (err) {
      if (err instanceof ApiError) {
        setCreateError(err.message);
      } else if (err instanceof Error) {
        setCreateError(err.message);
      } else {
        setCreateError("Create client failed");
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (client: RestClient) => {
    const ok = window.confirm(`Delete client \"${client.name}\"? This cannot be undone.`);
    if (!ok) return;
    try {
      await deleteClient(client.id);
      setReloadTick((v) => v + 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Delete failed");
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <div className="flex items-end justify-between gap-6">
          <div className="flex-1 max-w-md">
            <FieldLabel>Search</FieldLabel>
            <TextInput
              value={query}
              onChange={setQuery}
              placeholder="Name, key, note..."
            />
          </div>
          <button
            onClick={openCreateDialog}
            className="h-10 rounded-lg bg-teal-600 px-4 text-sm font-semibold text-white hover:bg-teal-700 transition"
          >
            New Client Key
          </button>
        </div>
      </Card>

      <Card>
        {error && <div className="mb-3 text-sm text-red-600">Error: {error}</div>}
        <div className="overflow-hidden rounded-lg ring-1 ring-slate-200">
          <table className="min-w-full text-sm">
            <thead className="bg-slate-50 text-slate-600">
              <tr>
                <th className="px-4 py-3 text-left font-semibold">Label</th>
                <th className="px-4 py-3 text-left font-semibold">Learning Record Store</th>
                <th className="px-4 py-3 text-left font-semibold">Key ID</th>
                <th className="px-4 py-3 text-left font-semibold">Users</th>
                <th className="px-4 py-3 text-left font-semibold">Expires</th>
                <th className="px-4 py-3 text-left font-semibold">Note</th>
                <th className="px-4 py-3 text-right font-semibold">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {loading && (
                <tr>
                  <td colSpan={7} className="px-4 py-6 text-center text-sm text-slate-500">Loading clients...</td>
                </tr>
              )}
              {!loading && items.length === 0 && (
                <tr>
                  <td colSpan={7} className="px-4 py-6 text-center text-sm text-slate-500">No clients found.</td>
                </tr>
              )}
              {!loading && items.map((row) => (
                <tr key={row.id} className="bg-white">
                  <td className="px-4 py-3 text-slate-700 font-medium">{row.name}</td>
                  <td className="px-4 py-3 text-slate-600">{row.lrsName ?? row.lrsId ?? "-"}</td>
                  <td className="px-4 py-3 text-slate-600 font-mono text-xs">{row.keyId}</td>
                  <td className="px-4 py-3 text-slate-600">{row.userIds.length}</td>
                  <td className="px-4 py-3 text-slate-600">{toLocalDateString(row.expiresAt) || "Never"}</td>
                  <td className="px-4 py-3 text-slate-600 max-w-[340px] truncate" title={row.note ?? ""}>{row.note || "-"}</td>
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

        <Pagination
          label={`Showing ${pageStart} to ${pageEnd} of ${total} entries`}
          limit={limit}
          onLimitChange={setLimit}
          onPrev={() => setPage((p) => Math.max(0, p - 1))}
          onNext={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
          prevDisabled={page === 0 || loading}
          nextDisabled={page >= totalPages - 1 || loading}
          currentPage={page + 1}
          totalPages={totalPages}
        />
      </Card>

      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Create Client Key"
        maxWidth="max-w-3xl"
        footer={
          <>
            <button
              onClick={() => setCreateOpen(false)}
              className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-50"
            >
              Close
            </button>
            <button
              onClick={() => void handleCreate()}
              disabled={submitting || !!created}
              className="rounded-lg bg-teal-600 px-4 py-2 text-sm font-semibold text-white hover:bg-teal-700 disabled:opacity-60"
            >
              {submitting ? "Creating..." : "Create Key"}
            </button>
          </>
        }
      >
        <div className="space-y-4">
          {createError && <div className="text-sm text-red-600">{createError}</div>}

          {created && (
            <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 space-y-1">
              <div className="font-semibold">Client credentials created. Secret is shown only once.</div>
              <div><span className="font-semibold">Key ID:</span> <span className="font-mono">{created.keyId}</span></div>
              <div><span className="font-semibold">Secret:</span> <span className="font-mono">{created.secret}</span></div>
            </div>
          )}

          <div className="grid grid-cols-2 gap-4">
            <div>
              <FieldLabel>Learning Record Store</FieldLabel>
              <SelectInput value={lrsId} onChange={setLrsId} options={lrsOptions} />
            </div>
            <div>
              <FieldLabel>Label</FieldLabel>
              <TextInput value={name} onChange={setName} placeholder="e.g. Moodle Ingest" />
            </div>
          </div>

          <div>
            <FieldLabel>Note</FieldLabel>
            <textarea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Longer internal note for this access key..."
              className="w-full min-h-[90px] rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300 focus:outline-none"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <FieldLabel>Expires At (optional)</FieldLabel>
              <input
                type="date"
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
                className="w-full rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300 focus:outline-none"
              />
            </div>
            <div>
              <FieldLabel>Scopes</FieldLabel>
              <div className="mt-1 space-y-2 rounded-lg border border-slate-200 p-3 text-sm text-slate-700">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={scopes.statementsRead}
                    onChange={(e) => setScopes((s) => ({ ...s, statementsRead: e.target.checked }))}
                  />
                  statements:read
                </label>
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={scopes.statementsWrite}
                    onChange={(e) => setScopes((s) => ({ ...s, statementsWrite: e.target.checked }))}
                  />
                  statements:write
                </label>
              </div>
            </div>
          </div>

          <div>
            <FieldLabel>Restrict To Users (optional)</FieldLabel>
            <div className="rounded-lg border border-slate-200 p-3">
              <div className="mb-2">
                <TextInput value={userSearch} onChange={setUserSearch} placeholder="Search users..." />
              </div>
              <div className="max-h-48 overflow-auto rounded-md border border-slate-100">
                {usersLoading && <div className="px-3 py-2 text-sm text-slate-500">Loading users...</div>}
                {!usersLoading && users.length === 0 && <div className="px-3 py-2 text-sm text-slate-500">No users found.</div>}
                {!usersLoading && users.map((user) => (
                  <label key={user.id} className="flex items-center gap-2 px-3 py-2 text-sm hover:bg-slate-50">
                    <input
                      type="checkbox"
                      checked={selectedUserSet.has(user.id)}
                      onChange={() => toggleUser(user.id)}
                    />
                    <span className="text-slate-700">{user.displayName || user.email}</span>
                    <span className="text-xs text-slate-500">{user.email}</span>
                  </label>
                ))}
              </div>
              <div className="mt-2 text-xs text-slate-500">Selected: {selectedUserIds.length}</div>
            </div>
          </div>
        </div>
      </Modal>
    </div>
  );
}
