import { useEffect } from "react";
import { useStatements } from "../hooks/useStatements";
import { useConfirm } from "../contexts/ConfirmContext";
import { downloadStatementJson, fetchStatementById } from "../api/statements";
import { StatementFilters } from "../components/statements/StatementFilters";
import { StatementTable } from "../components/statements/StatementTable";
import { StatementDetailModal } from "../components/statements/StatementDetailModal";

const canExport = import.meta.env.VITE_CAN_EXPORT !== "false";
const canVoid = import.meta.env.VITE_CAN_VOID !== "false";

export function StatementsView() {
    const confirm = useConfirm();
    const stmts = useStatements(confirm);

    useEffect(() => {
        const hash = window.location.hash;
        const qIndex = hash.indexOf("?");
        if (qIndex < 0) return;
        const query = hash.slice(qIndex + 1);
        const id = new URLSearchParams(query).get("statementId");
        if (!id) return;
        void stmts.openDetail(id);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const handleExport = async (id: string) => {
        // Use cached detail if available, otherwise fetch
        const detail = stmts.detail?.id === id ? stmts.detail : await fetchStatementById(id);
        if (detail) downloadStatementJson(detail);
    };

    return (
        <div className="space-y-6">
            <StatementFilters
                filters={stmts.filters}
                onChange={stmts.setFilter}
                onApply={stmts.applyFilters}
                loading={stmts.loading}
            />
            <StatementTable
                items={stmts.items}
                loading={stmts.loading}
                error={stmts.error}
                limit={stmts.filters.limit}
                onLimitChange={(v) => stmts.setFilter("limit", v)}
                onRefresh={stmts.refresh}
                onViewDetails={stmts.openDetail}
                onExport={handleExport}
                onVoid={stmts.voidDetail}
                canExport={canExport}
                canVoid={canVoid}
                pageIndex={stmts.pageIndex}
                hasNextPage={stmts.hasNextPage}
                onNext={stmts.goNext}
                onPrev={stmts.goPrev}
            />
            <StatementDetailModal
                open={stmts.detailOpen}
                detail={stmts.detail}
                loading={stmts.detailLoading}
                error={stmts.detailError}
                canExport={canExport}
                canVoid={canVoid}
                onClose={stmts.closeDetail}
                onExport={stmts.exportDetail}
                onVoid={stmts.voidDetail}
            />
        </div>
    );
}
