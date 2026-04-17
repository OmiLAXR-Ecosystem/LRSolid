import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";
import { Card } from "../../components/Card";
import type { WidgetPlugin } from "../types";

type CoverageItem = { name: string; pct: number; have: number; total: number };

const COLORS = ["#2F5DA8", "#2CBCC7", "#F26B5E"];

export const CoverageWidget: WidgetPlugin<CoverageItem[]> = {
  id: "coverage",
  name: "Coverage",
  colSpan: "col-span-2",
  component: ({ data }) => {
    const items = data ?? [];
    const pieData = items.map((i) => ({ name: i.name, value: i.pct }));
    return (
      <Card title="Coverage">
        <div className="grid grid-cols-[240px_1fr] gap-6 items-center">
          <div className="h-[220px]">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={pieData} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={45} outerRadius={82} paddingAngle={2}>
                  {pieData.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
          <div className="space-y-3 text-sm">
            {items.map((item, i) => (
              <div key={item.name} className="flex items-center justify-between text-slate-700">
                <div className="flex items-center gap-2">
                  <span className="h-3 w-3 rounded-full" style={{ backgroundColor: COLORS[i % COLORS.length] }} />
                  <span className="font-semibold">{item.name}</span>
                </div>
                <span>{item.pct}%</span>
              </div>
            ))}
            <div className="pt-2 text-xs text-slate-500">
              {items.map((i) => `${i.name}: ${i.have}/${i.total}`).join(" • ")}
            </div>
          </div>
        </div>
      </Card>
    );
  },
};
