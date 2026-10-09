import { useState, type MouseEvent } from "react";
import { Menu, X } from "lucide-react";
import logo from "../assets/LRSolid_Logo.png";
import { appUrl } from "../lib/site.ts";

const links = [
    { href: "#features", label: "Features" },
    { href: "#architecture", label: "Architecture" },
    { href: "#start", label: "Get started" },
];

function homeHref(hash: string): string {
    const path = window.location.pathname.replace(/\/+$/, "") || "/";
    return path === "/" ? hash : `/${hash}`;
}

export function SiteHeader() {
    const [open, setOpen] = useState(false);

    const onMobileNavClick = (event: MouseEvent<HTMLAnchorElement>) => {
        setOpen(false);
        const next = new URL(event.currentTarget.href);
        const here = window.location.pathname.replace(/\/+$/, "") || "/";
        const there = next.pathname.replace(/\/+$/, "") || "/";
        if (here !== there) {
            event.preventDefault();
            window.location.assign(next.href);
        }
    };

    return (
        <header className="sticky top-0 z-20 border-b border-white/10 bg-slate-950/80 backdrop-blur">
            <div className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-6">
                <a href="/" className="shrink-0" aria-label="LRSolid" onClick={() => setOpen(false)}>
                    <img src={logo} alt="" className="h-8 w-auto" />
                </a>
                <nav className="hidden items-center gap-8 text-sm text-slate-300 md:flex">
                    {links.map((link) => (
                        <a key={link.href} href={homeHref(link.href)} className="hover:text-white">
                            {link.label}
                        </a>
                    ))}
                </nav>
                <div className="flex items-center gap-3">
                    <a
                        href={appUrl}
                        className="rounded-md bg-teal-400 px-3 py-1.5 text-sm font-semibold text-slate-950 hover:bg-teal-300 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-300"
                    >
                        Open app
                    </a>
                    <button
                        type="button"
                        className="rounded-md p-2 text-slate-300 hover:bg-white/5 hover:text-white md:hidden"
                        aria-expanded={open}
                        aria-label={open ? "Close menu" : "Open menu"}
                        onClick={() => setOpen((value) => !value)}
                    >
                        {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
                    </button>
                </div>
            </div>
            <nav className={`${open ? "block" : "hidden"} border-t border-white/10 px-6 py-3 md:hidden`}>
                {links.map((link) => (
                    <a
                        key={link.href}
                        href={homeHref(link.href)}
                        className="block py-2 text-sm text-slate-300 hover:text-white"
                        onClick={onMobileNavClick}
                    >
                        {link.label}
                    </a>
                ))}
            </nav>
        </header>
    );
}
