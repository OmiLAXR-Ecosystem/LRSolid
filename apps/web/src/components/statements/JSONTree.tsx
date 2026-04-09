type JSONTreeProps = {
    data: unknown;
    depth?: number;
};

export function JSONTree({ data, depth = 0 }: JSONTreeProps) {
    if (data === null || data === undefined) {
        return <div className="text-slate-500">null</div>;
    }
    if (typeof data !== "object") {
        return <div className="text-slate-700">{String(data)}</div>;
    }

    const entries: [string, unknown][] = Array.isArray(data)
        ? data.map((v, i) => [String(i), v])
        : Object.entries(data as Record<string, unknown>);

    return (
        <div className="space-y-1">
            {entries.map(([key, value]) => (
                <details key={`${depth}-${key}`} className="rounded-md border border-slate-200 bg-white/70 px-3 py-2">
                    <summary className="cursor-pointer text-sm font-semibold text-slate-700">
                        {Array.isArray(data) ? `[#${key}]` : key}
                    </summary>
                    <div className="mt-2 pl-3 border-l border-slate-200">
                        <JSONTree data={value} depth={depth + 1} />
                    </div>
                </details>
            ))}
        </div>
    );
}
