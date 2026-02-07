import { AdminShell } from "../components/AdminShell";
import { Card } from "../components/Card";

type PlaceholderViewProps = {
    title: string;
    activeKey: "dashboard" | "statements" | "users" | "reports" | "datasources" | "marketplace" | "settings";
};

export function PlaceholderView({ title, activeKey }: PlaceholderViewProps) {
    return (
        <AdminShell activeKey={activeKey}>
            <div className="text-2xl font-semibold text-slate-800">{title}</div>
            <Card>
                <div className="rounded-lg border border-dashed border-slate-300 bg-slate-50 px-6 py-10 text-center">
                    <div className="text-lg font-semibold text-slate-700">{title}</div>
                    <div className="mt-2 text-sm text-slate-500">This view is not implemented yet.</div>
                </div>
            </Card>
        </AdminShell>
    );
}
