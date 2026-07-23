import { ChevronDown } from "lucide-react";

type SelectInputProps = {
    value: string;
    onChange: (value: string) => void;
    options: { label: string; value: string; disabled?: boolean }[];
};

export function SelectInput({ value, onChange, options }: SelectInputProps) {
    return (
        <div className="relative">
            <select
                value={value}
                onChange={(e) => onChange(e.target.value)}
                className="w-full appearance-none rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300 focus:outline-none"
            >
                {options.map((opt) => (
                    <option key={opt.value} value={opt.value} disabled={opt.disabled}>
                        {opt.label}
                    </option>
                ))}
            </select>
            <ChevronDown className="h-4 w-4 text-slate-400 absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none" />
        </div>
    );
}
