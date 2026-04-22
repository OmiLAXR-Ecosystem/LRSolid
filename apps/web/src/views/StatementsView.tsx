import { useEffect, useState } from "react";
import { useStatements } from "../hooks/useStatements";
import { useConfirm } from "../contexts/ConfirmContext";
import { downloadStatementJson, fetchStatementById } from "../api/statements";
import { StatementFilters } from "../components/statements/StatementFilters";
import { StatementTable } from "../components/statements/StatementTable";
import { StatementDetailModal } from "../components/statements/StatementDetailModal";
import { ExportFilterForm } from "../components/exports/ExportFilterForm";
import { Modal } from "../components/ui/Modal";
import { useOrg } from "../contexts/OrgContext";
import { fetchProjects, fetchLRSList } from "../api/lrs";
import { fetchClients } from "../api/clients";
import { createStatementExport, downloadExportJob, fetchExportJobs, fetchExportSettings } from "../api/export";
import { emptyExportFilter } from "../types/export";
import type { RestProject, RestLRS } from "../types/lrs";
import type { RestClient } from "../types/client";
import type { ExportFilterState, ExportJob } from "../types/export";

const canExport = import.meta.env.VITE_CAN_EXPORT !== "false";
const canVoid = import.meta.env.VITE_CAN_VOID !== "false";

function fmtBytes(n: number): string {
    if (n >= 1024 * 1024) return `${Math.round((n / (1024 * 1024)) * 10) / 10} MB`;
    if (n >= 1024) return `${Math.round(n / 1024)} KB`;
    return `${n} B`;
}

export function StatementsView() {
    const confirm = useConfirm();
    const stmts = useStatements(confirm);
    const { currentOrg } = useOrg();

    const [open, setOpen] = useState(false);
    const [initLoading, setInitLoading] = useState(false);
    const [submitLoading, setSubmitLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [successMessage, setSuccessMessage] = useState<string | null>(null);

    const [projects, setProjects] = useState<RestProject[]>([]);
    const [lrsItems, setLrsItems] = useState<RestLRS[]>([]);
    const [clients, setClients] = useState<RestClient[]>([]);
    const [thresholdGB, setThresholdGB] = useState<number | null>(null);
    const [filters, setFilters] = useState<ExportFilterState>(emptyExportFilter());

    const [jobs, setJobs] = useState<ExportJob[]>([]);
    const [jobsLoading, setJobsLoading] = useState(false);
    const [jobsError, setJobsError] = useState<string | null>(null);
    const [downloadingJobId, setDownloadingJobId] = useState<string | null>(null);
    const [downloadError, setDownloadError] = useState<string | null>(null);

    useEffect(() => {
        const id = new URLSearchParams(window.location.search).get("statementId");
        if (!id) return;
        void stmts.openDetail(id);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const loadJobs = async () => {
        if (!currentOrg) return;
        setJobsLoading(true);
        setJobsError(null);
        try {
            const res = await fetchExportJobs(currentOrg, 50);
            setJobs(res.items ?? []);
        } catch (err) {
            setJobsError(err instanceof Error ? err.message : "Failed to load export history");
        } finally {
            setJobsLoading(false);
        }
    };

    const openDialog = async () => {
        if (!currentOrg) return;
        setOpen(true);
        setError(null);
        setSuccessMessage(null);
        setInitLoading(true);
        try {
            const [projectsRes, lrsRes, clientsRes, settings] = await Promise.all([
                fetchProjects(currentOrg),
                fetchLRSList(currentOrg),
                fetchClients(currentOrg, { limit: 200, offset: 0 }),
                fetchExportSettings(currentOrg),
            ]);
            setProjects(projectsRes.items ?? []);
            setLrsItems(lrsRes.items ?? []);
            setClients(clientsRes.items ?? []);
            setThresholdGB(Math.round((settings.requireRemoteTargetAboveBytes / (1024 * 1024 * 1024)) * 100) / 100);
            setFilters((prev) => ({
                ...prev,
                targetType: settings.defaultTargetType === "local" ? "" : settings.defaultTargetType,
                targetPath: "",
            }));
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to load export options");
        } finally {
            setInitLoading(false);
        }
        await loadJobs();
    };

    const submitExport = async () => {
        if (!currentOrg) return;
        setSubmitLoading(true);
        setError(null);
        setSuccessMessage(null);
        try {
            const job = await createStatementExport(currentOrg, {
                projectId: filters.projectId || undefined,
                lrsId: filters.lrsId || undefined,
                clientId: filters.clientId || undefined,
                actorMbox: filters.actorMbox ?? undefined,
                verbId: filters.verbId ?? undefined,
                objectId: filters.objectId ?? undefined,
                storedAt: filters.storedAt ?? undefined,
                extensions: filters.extensions.length > 0 ? filters.extensions : undefined,
                format: "ndjson",
                compression: "gzip",
                statementWrapper: filters.statementWrapper,
                targetType: (filters.targetType as "s3" | "gdrive" | "onedrive" | "icloud") || undefined,
                targetPath: filters.targetPath || undefined,
            });
            setSuccessMessage(`Export created — ~${fmtBytes(job.estimatedBytes)} estimated`);
            await loadJobs();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to create export");
        } finally {
            setSubmitLoading(false);
        }
    };

    const handleDownloadJob = async (jobId: string) => {
        if (!currentOrg) return;
        setDownloadingJobId(jobId);
        setDownloadError(null);
        try {
            await downloadExportJob(currentOrg, jobId);
        } catch (err) {
            setDownloadError(err instanceof Error ? err.message : "Download failed");
        } finally {
            setDownloadingJobId(null);
        }
    };

    const handleExport = async (id: string) => {
        const detail = stmts.detail?.id === id ? stmts.detail : await fetchStatementById(id);
        if (detail) downloadStatementJson(detail);
    };

    const footer = (
        <>
            <div />
            <div className="flex items-center gap-2">
                <button
                    onClick={() => setOpen(false)}
                    className="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-50"
                >
                    Close
                </button>
                <button
                    onClick={() => void submitExport()}
                    disabled={submitLoading || initLoading}
                    className="rounded-lg bg-teal-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-teal-700 disabled:opacity-50"
                >
                    {submitLoading ? "Creating..." : "Create Export"}
                </button>
            </div>
        </>
    );

    return (
        <div className="space-y-6">
            <div className="flex justify-end">
                <button
                    onClick={() => void openDialog()}
                    className="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50"
                >
                    Export Statements
                </button>
            </div>
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
            <Modal open={open} onClose={() => setOpen(false)} title="Export Statements" footer={footer} maxWidth="max-w-4xl">
                <div className="space-y-6">
                    {error && <div className="text-sm text-red-600">{error}</div>}
                    {successMessage && <div className="text-sm text-emerald-700">{successMessage}</div>}

                    <ExportFilterForm
                        value={filters}
                        onChange={setFilters}
                        projects={projects}
                        lrsItems={lrsItems}
                        clients={clients}
                        thresholdGB={thresholdGB}
                        loading={initLoading}
                    />

                    {/* Export history */}
                    <div className="space-y-2">
                        <div className="flex items-center justify-between">
                            <div className="text-xs font-semibold uppercase tracking-wide text-slate-400">Previous Exports</div>
                            <button
                                onClick={() => void loadJobs()}
                                disabled={jobsLoading}
                                className="text-xs text-slate-400 hover:text-slate-600 disabled:opacity-50"
                            >
                                {jobsLoading ? "Refreshing..." : "Refresh"}
                            </button>
                        </div>
                        {jobsError && <div className="text-sm text-red-600">{jobsError}</div>}
                        {downloadError && <div className="text-sm text-red-600">{downloadError}</div>}
                        {jobsLoading && jobs.length === 0 && <div className="text-sm text-slate-400">Loading...</div>}
                        {!jobsLoading && jobs.length === 0 && <div className="text-sm text-slate-400">No exports yet.</div>}
                        {jobs.length > 0 && (
                            <div className="overflow-hidden rounded-lg ring-1 ring-slate-200">
                                <table className="min-w-full text-sm">
                                    <thead className="bg-slate-50 text-slate-500">
                                        <tr>
                                            <th className="px-3 py-2 text-left text-xs font-semibold">Created</th>
                                            <th className="px-3 py-2 text-left text-xs font-semibold">Status</th>
                                            <th className="px-3 py-2 text-left text-xs font-semibold">Size est.</th>
                                            <th className="px-3 py-2 text-left text-xs font-semibold">Target</th>
                                            <th className="px-3 py-2 text-left text-xs font-semibold">Download</th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y divide-slate-100">
                                        {jobs.map((j) => (
                                            <tr key={j.id}>
                                                <td className="px-3 py-2 text-slate-500 text-xs">{new Date(j.createdAt).toLocaleString()}</td>
                                                <td className="px-3 py-2">
                                                    <span className={`inline-block rounded px-1.5 py-0.5 text-xs font-medium ${
                                                        j.status === "completed" ? "bg-emerald-50 text-emerald-700" :
                                                        j.status === "failed"    ? "bg-red-50 text-red-700" :
                                                        j.status === "running"   ? "bg-blue-50 text-blue-700" :
                                                                                   "bg-slate-100 text-slate-600"
                                                    }`}>
                                                        {j.status}
                                                    </span>
                                                </td>
                                                <td className="px-3 py-2 text-slate-500 text-xs">{fmtBytes(j.estimatedBytes)}</td>
                                                <td className="px-3 py-2 text-slate-500 text-xs">{j.targetType ?? "local"}</td>
                                                <td className="px-3 py-2">
                                                    {j.artifactRef ? (
                                                        <a
                                                            href={j.artifactRef}
                                                            target="_blank"
                                                            rel="noreferrer"
                                                            className="text-xs font-semibold text-teal-600 hover:text-teal-700"
                                                        >
                                                            Open
                                                        </a>
                                                    ) : (
                                                        <button
                                                            onClick={() => void handleDownloadJob(j.id)}
                                                            disabled={downloadingJobId === j.id}
                                                            className="text-xs font-semibold text-teal-600 hover:text-teal-700 disabled:opacity-50"
                                                        >
                                                            {downloadingJobId === j.id ? "..." : "Download"}
                                                        </button>
                                                    )}
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        )}
                    </div>
                </div>
            </Modal>
        </div>
    );
}
