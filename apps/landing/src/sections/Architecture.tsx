import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { statementCurl } from "../lib/site.ts";

const steps = [
    { name: "React admin", detail: "Session cookie, org and LRS views" },
    { name: "Go API", detail: "Chi on :8080, /api and /xapi" },
    { name: "PostgreSQL", detail: "Statements, users, clients" },
];

export function Architecture() {
    const curl = statementCurl();
    const [copied, setCopied] = useState(false);

    const copy = async () => {
        try {
            await navigator.clipboard.writeText(curl);
            setCopied(true);
            window.setTimeout(() => setCopied(false), 1600);
        } catch {
            setCopied(false);
        }
    };

    return (
        <section id="architecture" className="scroll-mt-20 border-b border-white/10">
            <div className="mx-auto max-w-6xl px-6 py-20">
                <h2 className="text-sm font-medium uppercase tracking-[0.18em] text-teal-300">Architecture</h2>
                <p className="mt-3 max-w-2xl text-3xl font-semibold tracking-tight text-white">
                    The browser talks to one API. Postgres keeps the record.
                </p>
                <ol className="mt-10 grid gap-3 md:grid-cols-3">
                    {steps.map((step, index) => (
                        <li key={step.name} className="rounded-xl border border-white/10 bg-slate-900/50 p-5">
                            <p className="font-mono text-xs text-teal-300">{String(index + 1).padStart(2, "0")}</p>
                            <p className="mt-2 text-base font-semibold text-white">{step.name}</p>
                            <p className="mt-1 text-sm text-slate-400">{step.detail}</p>
                        </li>
                    ))}
                </ol>
                <p className="mt-4 text-sm text-slate-400">
                    Planned, off the ingest path: an outbox feeds Kafka, and a separate service writes aggregates to ClickHouse or TimescaleDB.
                </p>
                <div className="mt-8 overflow-hidden rounded-xl border border-white/10 bg-slate-900">
                    <div className="flex items-center justify-between gap-3 border-b border-white/10 px-4 py-3">
                        <p className="font-mono text-xs text-slate-400">POST a statement</p>
                        <button
                            type="button"
                            onClick={copy}
                            className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs text-slate-300 hover:bg-white/5 hover:text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-300"
                        >
                            {copied ? <Check className="h-3.5 w-3.5 text-teal-300" /> : <Copy className="h-3.5 w-3.5" />}
                            {copied ? "Copied" : "Copy"}
                        </button>
                    </div>
                    <pre className="overflow-x-auto px-4 py-4 font-mono text-[13px] leading-relaxed text-slate-200">
                        {curl}
                    </pre>
                </div>
            </div>
        </section>
    );
}
