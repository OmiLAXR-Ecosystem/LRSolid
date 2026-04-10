import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { fetchOrgs } from "../api/org";
import type { RestOrg } from "../types/org";
import { useAuth } from "./AuthContext";

export const ALL_ORGS = "_all";

type OrgContextType = {
  orgs: RestOrg[];
  currentOrg: string;
  setOrg: (slug: string) => void;
  loading: boolean;
};

const OrgContext = createContext<OrgContextType | null>(null);

function getOrgFromHash(): string {
  const hash = window.location.hash.replace(/^#\/?/, "");
  const [first] = hash.split("/");
  return first ?? "";
}

function getRouteFromHash(): string {
  const hash = window.location.hash.replace(/^#\/?/, "");
  const parts = hash.split("/");
  return parts.slice(1).join("/") || "dashboard";
}

export function OrgProvider({ children }: { children: ReactNode }) {
  const { state } = useAuth();
  const [orgs, setOrgs] = useState<RestOrg[]>([]);
  const [currentOrg, setCurrentOrg] = useState<string>("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (state.status !== "authenticated") {
      setOrgs([]);
      setCurrentOrg("");
      return;
    }
    setLoading(true);
    fetchOrgs()
      .then((res) => {
        const items = res.items ?? [];
        setOrgs(items);

        const orgFromHash = getOrgFromHash();
        const isValid =
          orgFromHash === ALL_ORGS ||
          items.some((o) => o.slug === orgFromHash);

        if (orgFromHash && isValid) {
          setCurrentOrg(orgFromHash);
        } else if (items.length > 0) {
          const first = items[0].slug;
          setCurrentOrg(first);
          const route = getRouteFromHash();
          window.location.hash = `#/${first}/${route}`;
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [state.status]);

  const setOrg = useCallback((slug: string) => {
    setCurrentOrg(slug);
    const route = slug === ALL_ORGS ? "organizations" : getRouteFromHash();
    window.location.hash = `#/${slug}/${route}`;
  }, []);

  return (
    <OrgContext.Provider value={{ orgs, currentOrg, setOrg, loading }}>
      {children}
    </OrgContext.Provider>
  );
}

export function useOrg(): OrgContextType {
  const ctx = useContext(OrgContext);
  if (!ctx) throw new Error("useOrg must be used inside OrgProvider");
  return ctx;
}
