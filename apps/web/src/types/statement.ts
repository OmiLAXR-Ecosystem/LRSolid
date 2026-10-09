export type StatementRow = {
    id: string;
    actorMbox?: string | null;
    verbId?: string | null;
    objectId?: string | null;
    timestamp?: string | null;
    storedAt?: string | null;
};

export type ConnectionEdge = {
    cursor: string;
    node: StatementRow;
};

export type ConnectionResponse = {
    edges: ConnectionEdge[];
    pageInfo: { hasNextPage: boolean; endCursor?: string };
};

export type StatementDetail = {
    id: string;
    storedAt: string;
    timestamp?: string | null;
    actorMbox?: string | null;
    verbId?: string | null;
    objectId?: string | null;
    version?: string | null;
    authority?: string | null;
    metadata: {
        tenantId: string;
        lrsId: string;
        lrsName: string;
        projectId: string;
        projectSlug: string;
        projectName: string;
        organizationId: string;
        organizationSlug: string;
        organizationName: string;
        dataStoreId: string;
        dataStoreName: string;
        dataStoreType: string;
        clientId?: string | null;
        clientName?: string | null;
    };
    raw: unknown;
};
