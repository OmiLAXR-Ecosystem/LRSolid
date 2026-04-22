import { getStatementKeyColorClass } from "./keyColor";

type JSONTreeProps = {
    data: unknown;
    depth?: number;
    path?: string[];
    onKeyHover?: (path: string[] | null) => void;
};

function formatLeafValue(value: unknown): string {
    if (value === null || value === undefined) return "null";
    if (typeof value === "string") return `"${value}"`;
    return String(value);
}

export function JSONTree({ data, depth = 0, path = [], onKeyHover }: JSONTreeProps) {
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
            {entries.map(([key, value]) => {
                const isLeaf = value === null || value === undefined || typeof value !== "object";
                const currentPath = [...path, key];
                const keyNode = Array.isArray(data) ? (
                    <span className="text-slate-700">{`[#${key}]`}</span>
                ) : (
                    <span className={getStatementKeyColorClass(key)}>{key}</span>
                );

                if (isLeaf) {
                    return (
                        <div
                            key={`${depth}-${key}`}
                            className="rounded-md border border-slate-200 bg-white/70 px-3 py-2 text-sm"
                            onMouseEnter={() => {
                                onKeyHover?.(currentPath);
                            }}
                            onMouseLeave={() => onKeyHover?.(null)}
                        >
                            <span className="font-semibold">{keyNode}</span>
                            <span className="text-slate-500">: </span>
                            <span className="text-slate-700">{formatLeafValue(value)}</span>
                        </div>
                    );
                }

                return (
                    <details key={`${depth}-${key}`} className="rounded-md border border-slate-200 bg-white/70 px-3 py-2">
                        <summary
                            className="cursor-pointer text-sm font-semibold"
                            onMouseEnter={() => {
                                onKeyHover?.(currentPath);
                            }}
                            onMouseLeave={() => onKeyHover?.(null)}
                        >
                            {keyNode}
                        </summary>
                        <div className="mt-2 pl-3 border-l border-slate-200">
                            <JSONTree data={value} depth={depth + 1} path={currentPath} onKeyHover={onKeyHover} />
                        </div>
                    </details>
                );
            })}
        </div>
    );
}
