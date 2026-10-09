import { appUrl, githubUrl, xapiUrl } from "../lib/site.ts";

const statement = `{
  "actor": { "mbox": "mailto:learner@university.edu" },
  "verb": { "id": "http://adlnet.gov/expapi/verbs/completed" },
  "object": { "id": "https://example.edu/activities/intro" },
  "result": { "completion": true, "success": true }
}`;

export function Hero() {
    return (
        <section className="hero-wash border-b border-white/10">
            <div className="mx-auto grid max-w-6xl items-center gap-12 px-6 py-20 lg:grid-cols-[1.15fr_0.85fr] lg:py-28">
                <div>
                    <p className="text-sm font-medium uppercase tracking-[0.18em] text-teal-300">
                        Open source · xAPI 1.0.3 and 2.0
                    </p>
                    <h1 className="mt-4 max-w-xl text-4xl font-semibold tracking-tight text-white sm:text-5xl">
                        Open source, stable, and performant.
                    </h1>
                    <p className="mt-5 max-w-xl text-lg leading-relaxed text-slate-300">
                        LRSolid is a Learning Record Store for universities and enterprises.
                        It keeps xAPI 1.0.3 and 2.0 statements in PostgreSQL, stays quick on the write
                        path, and gives each organisation its own projects and LRS instances.
                    </p>
                    <div className="mt-8 flex flex-wrap gap-3">
                        <a
                            href={appUrl}
                            className="rounded-md bg-teal-400 px-4 py-2.5 text-sm font-semibold text-slate-950 hover:bg-teal-300 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-300"
                        >
                            Open the app
                        </a>
                        <a
                            href={githubUrl}
                            className="rounded-md border border-slate-600 px-4 py-2.5 text-sm font-semibold text-white hover:border-slate-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slate-300"
                        >
                            GitHub
                        </a>
                    </div>
                    <p className="mt-6 font-mono text-xs text-slate-400">
                        xAPI <span className="text-slate-200">{xapiUrl}</span>
                    </p>
                </div>
                <div className="rounded-xl border border-white/10 bg-slate-900/80 p-4 shadow-2xl shadow-teal-950/40">
                    <div className="flex items-center justify-between gap-3 border-b border-white/10 pb-3">
                        <p className="font-mono text-xs text-teal-300">POST /xapi/statements</p>
                        <p className="font-mono text-xs text-slate-500">Basic key_id:secret</p>
                    </div>
                    <pre className="mt-4 overflow-x-auto font-mono text-[13px] leading-relaxed text-slate-200">
                        {statement}
                    </pre>
                </div>
            </div>
        </section>
    );
}
