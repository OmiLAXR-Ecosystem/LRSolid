import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Card } from "../../components/Card";
import type { WidgetPlugin } from "../types";

type Point = { day: string; count: number };

function dayLabel(iso: string) {
  const d = new Date(`${iso}T00:00:00Z`);
  return d.toLocaleDateString("de-DE", { weekday: "short" });
}

export const StatementTrendWidget: WidgetPlugin<Point[]> = {
  id: "statement_trend",
  name: "Statement Throughput",
  colSpan: "col-span-2",
  configSchema: [
    { key: "windowDays", label: "Window (days)", type: "int", default: 7, min: 3, max: 90 },
  ],
  component: ({ data, config }) => {
    const days = (config?.windowDays as number) ?? 7;
    const points = (data ?? []).map((p) => ({ day: dayLabel(p.day), count: p.count }));
    return (
      <Card title={`Statement Throughput (${days}d)`}>
        <div className="h-[260px]">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={points} margin={{ top: 10, right: 20, bottom: 0, left: 0 }}>
              <CartesianGrid strokeDasharray="4 4" />
              <XAxis dataKey="day" />
              <YAxis allowDecimals={false} />
              <Tooltip />
              <Line type="monotone" dataKey="count" stroke="#2CBCC7" strokeWidth={3} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </Card>
    );
  },
};
