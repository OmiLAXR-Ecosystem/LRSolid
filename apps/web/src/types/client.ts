export type RestLRS = {
  id: string;
  projectId: string;
  name: string;
  dataStoreId: string;
  dataStoreName: string;
  statementCount?: number;
};

export type RestLRSResponse = {
  items: RestLRS[];
};

export type RestClient = {
  id: string;
  projectId: string;
  lrsId?: string | null;
  lrsName?: string | null;
  name: string;
  statementCount?: number;
  note?: string | null;
  keyId: string;
  scopes: Record<string, boolean>;
  createdAt: string;
  expiresAt?: string | null;
  revokedAt?: string | null;
  userIds: string[];
};

export type RestClientResponse = {
  items: RestClient[];
  total: number;
  limit: number;
  offset: number;
};

export type CreateClientPayload = {
  name: string;
  lrsId: string;
  note?: string;
  expiresAt?: string;
  userIds?: string[];
  scopes?: Record<string, boolean>;
};

export type CreateClientResponse = {
  id: string;
  projectId: string;
  lrsId?: string | null;
  lrsName?: string | null;
  name: string;
  statementCount?: number;
  note?: string | null;
  keyId: string;
  secret: string;
  scopes: Record<string, boolean>;
  createdAt: string;
  expiresAt?: string | null;
  userIds: string[];
};
