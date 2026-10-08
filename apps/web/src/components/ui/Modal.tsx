import { useEffect, type ReactNode } from "react";

type ModalProps = {
    open: boolean;
    onClose: () => void;
    title: string;
    footer?: ReactNode;
    children: ReactNode;
    maxWidth?: string;
    showCloseButton?: boolean;
};

export function Modal({ open, onClose, title, footer, children, maxWidth = "max-w-5xl", showCloseButton = true }: ModalProps) {
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
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 sm:items-center sm:p-4">
            <div className={`flex max-h-[100dvh] w-full flex-col overflow-hidden rounded-t-2xl bg-white shadow-lg ring-1 ring-slate-200 sm:max-h-[90vh] sm:w-[90vw] sm:rounded-2xl ${maxWidth}`}>
                <div className="flex shrink-0 items-center justify-between gap-3 border-b border-slate-200 px-4 py-3 sm:px-6 sm:py-4">
                    <div className="min-w-0 text-lg font-semibold text-slate-800">{title}</div>
                    {showCloseButton && (
                        <button className="shrink-0 text-xl leading-none text-slate-500 hover:text-slate-700" onClick={onClose}>
                            ✕
                        </button>
                    )}
                </div>
                <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4 sm:px-6">{children}</div>
                {footer && (
                    <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-t border-slate-200 px-4 py-3 sm:px-6 sm:py-4">
                        {footer}
                    </div>
                )}
            </div>
        </div>
    );
}
