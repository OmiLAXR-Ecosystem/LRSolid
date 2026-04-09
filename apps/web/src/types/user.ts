export type RestUser = {
    id: string;
    displayName?: string | null;
    email: string;
    role: string;
    disabledAt?: string | null;
};

export type RestUserResponse = {
    items: RestUser[];
    total: number;
    limit: number;
    offset: number;
};
