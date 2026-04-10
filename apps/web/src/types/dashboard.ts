export type DashboardKpis = {
  totalStatements: number;
  activeUsers: number;
  statementsToday: number;
  errorLogs: number;
  projectsCount: number;
  lrsCount: number;
  clientsCount: number;
  projectsWithStatements: number;
  lrsWithStatements: number;
  clientsWithStatements: number;
};

export type DashboardRecentStatement = {
  id: string;
  actorName?: string | null;
  actorMbox?: string | null;
  verbId?: string | null;
  objectId?: string | null;
  tenantId: string;
  lrsName?: string | null;
  projectName: string;
  clientId?: string | null;
  clientName?: string | null;
  storedAt: string;
  timestamp?: string | null;
};

export type DashboardDayCount = {
  day: string;
  count: number;
};

export type DashboardResponse = {
  kpis: DashboardKpis;
  recentStatements: DashboardRecentStatement[];
  activityDaily: DashboardDayCount[];
};
