import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { login as apiLogin, logout as apiLogout, me } from "../api/auth";
import type { AuthUser } from "../types/auth";

type AuthState =
    | { status: "loading" }
    | { status: "unauthenticated" }
    | { status: "authenticated"; user: AuthUser };

type AuthContextType = {
    state: AuthState;
    login: (email: string, password: string) => Promise<void>;
    logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
    const [state, setState] = useState<AuthState>({ status: "loading" });

    // Check session on mount
    useEffect(() => {
        me()
            .then((user) => setState({ status: "authenticated", user }))
            .catch(() => setState({ status: "unauthenticated" }));
    }, []);

    const login = useCallback(async (email: string, password: string) => {
        const user = await apiLogin({ email, password });
        setState({ status: "authenticated", user });
    }, []);

    const logout = useCallback(async () => {
        await apiLogout();
        setState({ status: "unauthenticated" });
    }, []);

    return (
        <AuthContext.Provider value={{ state, login, logout }}>
            {children}
        </AuthContext.Provider>
    );
}

export function useAuth(): AuthContextType {
    const ctx = useContext(AuthContext);
    if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
    return ctx;
}
