import { appUrl, docsUrl, githubUrl } from "../lib/site.ts";

export function SiteFooter() {
    return (
        <footer className="border-t border-white/10">
            <div className="mx-auto flex max-w-6xl flex-col gap-6 px-6 py-10 sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <p className="text-sm font-medium text-white">LRSolid</p>
                    <p className="mt-1 text-sm text-slate-400">
                        Open source, stable, and performant LRS. Apache License 2.0.
                    </p>
                </div>
                <nav className="flex flex-wrap gap-x-6 gap-y-2 text-sm text-slate-300">
                    <a href={docsUrl} className="hover:text-white">Docs</a>
                    <a href={githubUrl} className="hover:text-white">GitHub</a>
                    <a href={appUrl} className="hover:text-white">App</a>
                    <a href="/imprint" className="hover:text-white">Impressum</a>
                    <a href="/privacy" className="hover:text-white">Datenschutz</a>
                </nav>
            </div>
        </footer>
    );
}
