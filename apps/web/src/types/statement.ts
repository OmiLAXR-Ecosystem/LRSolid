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
    raw: unknown;
};
