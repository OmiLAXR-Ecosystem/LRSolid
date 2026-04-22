export type ExportSettings = {
  requireRemoteTargetAboveBytes: number;
  localMaxBytes: number;
  defaultTargetType: "local" | "s3" | "gdrive" | "onedrive" | "icloud";
  s3UploadBaseUrl?: string;
  s3PublicBaseUrl?: string;
  s3AuthToken?: string;
  s3PathPrefix: string;
};

export type UpdateExportSettingsPayload = Partial<ExportSettings>;

// ---- Filter types ----

export type TextOperator = "eq" | "neq" | "contains";
export type NumericOperator = "eq" | "neq" | "lt" | "lte" | "gt" | "gte";
export type TimestampOperator = "eq" | "lt" | "lte" | "gt" | "gte" | "between";
export type ExtensionOperator = TextOperator | NumericOperator;

export type FieldCondition<Op extends string = string> = {
  operator: Op;
  value: string;
  valueTo?: string; // only used when operator = "between"
};

export type ExtensionCondition = {
  key: string;
  operator: ExtensionOperator;
  value: string;
};

export type ExportFilterState = {
  // Scope
  projectId: string;
  lrsId: string;
  clientId: string;
  // Statement fields
  actorMbox: FieldCondition<TextOperator> | null;
  verbId: FieldCondition<TextOperator> | null;
  objectId: FieldCondition<TextOperator> | null;
  // Timestamp
  storedAt: FieldCondition<TimestampOperator> | null;
  // Extensions
  extensions: ExtensionCondition[];
  // Target
  targetType: string;
  targetPath: string;
};

export function emptyExportFilter(): ExportFilterState {
  return {
    projectId: "",
    lrsId: "",
    clientId: "",
    actorMbox: null,
    verbId: null,
    objectId: null,
    storedAt: null,
    extensions: [],
    targetType: "",
    targetPath: "",
  };
}

// ---- API payload types ----

export type CreateStatementExportPayload = {
  projectId?: string;
  lrsId?: string;
  clientId?: string;
  actorMbox?: FieldCondition<TextOperator>;
  verbId?: FieldCondition<TextOperator>;
  objectId?: FieldCondition<TextOperator>;
  storedAt?: FieldCondition<TimestampOperator>;
  extensions?: ExtensionCondition[];
  format?: "ndjson";
  compression?: "gzip" | "none";
  targetType?: "s3" | "gdrive" | "onedrive" | "icloud";
  targetPath?: string;
};

export type ExportJob = {
  id: string;
  status: "queued" | "running" | "completed" | "failed" | "canceled";
  filters: Record<string, unknown>;
  format: string;
  compression: string;
  estimatedRows: number;
  estimatedBytes: number;
  targetType?: string | null;
  targetPath?: string | null;
  artifactRef?: string | null;
  errorMessage?: string | null;
  createdAt: string;
  startedAt?: string | null;
  finishedAt?: string | null;
};

export type ExportJobListResponse = {
  items: ExportJob[];
};
