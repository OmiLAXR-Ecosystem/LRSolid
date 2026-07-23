import type { ReactNode } from "react";

export type ConfigFieldType = "int" | "select";

export interface ConfigField {
  key: string;
  label: string;
  type: ConfigFieldType;
  default: unknown;
  options?: unknown[];
  min?: number;
  max?: number;
}

export interface WidgetPlugin<T = unknown> {
  id: string;
  name: string;
  /** Tailwind col-span class */
  colSpan: "col-span-1" | "col-span-2" | "col-span-4";
  /** Config fields this widget exposes. Empty/undefined = no settings. */
  configSchema?: ConfigField[];
  component: (props: WidgetProps<T>) => ReactNode;
}

export interface WidgetProps<T = unknown> {
  data: T;
  /** Per-widget config values (keys match configSchema). */
  config: Record<string, unknown>;
  orgSlug: string;
}

export interface WidgetEntry {
  id: string;
  enabled: boolean;
  order: number;
  config: Record<string, unknown>;
}
