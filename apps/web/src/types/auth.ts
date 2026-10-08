export type AuthUser = {
    id: string;
    email: string;
    role: string;
    authProvider?: string;
    ssoOnly?: boolean;
};
