import { FieldLabel } from "../ui/FieldLabel";
import { SelectInput } from "../ui/SelectInput";
import { TextInput } from "../ui/TextInput";
import type {
    ExportFilterState,
    ExtensionCondition,
    ExtensionOperator,
    FieldCondition,
    TextOperator,
    TimestampOperator,
} from "../../types/export";
import type { RestProject, RestLRS } from "../../types/lrs";
import type { RestClient } from "../../types/client";

const TEXT_OPERATORS: { label: string; value: TextOperator }[] = [
    { label: "equals", value: "eq" },
    { label: "not equals", value: "neq" },
    { label: "contains", value: "contains" },
];

const TIMESTAMP_OPERATORS: { label: string; value: TimestampOperator }[] = [
    { label: "before (<)", value: "lt" },
    { label: "before or at (≤)", value: "lte" },
    { label: "equals (=)", value: "eq" },
    { label: "after or at (≥)", value: "gte" },
    { label: "after (>)", value: "gt" },
    { label: "between", value: "between" },
];

const EXTENSION_OPERATORS: { label: string; value: ExtensionOperator }[] = [
    { label: "equals", value: "eq" },
    { label: "not equals", value: "neq" },
    { label: "contains", value: "contains" },
    { label: "< (less than)", value: "lt" },
    { label: "≤ (less or equal)", value: "lte" },
    { label: "≥ (greater or equal)", value: "gte" },
    { label: "> (greater than)", value: "gt" },
];

type Props = {
    value: ExportFilterState;
    onChange: (v: ExportFilterState) => void;
    projects: RestProject[];
    lrsItems: RestLRS[];
    clients: RestClient[];
    thresholdGB?: number | null;
    loading?: boolean;
};

// ---- helper: inline operator + value row ----

type TextConditionRowProps = {
    label: string;
    condition: FieldCondition<TextOperator> | null;
    placeholder?: string;
    onEnable: () => void;
    onDisable: () => void;
    onChange: (c: FieldCondition<TextOperator>) => void;
};

function TextConditionRow({ label, condition, placeholder, onEnable, onDisable, onChange }: TextConditionRowProps) {
    return (
        <div className="space-y-1">
            <div className="flex items-center gap-2">
                <input
                    type="checkbox"
                    checked={condition !== null}
                    onChange={(e) => (e.target.checked ? onEnable() : onDisable())}
                    className="accent-teal-600"
                    id={`chk-${label}`}
                />
                <label htmlFor={`chk-${label}`} className="text-xs font-medium text-slate-600 select-none cursor-pointer">
                    {label}
                </label>
            </div>
            {condition !== null && (
                <div className="flex gap-2 pl-5">
                    <div className="w-36 shrink-0">
                        <SelectInput
                            value={condition.operator}
                            onChange={(op) => onChange({ ...condition, operator: op as TextOperator })}
                            options={TEXT_OPERATORS}
                        />
                    </div>
                    <div className="flex-1">
                        <TextInput
                            value={condition.value}
                            onChange={(v) => onChange({ ...condition, value: v })}
                            placeholder={placeholder}
                        />
                    </div>
                </div>
            )}
        </div>
    );
}

type TimestampConditionRowProps = {
    condition: FieldCondition<TimestampOperator> | null;
    onEnable: () => void;
    onDisable: () => void;
    onChange: (c: FieldCondition<TimestampOperator>) => void;
};

function TimestampConditionRow({ condition, onEnable, onDisable, onChange }: TimestampConditionRowProps) {
    return (
        <div className="space-y-1">
            <div className="flex items-center gap-2">
                <input
                    type="checkbox"
                    checked={condition !== null}
                    onChange={(e) => (e.target.checked ? onEnable() : onDisable())}
                    className="accent-teal-600"
                    id="chk-stored-at"
                />
                <label htmlFor="chk-stored-at" className="text-xs font-medium text-slate-600 select-none cursor-pointer">
                    Stored at
                </label>
            </div>
            {condition !== null && (
                <div className="flex gap-2 pl-5 flex-wrap">
                    <div className="w-44 shrink-0">
                        <SelectInput
                            value={condition.operator}
                            onChange={(op) =>
                                onChange({
                                    operator: op as TimestampOperator,
                                    value: condition.value,
                                    valueTo: op === "between" ? condition.valueTo ?? "" : undefined,
                                })
                            }
                            options={TIMESTAMP_OPERATORS}
                        />
                    </div>
                    <div className="flex-1 min-w-[160px]">
                        <input
                            type="datetime-local"
                            value={condition.value}
                            onChange={(e) => onChange({ ...condition, value: e.target.value })}
                            className="w-full rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300 focus:outline-none"
                        />
                    </div>
                    {condition.operator === "between" && (
                        <div className="flex-1 min-w-[160px]">
                            <input
                                type="datetime-local"
                                value={condition.valueTo ?? ""}
                                onChange={(e) => onChange({ ...condition, valueTo: e.target.value })}
                                className="w-full rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300 focus:outline-none"
                            />
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}

// ---- main component ----

export function ExportFilterForm({ value, onChange, projects, lrsItems, clients, thresholdGB, loading }: Props) {
    const set = <K extends keyof ExportFilterState>(key: K, val: ExportFilterState[K]) =>
        onChange({ ...value, [key]: val });

    const addExtension = () =>
        set("extensions", [...value.extensions, { key: "", operator: "eq", value: "" }]);

    const updateExtension = (i: number, ext: ExtensionCondition) => {
        const next = [...value.extensions];
        next[i] = ext;
        set("extensions", next);
    };

    const removeExtension = (i: number) =>
        set("extensions", value.extensions.filter((_, idx) => idx !== i));

    return (
        <div className="space-y-5">
            {loading && <div className="text-xs text-slate-400">Loading options...</div>}

            {/* Scope */}
            <div>
                <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Scope</div>
                <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
                    <div className="space-y-1">
                        <FieldLabel>Project</FieldLabel>
                        <SelectInput
                            value={value.projectId}
                            onChange={(v) => set("projectId", v)}
                            options={[
                                { label: "All Projects", value: "" },
                                ...projects.map((p) => ({ label: p.name, value: p.id })),
                            ]}
                        />
                    </div>
                    <div className="space-y-1">
                        <FieldLabel>LRS</FieldLabel>
                        <SelectInput
                            value={value.lrsId}
                            onChange={(v) => set("lrsId", v)}
                            options={[
                                { label: "All LRS", value: "" },
                                ...lrsItems.map((l) => ({ label: l.name, value: l.id })),
                            ]}
                        />
                    </div>
                    <div className="space-y-1">
                        <FieldLabel>Client</FieldLabel>
                        <SelectInput
                            value={value.clientId}
                            onChange={(v) => set("clientId", v)}
                            options={[
                                { label: "All Clients", value: "" },
                                ...clients.map((c) => ({ label: c.name, value: c.id })),
                            ]}
                        />
                    </div>
                </div>
            </div>

            {/* Statement filters */}
            <div>
                <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Statement Filters</div>
                <div className="space-y-3">
                    <TextConditionRow
                        label="Actor (mbox)"
                        condition={value.actorMbox}
                        placeholder="mailto:user@example.com"
                        onEnable={() => set("actorMbox", { operator: "eq", value: "" })}
                        onDisable={() => set("actorMbox", null)}
                        onChange={(c) => set("actorMbox", c)}
                    />
                    <TextConditionRow
                        label="Verb"
                        condition={value.verbId}
                        placeholder="http://adlnet.gov/expapi/verbs/completed"
                        onEnable={() => set("verbId", { operator: "eq", value: "" })}
                        onDisable={() => set("verbId", null)}
                        onChange={(c) => set("verbId", c)}
                    />
                    <TextConditionRow
                        label="Activity (object)"
                        condition={value.objectId}
                        placeholder="http://example.com/activities/course-1"
                        onEnable={() => set("objectId", { operator: "eq", value: "" })}
                        onDisable={() => set("objectId", null)}
                        onChange={(c) => set("objectId", c)}
                    />
                    <TimestampConditionRow
                        condition={value.storedAt}
                        onEnable={() => set("storedAt", { operator: "gte", value: "" })}
                        onDisable={() => set("storedAt", null)}
                        onChange={(c) => set("storedAt", c)}
                    />
                </div>
            </div>

            {/* Extension filters */}
            <div>
                <div className="mb-2 flex items-center justify-between">
                    <div className="text-xs font-semibold uppercase tracking-wide text-slate-400">Extension Filters</div>
                    <button
                        type="button"
                        onClick={addExtension}
                        className="text-xs font-semibold text-teal-600 hover:text-teal-700"
                    >
                        + Add
                    </button>
                </div>
                {value.extensions.length === 0 && (
                    <div className="text-xs text-slate-400">No extension filters.</div>
                )}
                <div className="space-y-2">
                    {value.extensions.map((ext, i) => (
                        <div key={i} className="flex gap-2 items-start">
                            <div className="flex-1 min-w-0">
                                <TextInput
                                    value={ext.key}
                                    onChange={(v) => updateExtension(i, { ...ext, key: v })}
                                    placeholder="Extension key IRI"
                                />
                            </div>
                            <div className="w-36 shrink-0">
                                <SelectInput
                                    value={ext.operator}
                                    onChange={(v) => updateExtension(i, { ...ext, operator: v as ExtensionOperator })}
                                    options={EXTENSION_OPERATORS}
                                />
                            </div>
                            <div className="flex-1 min-w-0">
                                <TextInput
                                    value={ext.value}
                                    onChange={(v) => updateExtension(i, { ...ext, value: v })}
                                    placeholder="Value"
                                />
                            </div>
                            <button
                                type="button"
                                onClick={() => removeExtension(i)}
                                className="mt-2 text-slate-400 hover:text-red-500 text-xs"
                            >
                                ✕
                            </button>
                        </div>
                    ))}
                </div>
            </div>

            {/* Export target */}
            <div>
                <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Export Target</div>
                <div className="space-y-2">
                    <div className="space-y-1">
                        <FieldLabel>Target Type</FieldLabel>
                        <SelectInput
                            value={value.targetType}
                            onChange={(v) => set("targetType", v)}
                            options={[
                                { label: "Local (no target)", value: "" },
                                { label: "S3", value: "s3" },
                                { label: "Google Drive", value: "gdrive" },
                                { label: "OneDrive", value: "onedrive" },
                                { label: "iCloud", value: "icloud" },
                            ]}
                        />
                    </div>
                    {value.targetType && (
                        <div className="space-y-1">
                            <FieldLabel>Target Path / Folder</FieldLabel>
                            <TextInput
                                value={value.targetPath}
                                onChange={(v) => set("targetPath", v)}
                                placeholder="e.g. exports/my-job.ndjson.gz or provider folder path"
                            />
                        </div>
                    )}
                    {thresholdGB !== null && thresholdGB > 0 && (
                        <div className="text-xs text-slate-400">
                            Remote target required for exports above {thresholdGB} GB (org setting).
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
