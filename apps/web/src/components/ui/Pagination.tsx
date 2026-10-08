const LIMIT_OPTIONS = [10, 25, 50, 100];

type PaginationProps = {
    label: string;
    limit: number;
    onLimitChange: (limit: number) => void;
    onPrev: () => void;
    onNext: () => void;
    prevDisabled: boolean;
    nextDisabled: boolean;
    // Optional: show page indicator (e.g. "2 / 5")
    currentPage?: number;
    totalPages?: number;
};

export function Pagination({
    label,
    limit,
    onLimitChange,
    onPrev,
    onNext,
    prevDisabled,
    nextDisabled,
    currentPage,
    totalPages,
}: PaginationProps) {
    return (
        <div className="mt-4 flex flex-col gap-3 text-sm text-slate-600 sm:flex-row sm:items-center sm:justify-between">
            <div className="min-w-0">{label}</div>
            <div className="flex flex-wrap items-center gap-2">
                <label className="text-xs font-semibold text-slate-500">Limit</label>
                <select
                    value={limit}
                    onChange={(e) => onLimitChange(Number(e.target.value))}
                    className="rounded-md border border-slate-200 bg-white px-2 py-1 text-sm"
                >
                    {LIMIT_OPTIONS.map((v) => (
                        <option key={v} value={v}>{v}</option>
                    ))}
                </select>
                <button
                    onClick={onPrev}
                    disabled={prevDisabled}
                    className="rounded-md border border-slate-200 px-2.5 py-1 text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                >
                    Previous
                </button>
                {currentPage !== undefined && totalPages !== undefined && (
                    <div className="flex items-center gap-2">
                        <span className="rounded-md border border-slate-200 bg-white px-2.5 py-1 text-slate-600">
                            {currentPage}
                        </span>
                        <span className="text-slate-400">/</span>
                        <span className="text-slate-600">{totalPages}</span>
                    </div>
                )}
                <button
                    onClick={onNext}
                    disabled={nextDisabled}
                    className="rounded-md border border-slate-200 px-2.5 py-1 text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                >
                    Next
                </button>
            </div>
        </div>
    );
}
