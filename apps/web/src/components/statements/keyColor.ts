type KeyColorSpec = {
    textClass: string;
    tabClass: string;
    rawHighlightClass: string;
};

const DEFAULT_COLOR: KeyColorSpec = {
    textClass: "text-slate-700",
    tabClass: "border-slate-200 bg-slate-50 text-slate-700",
    rawHighlightClass: "bg-slate-100/80 ring-1 ring-inset ring-slate-200",
};

const KEY_COLORS: Record<string, KeyColorSpec> = {
    id: {
        textClass: "text-sky-700",
        tabClass: "border-sky-200 bg-sky-50 text-sky-700",
        rawHighlightClass: "bg-sky-100/70 ring-1 ring-inset ring-sky-200",
    },
    actor: {
        textClass: "text-emerald-700",
        tabClass: "border-emerald-200 bg-emerald-50 text-emerald-700",
        rawHighlightClass: "bg-emerald-100/70 ring-1 ring-inset ring-emerald-200",
    },
    verb: {
        textClass: "text-violet-700",
        tabClass: "border-violet-200 bg-violet-50 text-violet-700",
        rawHighlightClass: "bg-violet-100/70 ring-1 ring-inset ring-violet-200",
    },
    object: {
        textClass: "text-amber-700",
        tabClass: "border-amber-200 bg-amber-50 text-amber-700",
        rawHighlightClass: "bg-amber-100/70 ring-1 ring-inset ring-amber-200",
    },
    result: {
        textClass: "text-rose-700",
        tabClass: "border-rose-200 bg-rose-50 text-rose-700",
        rawHighlightClass: "bg-rose-100/70 ring-1 ring-inset ring-rose-200",
    },
    context: {
        textClass: "text-cyan-700",
        tabClass: "border-cyan-200 bg-cyan-50 text-cyan-700",
        rawHighlightClass: "bg-cyan-100/70 ring-1 ring-inset ring-cyan-200",
    },
    timestamp: {
        textClass: "text-indigo-700",
        tabClass: "border-indigo-200 bg-indigo-50 text-indigo-700",
        rawHighlightClass: "bg-indigo-100/70 ring-1 ring-inset ring-indigo-200",
    },
    stored: {
        textClass: "text-indigo-700",
        tabClass: "border-indigo-200 bg-indigo-50 text-indigo-700",
        rawHighlightClass: "bg-indigo-100/70 ring-1 ring-inset ring-indigo-200",
    },
    authority: {
        textClass: "text-fuchsia-700",
        tabClass: "border-fuchsia-200 bg-fuchsia-50 text-fuchsia-700",
        rawHighlightClass: "bg-fuchsia-100/70 ring-1 ring-inset ring-fuchsia-200",
    },
    version: {
        textClass: "text-orange-700",
        tabClass: "border-orange-200 bg-orange-50 text-orange-700",
        rawHighlightClass: "bg-orange-100/70 ring-1 ring-inset ring-orange-200",
    },
    attachments: {
        textClass: "text-orange-700",
        tabClass: "border-orange-200 bg-orange-50 text-orange-700",
        rawHighlightClass: "bg-orange-100/70 ring-1 ring-inset ring-orange-200",
    },
};

export function getStatementKeyColorClass(key: string): string {
    return (KEY_COLORS[key] ?? DEFAULT_COLOR).textClass;
}

export function getStatementTabColorClass(key: string): string {
    return (KEY_COLORS[key] ?? DEFAULT_COLOR).tabClass;
}

export function getStatementRawHighlightClass(key: string): string {
    return (KEY_COLORS[key] ?? DEFAULT_COLOR).rawHighlightClass;
}
