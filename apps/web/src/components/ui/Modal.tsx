import { useEffect, type ReactNode } from "react";

type ModalProps = {
    open: boolean;
    onClose: () => void;
    title: string;
    footer?: ReactNode;
    children: ReactNode;
    maxWidth?: string;
};

export function Modal({ open, onClose, title, footer, children, maxWidth = "max-w-5xl" }: ModalProps) {
    useEffect(() => {
        if (!open) return;
        const handler = (e: KeyboardEvent) => {
            if (e.key === "Escape") onClose();
        };
        window.addEventListener("keydown", handler);
        return () => window.removeEventListener("keydown", handler);
    }, [open, onClose]);

    if (!open) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
            <div className={`w-[90vw] ${maxWidth} rounded-2xl bg-white shadow-lg ring-1 ring-slate-200`}>
                <div className="flex items-center justify-between border-b border-slate-200 px-6 py-4">
                    <div className="text-lg font-semibold text-slate-800">{title}</div>
                    <button className="text-slate-500 hover:text-slate-700 text-xl leading-none" onClick={onClose}>
                        ✕
                    </button>
                </div>
                <div className="px-6 py-4">{children}</div>
                {footer && (
                    <div className="flex items-center justify-between border-t border-slate-200 px-6 py-4">
                        {footer}
                    </div>
                )}
            </div>
        </div>
    );
}
