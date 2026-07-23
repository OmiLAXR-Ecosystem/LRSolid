import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { fetchOrgs } from "../api/org";
import type { RestOrg } from "../types/org";
import { useAuth } from "./AuthContext";
import { navigate } from "../lib/navigation";

type OrgContextType = {
  orgs: RestOrg[];
  currentOrg: string;
  setOrg: (slug: string) => void;
  loading: boolean;
};

const OrgContext = createContext<OrgContextType | null>(null);

function getOrgFromPath(): string {
  const cleaned = window.location.pathname.replace(/^\/+|\/+$/g, "");
  if (!cleaned || cleaned === "profile") return "";
  const [first] = cleaned.split("/");
  return first ?? "";
}

function getRouteFromPath(): string {
  const cleaned = window.location.pathname.replace(/^\/+|\/+$/g, "");
  if (!cleaned || cleaned === "profile") return "dashboard";
  const parts = cleaned.split("/");
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

        const orgFromPath = getOrgFromPath();
        const isValid = items.some((o) => o.slug === orgFromPath);

        if (orgFromPath && isValid) {
          setCurrentOrg(orgFromPath);
        } else if (items.length > 0) {
          const first = items[0].slug;
          setCurrentOrg(first);
          if (window.location.pathname !== "/profile") {
            const route = getRouteFromPath();
            navigate(`/${first}/${route}${window.location.search}`, { replace: true });
          }
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [state.status]);

  const setOrg = useCallback((slug: string) => {
    setCurrentOrg(slug);
    if (window.location.pathname === "/profile") return;
    const route = getRouteFromPath();
    navigate(`/${slug}/${route}${window.location.search}`);
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
