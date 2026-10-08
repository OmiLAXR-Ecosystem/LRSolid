import { Calendar, ChevronDown } from "lucide-react";
import { Card } from "../ui/Card";
import { FieldLabel } from "../ui/FieldLabel";
import { TextInput } from "../ui/TextInput";
import type { StatementFilters as Filters } from "../../hooks/useStatements";

type StatementFiltersProps = {
    filters: Filters;
    onChange: <K extends keyof Filters>(field: K, value: Filters[K]) => void;
    onApply: () => void;
    loading: boolean;
};

export function StatementFilters({ filters, onChange, onApply, loading }: StatementFiltersProps) {
    return (
        <Card>
            <div className="grid grid-cols-1 items-end gap-3 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-7">
                <div>
                    <FieldLabel>Statement ID:</FieldLabel>
                    <TextInput
                        value={filters.statementId}
                        onChange={(v) => onChange("statementId", v)}
                        placeholder="Exact ID"
                        icon={<ChevronDown className="h-4 w-4" />}
                    />
                </div>
                <div>
                    <FieldLabel>Actor:</FieldLabel>
                    <TextInput
                        value={filters.actorMbox}
                        onChange={(v) => onChange("actorMbox", v)}
                        placeholder="All"
                        icon={<ChevronDown className="h-4 w-4" />}
                    />
                </div>
                <div>
                    <FieldLabel>Verb:</FieldLabel>
                    <TextInput
                        value={filters.verbId}
                        onChange={(v) => onChange("verbId", v)}
                        placeholder="All"
                        icon={<ChevronDown className="h-4 w-4" />}
                    />
                </div>
                <div>
                    <FieldLabel>Object:</FieldLabel>
                    <TextInput
                        value={filters.objectId}
                        onChange={(v) => onChange("objectId", v)}
                        placeholder="All"
                        icon={<ChevronDown className="h-4 w-4" />}
                    />
                </div>
                <div>
                    <FieldLabel>Date from:</FieldLabel>
                    <TextInput
                        value={filters.dateFrom}
                        onChange={(v) => onChange("dateFrom", v)}
                        placeholder="2022-01-01"
                        icon={<Calendar className="h-4 w-4" />}
                    />
                </div>
                <div>
                    <FieldLabel>Date to:</FieldLabel>
                    <TextInput
                        value={filters.dateTo}
                        onChange={(v) => onChange("dateTo", v)}
                        placeholder="2022-01-30"
                        icon={<Calendar className="h-4 w-4" />}
                    />
                </div>
                <button
                    onClick={onApply}
                    disabled={loading}
                    className="h-10 w-full rounded-lg bg-teal-600 px-4 text-sm font-semibold text-white transition hover:bg-teal-700 disabled:opacity-60 sm:w-auto"
                >
                    {loading ? "Loading..." : "Apply Filters"}
                </button>
            </div>
        </Card>
    );
}
