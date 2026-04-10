export type RestLRS = {
  id: string;
  projectId: string;
  name: string;
  dataStoreId: string;
  dataStoreName: string;
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
  createdAt: string;
};

export type RestProjectListResponse = {
  items: RestProject[];
};

export type CreateProjectPayload = {
  name: string;
  description?: string;
};

export type RestDataStore = {
  id: string;
  name: string;
  databaseType: string;
  host: string;
  isDefault: boolean;
};

export type RestDataStoreListResponse = {
  items: RestDataStore[];
};
