export type RestLRS = {
  id: string;
  projectId: string;
  name: string;
  dataStoreId: string;
  dataStoreName: string;
  statementCount?: number;
  createdAt: string;
};

export type RestLRSListResponse = {
  items: RestLRS[];
};

export type CreateLRSPayload = {
  name: string;
  projectId: string;
  dataStoreId: string;
};

export type RestProject = {
  id: string;
  slug: string;
  name: string;
  description?: string | null;
  statementCount?: number;
  createdAt: string;
};

export type RestProjectListResponse = {
  items: RestProject[];
};

export type CreateProjectPayload = {
  name: string;
  description?: string;
};

export type DataStoreCapabilities = {
  statements: boolean;
  state: boolean;
  profiles: boolean;
  orgWideQuery: boolean;
  exports: boolean;
  dashboardAggregates: boolean;
};

export type RestDataStore = {
  id: string;
  name: string;
  databaseType: string;
  host?: string;
  isDefault: boolean;
  isActive?: boolean;
  hasConnectionString?: boolean;
  description?: string | null;
  status?: "connected" | "control_plane" | "error" | "unavailable" | "inactive" | "not_configured" | string;
  statusError?: string;
  capabilities?: DataStoreCapabilities;
};

export type RestDataStoreListResponse = {
  items: RestDataStore[];
};
