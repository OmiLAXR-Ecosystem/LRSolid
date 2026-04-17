import { Users } from "lucide-react";
import { Card } from "../../components/Card";
import type { WidgetPlugin } from "../types";

type Data = { value: number; windowDays: number };

function compact(n: number) {
  return new Intl.NumberFormat("de-DE", { notation: "compact", maximumFractionDigits: 1 }).format(n);
}

export const ActiveUsersWidget: WidgetPlugin<Data> = {
  id: "active_users",
  name: "Active Users",
  colSpan: "col-span-1",
  configSchema: [
    { key: "windowDays", label: "Lookback (days)", type: "int", default: 30, min: 1, max: 365 },
  ],
  component: ({ data }) => (
    <Card className="p-5">
      <div className="flex items-start justify-between">
        <div>
          <div className="text-sm font-semibold text-slate-700">Active Users ({data?.windowDays ?? 30}d)</div>
          <div className="mt-2 text-3xl font-extrabold text-slate-900">{compact(data?.value ?? 0)}</div>
        </div>
        <div className="text-teal-500"><Users className="h-6 w-6" /></div>
      </div>
    </Card>
  ),
};
