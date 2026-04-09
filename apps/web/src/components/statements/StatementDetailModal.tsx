import { Ban, Download } from "lucide-react";
import { Modal } from "../ui/Modal";
import { JSONTree } from "./JSONTree";
import type { StatementDetail } from "../../types/statement";

type StatementDetailModalProps = {
    open: boolean;
    detail: StatementDetail | null;
    loading: boolean;
    error: string | null;
    canExport: boolean;
    canVoid: boolean;
    onClose: () => void;
    onExport: () => void;
    onVoid: () => void;
};

export function StatementDetailModal({
    open,
    detail,
    loading,
    error,
    canExport,
    canVoid,
    onClose,
    onExport,
    onVoid,
}: StatementDetailModalProps) {
    const footer = (
        <>
            <div className="text-xs text-slate-500">{detail?.id ?? ""}</div>
            <div className="flex items-center gap-2">
                <button
                    onClick={onExport}
                    disabled={!detail || !canExport}
                    className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                >
                    <Download className="h-3.5 w-3.5" />
                    Export
                </button>
                <button
                    onClick={onVoid}
                    disabled={!detail || !canVoid}
                    className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-red-600 hover:bg-red-50 disabled:opacity-50"
                >
                    <Ban className="h-3.5 w-3.5" />
                    Void
                </button>
            </div>
        </>
    );

    return (
        <Modal open={open} onClose={onClose} title="Statement Details" footer={footer}>
            {loading && <div className="text-sm text-slate-500 py-4">Loading...</div>}
            {error && <div className="text-sm text-red-600 py-2">{error}</div>}
            {!loading && detail && (
                <div className="grid grid-cols-[1fr_1fr] gap-6">
                    <div>
                        <div className="text-sm font-semibold text-slate-700 mb-2">Raw</div>
                        <pre className="rounded-lg bg-slate-50 p-3 text-xs text-slate-700 overflow-auto max-h-[60vh]">
                            {JSON.stringify(detail.raw, null, 2)}
                        </pre>
                    </div>
                    <div>
                        <div className="text-sm font-semibold text-slate-700 mb-2">Tree</div>
                        <div className="max-h-[60vh] overflow-auto">
                            <JSONTree data={detail.raw} />
                        </div>
                    </div>
                </div>
            )}
        </Modal>
    );
}
