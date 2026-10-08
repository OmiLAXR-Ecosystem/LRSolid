import { useCallback, useEffect, useState } from "react";
import { Card } from "../Card";
import { FieldLabel } from "../ui/FieldLabel";
import { TextInput } from "../ui/TextInput";
import { ApiError } from "../../api/client";
import {
  createDataStore,
  deleteDataStore,
  listDataStores,
  testDataStore,
  testDataStoreByID,
  updateDataStore,
} from "../../api/dataStores";
import type { RestDataStore } from "../../types/lrs";

const inputClass =
  "w-full rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300 focus:outline-none";

function statusLabel(ds: RestDataStore): string {
  switch (ds.status) {
    case "connected":
      return "Connected";
    case "control_plane":
      return "Primary database";
    case "error":
      return "Connection failed";
    case "inactive":
      return "Inactive";
    case "not_configured":
      return "Not configured";
    default:
      return "Unavailable";
  }
}

export function DataStoresSection({ org }: { org: string }) {
  const [items, setItems] = useState<RestDataStore[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [databaseType, setDatabaseType] = useState<"postgres" | "mongodb">("mongodb");
  const [connectionString, setConnectionString] = useState("");
  const [description, setDescription] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listDataStores(org, true);
      setItems(res.items ?? []);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load data stores.");
    } finally {
      setLoading(false);
    }
  }, [org]);

  useEffect(() => {
    void load();
  }, [load]);

  const test = async () => {
    setError(null);
    setMessage(null);
    setBusy(true);
    try {
      await testDataStore({ name: name.trim(), databaseType, connectionString: connectionString.trim() });
      setMessage("Connection succeeded.");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Connection test failed.");
    } finally {
      setBusy(false);
    }
  };

  const save = async () => {
    setError(null);
    setMessage(null);
    if (!name.trim() || !connectionString.trim()) {
      setError("Name and connection string are required.");
      return;
    }
    setBusy(true);
    try {
      await createDataStore({
        name: name.trim(),
        databaseType,
        connectionString: connectionString.trim(),
        description: description.trim() || undefined,
      });
      setName("");
      setConnectionString("");
      setDescription("");
      setMessage("Data store connected.");
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not save the data store.");
    } finally {
      setBusy(false);
    }
  };

  const toggle = async (ds: RestDataStore) => {
    setError(null);
    setMessage(null);
    try {
      await updateDataStore(ds.id, { isActive: ds.isActive === false });
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not update the data store.");
    }
  };

  const remove = async (ds: RestDataStore) => {
    setError(null);
    setMessage(null);
    try {
      await deleteDataStore(ds.id);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not delete the data store.");
    }
  };

  const retest = async (ds: RestDataStore) => {
    setError(null);
    setMessage(null);
    try {
      await testDataStoreByID(ds.id);
      setMessage(`${ds.name} is reachable.`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Connection test failed.");
    }
  };

  return (
    <Card>
      <div className="space-y-4">
        <div>
          <div className="text-lg font-semibold text-slate-800">Data stores</div>
          <div className="text-sm text-slate-500">
            Attach another PostgreSQL or MongoDB database the same way as any external store. Statements, state, and profiles go there.
            Exports and dashboard aggregates stay on the primary database.
          </div>
        </div>
        {loading && <div className="text-sm text-slate-500">Loading...</div>}
        {error && <div className="text-sm text-red-600">{error}</div>}
        {message && <div className="text-sm text-emerald-700">{message}</div>}

        <div className="overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead className="text-xs uppercase tracking-wide text-slate-500">
              <tr>
                <th className="px-2 py-2">Name</th>
                <th className="px-2 py-2">Type</th>
                <th className="px-2 py-2">Status</th>
                <th className="px-2 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {items.map((ds) => (
                <tr key={ds.id} className="border-t border-slate-100">
                  <td className="px-2 py-2 text-slate-800">
                    {ds.name}
                    {ds.isDefault ? <span className="ml-2 text-xs text-slate-400">default</span> : null}
                    {ds.statusError ? <div className="text-xs text-red-600">{ds.statusError}</div> : null}
                  </td>
                  <td className="px-2 py-2 text-slate-600">{ds.databaseType}</td>
                  <td className="px-2 py-2 text-slate-600">{statusLabel(ds)}</td>
                  <td className="px-2 py-2 text-right whitespace-nowrap">
                    {ds.status !== "control_plane" && (
                      <>
                        <button type="button" className="mr-3 text-teal-700 hover:underline" onClick={() => void retest(ds)}>
                          Test
                        </button>
                        <button type="button" className="mr-3 text-slate-600 hover:underline" onClick={() => void toggle(ds)}>
                          {ds.isActive === false ? "Activate" : "Deactivate"}
                        </button>
                        <button type="button" className="text-red-600 hover:underline" onClick={() => void remove(ds)}>
                          Delete
                        </button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
              {items.length === 0 && !loading && (
                <tr>
                  <td className="px-2 py-3 text-slate-500" colSpan={4}>
                    No data stores yet.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div className="space-y-1">
            <FieldLabel>Name</FieldLabel>
            <TextInput value={name} onChange={setName} placeholder="Campus MongoDB" />
          </div>
          <div className="space-y-1">
            <FieldLabel>Type</FieldLabel>
            <select className={inputClass} value={databaseType} onChange={(e) => setDatabaseType(e.target.value as "postgres" | "mongodb")}>
              <option value="mongodb">MongoDB</option>
              <option value="postgres">PostgreSQL</option>
            </select>
          </div>
          <div className="space-y-1 md:col-span-2">
            <FieldLabel>Connection string</FieldLabel>
            <TextInput
              value={connectionString}
              onChange={setConnectionString}
              type="password"
              placeholder={databaseType === "mongodb" ? "mongodb://user:pass@host:27017/lrsolid" : "postgres://user:pass@host:5432/lrsolid?sslmode=disable"}
            />
          </div>
          <div className="space-y-1 md:col-span-2">
            <FieldLabel>Description</FieldLabel>
            <TextInput value={description} onChange={setDescription} placeholder="Optional" />
          </div>
        </div>
        <div className="flex gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={() => void test()}
            className="rounded-lg bg-white px-4 py-2 text-sm font-semibold text-slate-700 ring-1 ring-slate-200 hover:bg-slate-50 disabled:opacity-60"
          >
            Test connection
          </button>
          <button
            type="button"
            disabled={busy}
            onClick={() => void save()}
            className="rounded-lg bg-teal-600 px-4 py-2 text-sm font-semibold text-white hover:bg-teal-700 disabled:opacity-60"
          >
            Connect
          </button>
        </div>
      </div>
    </Card>
  );
}
