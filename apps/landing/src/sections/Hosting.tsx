import { appUrl, githubUrl, xapiUrl } from "../lib/site.ts";

const compose = `git clone ${githubUrl}.git
cd LRSolid
docker compose up -d --build`;

export function Hosting() {
    return (
        <section id="start" className="scroll-mt-20">
            <div className="mx-auto max-w-6xl px-6 py-20">
                <h2 className="text-sm font-medium uppercase tracking-[0.18em] text-teal-300">Get started</h2>
                <p className="mt-3 max-w-2xl text-3xl font-semibold tracking-tight text-white">
                    Open the app and post a statement.
                </p>
                <p className="mt-4 max-w-2xl text-sm leading-relaxed text-slate-400">
                    Login, the admin UI, and the xAPI endpoint live on the app host.
                    A Docker Compose stack is there if you want your own instance.
                </p>
                <div className="mt-10 grid gap-4 lg:grid-cols-2">
                    <article className="flex flex-col rounded-xl border border-teal-400/30 bg-teal-400/5 p-6">
                        <h3 className="text-lg font-semibold text-white">Hosted</h3>
                        <p className="mt-3 flex-1 text-sm leading-relaxed text-slate-300">
                            Sign in at {appUrl.replace(/^https:\/\//, "")}. Learning tools post statements to {xapiUrl} with a client key and secret.
                        </p>
                        <a
                            href={appUrl}
                            className="mt-6 inline-flex w-fit rounded-md bg-teal-400 px-4 py-2 text-sm font-semibold text-slate-950 hover:bg-teal-300"
                        >
                            Open the app
                        </a>
                    </article>
                    <article className="flex flex-col rounded-xl border border-white/10 bg-slate-900/50 p-6">
                        <h3 className="text-lg font-semibold text-white">Your own instance</h3>
                        <p className="mt-3 text-sm leading-relaxed text-slate-300">
                            Docker Compose starts Postgres, the API, and the admin UI.
                            The admin UI listens on port 80 and the API on 8080.
                        </p>
                        <pre className="mt-6 overflow-x-auto rounded-lg bg-slate-950 p-4 font-mono text-[13px] leading-relaxed text-slate-200">
                            {compose}
                        </pre>
                    </article>
                </div>
            </div>
        </section>
    );
}
