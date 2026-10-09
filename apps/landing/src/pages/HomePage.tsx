import { useEffect } from "react";
import { SiteFooter } from "../components/SiteFooter.tsx";
import { SiteHeader } from "../components/SiteHeader.tsx";
import { Architecture } from "../sections/Architecture.tsx";
import { Features } from "../sections/Features.tsx";
import { Hero } from "../sections/Hero.tsx";
import { Hosting } from "../sections/Hosting.tsx";

export function HomePage() {
    useEffect(() => {
        const id = window.location.hash.slice(1);
        if (!id) return;
        document.getElementById(id)?.scrollIntoView();
    }, []);

    return (
        <div className="min-h-screen">
            <SiteHeader />
            <main>
                <Hero />
                <Features />
                <Architecture />
                <Hosting />
            </main>
            <SiteFooter />
        </div>
    );
}
