import { useState } from "react";
import { Modal } from "../ui/Modal";
import { FieldLabel } from "../ui/FieldLabel";
import { TextInput } from "../ui/TextInput";
import { SelectInput } from "../ui/SelectInput";
import { inviteUser } from "../../api/users";
import { ApiError } from "../../api/client";

const ROLE_OPTIONS = [
    { label: "Viewer", value: "viewer" },
    { label: "Member", value: "member" },
    { label: "Admin", value: "admin" },
    { label: "Owner", value: "owner" },
];

type Props = {
    open: boolean;
    org: string;
    onClose: () => void;
    onSuccess: () => void;
};

export function InviteUserDialog({ open, org, onClose, onSuccess }: Props) {
    const [email, setEmail] = useState("");
    const [displayName, setDisplayName] = useState("");
    const [role, setRole] = useState("member");
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    function reset() {
        setEmail("");
        setDisplayName("");
        setRole("member");
        setError(null);
    }

    function handleClose() {
        reset();
        onClose();
    }

    async function handleSubmit() {
        setError(null);
        if (!email.trim()) {
            setError("Email ist erforderlich.");
            return;
        }
        setLoading(true);
        try {
            await inviteUser(org, {
                email: email.trim(),
                displayName: displayName.trim() || undefined,
                role,
            });
            reset();
            onSuccess();
        } catch (e) {
            setError(e instanceof ApiError ? e.message : "Einladung fehlgeschlagen.");
        } finally {
            setLoading(false);
        }
    }

    return (
        <Modal
            open={open}
            onClose={handleClose}
            title="Benutzer einladen"
            maxWidth="max-w-md"
            footer={
                <>
                    <button
                        onClick={handleClose}
                        className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-50"
                        disabled={loading}
                    >
                        Abbrechen
                    </button>
                    <button
                        onClick={handleSubmit}
                        disabled={loading}
                        className="rounded-lg bg-teal-600 px-4 py-2 text-sm font-semibold text-white hover:bg-teal-700 disabled:opacity-50"
                    >
                        {loading ? "Sende..." : "Einladung senden"}
                    </button>
                </>
            }
        >
            <div className="space-y-4">
                {error && <div className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{error}</div>}
                <div>
                    <FieldLabel>E-Mail *</FieldLabel>
                    <TextInput
                        value={email}
                        onChange={setEmail}
                        placeholder="user@example.com"
                        type="email"
                    />
                </div>
                <div>
                    <FieldLabel>Name (optional)</FieldLabel>
                    <TextInput
                        value={displayName}
                        onChange={setDisplayName}
                        placeholder="Max Mustermann"
                    />
                </div>
                <div>
                    <FieldLabel>Rolle</FieldLabel>
                    <SelectInput value={role} onChange={setRole} options={ROLE_OPTIONS} />
                </div>
            </div>
        </Modal>
    );
}
