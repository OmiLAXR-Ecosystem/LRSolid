import { Database, Gauge, KeyRound, Layers, LineChart, ShieldCheck, type LucideIcon } from "lucide-react";

const features: { title: string; body: string; icon: LucideIcon }[] = [
    {
        title: "xAPI 1.0.3 and 2.0",
        body: "Ingest and query statements, state, and agent and activity profiles on either version. The statement document stays in JSONB, with columns extracted for the filters you actually use.",
        icon: Database,
    },
    {
        title: "Organisation, project, LRS",
        body: "Each LRS is its own tenant. Roles are granted independently at organisation, project, and LRS level, and LRS access falls back to the project role.",
        icon: Layers,
    },
    {
        title: "Client credentials",
        body: "Machine clients authenticate with HTTP Basic (key_id:secret). A JSON scopes map, such as statements:write, limits what each client can do.",
        icon: KeyRound,
    },
    {
        title: "SSO",
        body: "Password login, OpenID Connect, and SAML 2.0 including Shibboleth issue the same httpOnly session cookie for the admin UI.",
        icon: ShieldCheck,
    },
    {
        title: "Stable under load",
        body: "PostgreSQL is the source of truth. Statement counters update in the background, so a write does not wait on dashboard aggregates.",
        icon: Gauge,
    },
    {
        title: "Analytics off the write path",
        body: "A planned pipeline streams statements through Kafka into ClickHouse or TimescaleDB, so dashboards do not query the hot statements table.",
        icon: LineChart,
    },
];

export function Features() {
    return (
        <section id="features" className="scroll-mt-20 border-b border-white/10">
            <div className="mx-auto max-w-6xl px-6 py-20">
                <h2 className="text-sm font-medium uppercase tracking-[0.18em] text-teal-300">Features</h2>
                <p className="mt-3 max-w-2xl text-3xl font-semibold tracking-tight text-white">
                    Built for more than one LRS on one database.
                </p>
                <ul className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    {features.map((feature) => {
                        const Icon = feature.icon;
                        return (
                            <li key={feature.title} className="rounded-xl border border-white/10 bg-slate-900/50 p-5">
                                <Icon className="h-5 w-5 text-teal-300" aria-hidden="true" />
                                <h3 className="mt-4 text-base font-semibold text-white">{feature.title}</h3>
                                <p className="mt-2 text-sm leading-relaxed text-slate-300">{feature.body}</p>
                            </li>
                        );
                    })}
                </ul>
            </div>
        </section>
    );
}
