import type { ReactNode } from "react";
import { SiteFooter } from "../components/SiteFooter.tsx";
import { SiteHeader } from "../components/SiteHeader.tsx";

type LegalPageProps = {
    title: string;
    kicker: string;
    children: ReactNode;
};

export function LegalPage({ title, kicker, children }: LegalPageProps) {
    return (
        <div className="flex min-h-screen flex-col">
            <SiteHeader />
            <main className="mx-auto w-full max-w-3xl flex-1 px-6 py-16">
                <p className="text-sm font-medium uppercase tracking-[0.18em] text-teal-300">{kicker}</p>
                <h1 className="mt-3 text-3xl font-semibold tracking-tight text-white">{title}</h1>
                <div className="mt-6 space-y-4 text-sm leading-relaxed text-slate-300">{children}</div>
            </main>
            <SiteFooter />
        </div>
    );
}
