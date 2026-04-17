import type { WidgetPlugin } from "./types";

const widgets: WidgetPlugin[] = [];

export function registerWidget<T>(plugin: WidgetPlugin<T>) {
  widgets.push(plugin as WidgetPlugin);
}

export function getWidgets(): WidgetPlugin[] {
  return widgets;
}
