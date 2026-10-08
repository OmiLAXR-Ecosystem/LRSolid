import { useState, type FormEvent } from "react";
import { useAuth } from "../contexts/AuthContext";
import Logo from "../assets/LRSolid_Logo.png";
import { ApiError } from "../api/client";

const SSO_PROVIDERS = [
    {
        name: "Google",
        icon: (
            <svg className="h-5 w-5" viewBox="0 0 24 24" aria-hidden="true">
                <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
                <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
                <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z" />
                <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
            </svg>
        ),
    },
    {
        name: "Microsoft",
        icon: (
            <svg className="h-5 w-5" viewBox="0 0 24 24" aria-hidden="true">
                <path fill="#F25022" d="M1 1h10v10H1z" />
                <path fill="#7FBA00" d="M13 1h10v10H13z" />
                <path fill="#00A4EF" d="M1 13h10v10H1z" />
                <path fill="#FFB900" d="M13 13h10v10H13z" />
            </svg>
        ),
    },
];

export function LoginView() {
    const { login } = useAuth();
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        if (!email || !password) return;
        setLoading(true);
        setError(null);
        try {
            await login(email.trim(), password);
        } catch (err) {
            if (err instanceof ApiError && err.status === 401) {
                setError("Invalid email or password.");
            } else if (err instanceof Error) {
                setError(err.message || "Login failed.");
            } else {
                setError("Login failed.");
            }
        } finally {
            setLoading(false);
        }
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
                <div className="rounded-2xl bg-white shadow-sm ring-1 ring-slate-200 p-8">
                    <h1 className="text-xl font-semibold text-slate-800 mb-1">Sign in</h1>
                    <p className="text-sm text-slate-500 mb-6">to your LRSolid workspace</p>

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
                            {loading ? "Signing in..." : "Sign in"}
                        </button>
                    </form>

                    {/* Divider */}
                    <div className="relative my-6">
                        <div className="absolute inset-0 flex items-center">
                            <div className="w-full border-t border-slate-200" />
                        </div>
                        <div className="relative flex justify-center">
                            <span className="bg-white px-3 text-xs text-slate-400">or continue with</span>
                        </div>
                    </div>

                    {/* SSO — coming soon */}
                    <div className="space-y-2">
                        {SSO_PROVIDERS.map((provider) => (
                            <div key={provider.name} className="relative group">
                                <button
                                    type="button"
                                    disabled
                                    className="w-full flex items-center justify-center gap-3 rounded-lg border border-slate-200 bg-slate-50 px-4 py-2.5 text-sm font-medium text-slate-400 cursor-not-allowed"
                                >
                                    {provider.icon}
                                    Continue with {provider.name}
                                </button>
                                <span className="pointer-events-none absolute -top-7 left-1/2 -translate-x-1/2 rounded-md bg-slate-800 px-2 py-1 text-xs text-white opacity-0 group-hover:opacity-100 transition whitespace-nowrap">
                                    Coming soon
                                </span>
                            </div>
                        ))}
                    </div>
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
