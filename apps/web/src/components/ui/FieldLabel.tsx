import type { ReactNode } from "react";

export function FieldLabel({ children }: { children: ReactNode }) {
    return <div className="text-xs font-semibold text-slate-600 mb-1">{children}</div>;
}
