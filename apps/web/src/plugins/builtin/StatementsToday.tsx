import { Calendar } from "lucide-react";
import { Card } from "../../components/Card";
import type { WidgetPlugin } from "../types";

type Data = { value: number };

function compact(n: number) {
  return new Intl.NumberFormat("de-DE", { notation: "compact", maximumFractionDigits: 1 }).format(n);
}

export const StatementsTodayWidget: WidgetPlugin<Data> = {
  id: "statements_today",
  name: "Statements Today",
  colSpan: "col-span-1",
  component: ({ data }) => (
    <Card className="p-5">
      <div className="flex items-start justify-between">
        <div>
          <div className="text-sm font-semibold text-slate-700">Statements Today</div>
          <div className="mt-2 text-3xl font-extrabold text-slate-900">{compact(data?.value ?? 0)}</div>
        </div>
        <div className="text-teal-500"><Calendar className="h-6 w-6" /></div>
      </div>
    </Card>
  ),
};
