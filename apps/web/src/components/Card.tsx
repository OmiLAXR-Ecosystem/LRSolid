import type { ReactNode } from "react";

type CardProps = {
    title?: string;
    className?: string;
    bodyClassName?: string;
    children: ReactNode;
};

export function Card({ title, className, bodyClassName, children }: CardProps) {
    return (
        <div className={["rounded-xl bg-white shadow-sm ring-1 ring-slate-200", className].filter(Boolean).join(" ")}>
            {title && (
                <div className="px-5 pt-4 pb-2 text-sm font-semibold text-slate-800">
                    {title}
                </div>
            )}
            <div className={bodyClassName ?? (title ? "px-5 pb-5" : "p-5")}>{children}</div>
        </div>
    );
}
