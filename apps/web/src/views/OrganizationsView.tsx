import { Building2 } from "lucide-react";
import { Card } from "../components/ui/Card";
import { useOrg } from "../contexts/OrgContext";

export function OrganizationsView() {
  const { orgs, loading, setOrg } = useOrg();

  return (
    <Card>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-lg font-semibold text-slate-800">Organizations</h2>
      </div>

      {loading && (
        <div className="text-sm text-slate-400 py-8 text-center">Loading…</div>
      )}

      {!loading && orgs.length === 0 && (
        <div className="text-sm text-slate-400 py-8 text-center">No organizations found.</div>
      )}

      {!loading && orgs.length > 0 && (
        <div className="divide-y divide-slate-100">
          {orgs.map((org) => (
            <div
              key={org.id}
              className="flex items-center gap-4 py-4 px-2 hover:bg-slate-50 rounded-lg cursor-pointer transition"
              onClick={() => setOrg(org.slug)}
            >
              <div className="h-10 w-10 rounded-xl bg-teal-50 ring-1 ring-teal-200 grid place-items-center shrink-0">
                <Building2 className="h-5 w-5 text-teal-600" />
              </div>
              <div className="min-w-0">
                <div className="text-sm font-semibold text-slate-800 truncate">{org.name}</div>
                <div className="text-xs text-slate-400 font-mono truncate">{org.slug}</div>
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}
