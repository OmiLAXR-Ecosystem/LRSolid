const NAVIGATION_EVENT = "app:navigate";

export function navigate(to: string, opts?: { replace?: boolean }) {
    const { replace = false } = opts ?? {};
    if (replace) {
        window.history.replaceState(null, "", to);
    } else {
        window.history.pushState(null, "", to);
    }
    window.dispatchEvent(new Event(NAVIGATION_EVENT));
}

export function listenNavigation(listener: () => void) {
    window.addEventListener("popstate", listener);
    window.addEventListener(NAVIGATION_EVENT, listener);
    return () => {
        window.removeEventListener("popstate", listener);
        window.removeEventListener(NAVIGATION_EVENT, listener);
    };
}
