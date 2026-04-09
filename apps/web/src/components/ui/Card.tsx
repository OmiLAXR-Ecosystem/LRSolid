import type { ReactNode } from "react";

type CardProps = {
    title?: string;
    className?: string;
    bodyClassName?: string;
    children: ReactNode;
};

export function Card({ title, className = "", bodyClassName = "", children }: CardProps) {
    return (
        <div className={`rounded-2xl bg-white shadow-sm ring-1 ring-slate-200 ${className}`}>
            {title && (
                <div className="px-5 py-3 border-b border-slate-100 text-sm font-semibold text-slate-700">
                    {title}
                </div>
            )}
            <div className={title ? `px-5 py-4 ${bodyClassName}` : `p-5 ${bodyClassName}`}>
                {children}
            </div>
        </div>
    );
}
