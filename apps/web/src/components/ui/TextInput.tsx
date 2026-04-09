import type { ReactNode } from "react";

type TextInputProps = {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
    type?: string;
    icon?: ReactNode;
};

export function TextInput({ value, onChange, placeholder, type = "text", icon }: TextInputProps) {
    return (
        <div className="relative">
            <input
                type={type}
                value={value}
                onChange={(e) => onChange(e.target.value)}
                placeholder={placeholder}
                className="w-full rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300 focus:outline-none"
            />
            {icon && (
                <span className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-slate-400">
                    {icon}
                </span>
            )}
        </div>
    );
}
