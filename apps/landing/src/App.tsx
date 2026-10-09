import { HomePage } from "./pages/HomePage.tsx";
import { LegalPage } from "./pages/LegalPage.tsx";

function pathName(): string {
    return window.location.pathname.replace(/\/+$/, "") || "/";
}

export default function App() {
    const path = pathName();

    if (path === "/imprint") {
        return (
            <LegalPage title="Impressum" kicker="Legal">
                <p>
                    This page is a placeholder for the operator of lrsolid.cloud. Publish the name of the
                    responsible person or organisation, a postal address, and a contact address before the site goes live.
                </p>
                <p>
                    LRSolid itself is open-source software under the Apache License 2.0. That license is not a substitute
                    for the site operator&apos;s legal notice.
                </p>
            </LegalPage>
        );
    }

    if (path === "/privacy") {
        return (
            <LegalPage title="Datenschutz" kicker="Legal">
                <p>
                    This page is a placeholder for the privacy notice of lrsolid.cloud. Describe which data this site and
                    the app host process, on what basis, and how to reach the operator.
                </p>
                <p>
                    The landing page does not set tracking cookies and does not call the LRSolid API. The admin UI on the
                    app host uses an httpOnly session cookie after sign-in.
                </p>
            </LegalPage>
        );
    }

    return <HomePage />;
}
