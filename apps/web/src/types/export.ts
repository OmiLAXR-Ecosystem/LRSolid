export type ExportSettings = {
  requireRemoteTargetAboveBytes: number;
  localMaxBytes: number;
};

export type UpdateExportSettingsPayload = Partial<ExportSettings>;

export type CreateStatementExportPayload = {
  projectId?: string;
  lrsId?: string;
  clientId?: string;
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
