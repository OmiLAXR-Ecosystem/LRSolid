import { Card } from "../components/Card";

type PlaceholderViewProps = {
    title: string;
};

export function PlaceholderView({ title }: PlaceholderViewProps) {
    return (
        <Card>
            <div className="rounded-lg border border-dashed border-slate-300 bg-slate-50 px-6 py-10 text-center">
                <div className="text-lg font-semibold text-slate-700">{title}</div>
                <div className="mt-2 text-sm text-slate-500">This view is not implemented yet.</div>
            </div>
        </Card>
    );
}
