import { useEffect, useState, type FormEvent } from "react";
import { useAuth } from "../contexts/AuthContext";
import Logo from "../assets/LRSolid_Logo.png";
import { ApiError } from "../api/client";
import { acceptInvite, getInvite, listLoginProviders, ssoStartPath, type InviteInfo, type LoginProvider } from "../api/sso";

const LOGIN_ERRORS: Record<string, string> = {
    no_account: "No LRSolid account exists for this identity. Ask an administrator for an invite.",
    disabled: "This account is disabled.",
    email_unverified: "The identity provider did not confirm your email address.",
    email_required: "The identity provider did not release an email address.",
    transient_subject: "The identity provider did not release a stable user identifier.",
    unknown_provider: "That sign-in method is not available.",
    provider_error: "Sign-in with the identity provider failed. Try again.",
};

function orgFromLocation(): string {
    const first = window.location.pathname.replace(/^\/+|\/+$/g, "").split("/")[0] ?? "";
    if (!first || first === "login" || first === "accept-invite" || first === "profile") return "";
    return first;
}

function returnPath(inviteOrg: string): string {
    const path = window.location.pathname;
    if (path === "/login" || path === "/accept-invite") {
        return inviteOrg ? `/${inviteOrg}/dashboard` : "/";
    }
    return path + window.location.search;
}

export function LoginView() {
    const { login } = useAuth();
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [providers, setProviders] = useState<LoginProvider[]>([]);
    const [localLogin, setLocalLogin] = useState(true);
    const [invite, setInvite] = useState<InviteInfo | null>(null);
    const inviteToken = window.location.pathname === "/accept-invite"
        ? new URLSearchParams(window.location.search).get("token")
        : null;

    useEffect(() => {
        const code = new URLSearchParams(window.location.search).get("error");
        if (code) setError(LOGIN_ERRORS[code] ?? "Sign-in failed.");
    }, []);

    useEffect(() => {
        let cancelled = false;
        listLoginProviders(orgFromLocation())
            .then((res) => {
                if (cancelled) return;
                setProviders(res.providers);
                setLocalLogin(res.localLoginEnabled);
            })
            .catch(() => {
                if (!cancelled) setLocalLogin(true);
            });
        return () => {
            cancelled = true;
        };
    }, []);

    useEffect(() => {
        if (!inviteToken) return;
        let cancelled = false;
        getInvite(inviteToken)
            .then((info) => {
                if (cancelled) return;
                setInvite(info);
                setEmail(info.email);
            })
            .catch((err) => {
                if (!cancelled) setError(err instanceof Error ? err.message : "Invite not found.");
            });
        return () => {
            cancelled = true;
        };
    }, [inviteToken]);

    const showPassword = localLogin;

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        if (!email || !password) return;
        setLoading(true);
        setError(null);
        try {
            if (inviteToken) {
                await acceptInvite(inviteToken, password);
                window.location.assign(returnPath(invite?.orgSlug ?? ""));
                return;
            }
            await login(email.trim(), password);
        } catch (err) {
            if (err instanceof ApiError && err.status === 401) {
                setError("Invalid email or password.");
            } else if (err instanceof ApiError && err.status === 403) {
                setError("Password sign-in is disabled for this organisation.");
            } else if (err instanceof Error) {
                setError(err.message || "Login failed.");
            } else {
                setError("Login failed.");
            }
        } finally {
            setLoading(false);
        }
    };

    const startSSO = (provider: LoginProvider) => {
        const org = invite?.orgSlug || orgFromLocation();
        window.location.assign(ssoStartPath(provider.slug, returnPath(org), org, inviteToken ?? undefined));
    };

    return (
        <div className="min-h-screen bg-slate-100 flex items-center justify-center p-4">
            <div className="w-full max-w-sm">
                <div className="mb-8 flex justify-center">
                    <img
                        src={Logo}
                        alt="LRSolid"
                        className="brand-logo h-16 w-auto"
                    />
                </div>

                {/* Card */}
                <div className="rounded-2xl bg-white p-5 shadow-sm ring-1 ring-slate-200 sm:p-8">
                    <h1 className="text-xl font-semibold text-slate-800 mb-1">{invite ? "Accept invite" : "Sign in"}</h1>
                    <p className="text-sm text-slate-500 mb-6">
                        {invite
                            ? `Join ${invite.orgName} as ${invite.role}.`
                            : "to your LRSolid workspace"}
                    </p>

                    {showPassword && (
                    <form onSubmit={handleSubmit} className="space-y-4">
                        <div>
                            <label className="block text-xs font-semibold text-slate-600 mb-1">
                                Email
                            </label>
                            <input
                                type="email"
                                autoComplete="email"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                placeholder="you@example.com"
                                required
                                className="w-full rounded-lg bg-slate-50 px-3 py-2.5 text-sm text-slate-800 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-400 focus:outline-none"
                            />
                        </div>
                        <div>
                            <label className="block text-xs font-semibold text-slate-600 mb-1">
                                Password
                            </label>
                            <input
                                type="password"
                                autoComplete="current-password"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                placeholder="••••••••"
                                required
                                className="w-full rounded-lg bg-slate-50 px-3 py-2.5 text-sm text-slate-800 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-400 focus:outline-none"
                            />
                        </div>

                        {error && (
                            <p className="text-xs text-red-600 bg-red-50 rounded-lg px-3 py-2">
                                {error}
                            </p>
                        )}

                        <button
                            type="submit"
                            disabled={loading}
                            className="w-full rounded-lg bg-teal-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-teal-700 transition disabled:opacity-60"
                        >
                            {loading ? "Signing in..." : invite ? "Accept invite" : "Sign in"}
                        </button>
                    </form>
                    )}

                    {!showPassword && error && (
                        <p className="text-xs text-red-600 bg-red-50 rounded-lg px-3 py-2 mb-4">{error}</p>
                    )}

                    {providers.length > 0 && (
                    <>
                    {showPassword && (
                    <div className="relative my-6">
                        <div className="absolute inset-0 flex items-center">
                            <div className="w-full border-t border-slate-200" />
                        </div>
                        <div className="relative flex justify-center">
                            <span className="bg-white px-3 text-xs text-slate-400">or continue with</span>
                        </div>
                    </div>
                    )}

                    <div className="space-y-2">
                        {providers.map((provider) => (
                            <button
                                key={provider.slug}
                                type="button"
                                onClick={() => startSSO(provider)}
                                className="w-full flex items-center justify-center gap-3 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-sm font-medium text-slate-700 hover:bg-slate-50"
                            >
                                Continue with {provider.displayName}
                            </button>
                        ))}
                    </div>
                    </>
                    )}
                </div>

                {/* Copyright */}
                <p className="mt-6 text-center text-xs text-slate-400">
                    © 2026{" "}
                    <a
                        href="https://goerzen.consulting"
                        target="_blank"
                        rel="noreferrer"
                        className="hover:text-slate-600 hover:underline"
                    >
                        Dr.-Ing. Sergej Görzen
                    </a>
                    . All rights reserved.
                    <span className="mx-1.5" aria-hidden="true">·</span>
                    <a
                        href="https://github.com/SGoerzen"
                        target="_blank"
                        rel="noreferrer"
                        className="hover:text-slate-600 hover:underline"
                    >
                        GitHub
                    </a>
                </p>
            </div>
        </div>
    );
}
