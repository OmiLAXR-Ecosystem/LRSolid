import { Users } from "lucide-react";
import { Card } from "../../components/Card";
import type { WidgetPlugin } from "../types";

type Data = { value: number; total: number };

export const TrackedClientsWidget: WidgetPlugin<Data> = {
  id: "tracked_clients",
  name: "Tracked Clients",
  colSpan: "col-span-1",
  component: ({ data }) => (
    <Card className="p-5">
      <div className="flex items-start justify-between">
        <div>
          <div className="text-sm font-semibold text-slate-700">Tracked Clients</div>
          <div className="mt-2 text-3xl font-extrabold text-slate-900">
            {data?.value ?? 0}<span className="text-lg text-slate-400">/{data?.total ?? 0}</span>
          </div>
        </div>
        <div className="text-teal-500"><Users className="h-6 w-6" /></div>
      </div>
    </Card>
  ),
};
