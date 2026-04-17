/** Widget data map returned by GET /api/v1/{org}/dashboard */
export type DashboardWidgetMap = Record<string, unknown>;

export type DashboardResponse = {
  widgets: DashboardWidgetMap;
};
