import { Card } from "../ui/Card";
import { FieldLabel } from "../ui/FieldLabel";
import { TextInput } from "../ui/TextInput";
import { SelectInput } from "../ui/SelectInput";

const ROLE_OPTIONS = [
    { label: "All", value: "All" },
    { label: "Admin", value: "Admin" },
    { label: "User", value: "User" },
];

type UserFiltersProps = {
    name: string;
    email: string;
    role: string;
    onNameChange: (v: string) => void;
    onEmailChange: (v: string) => void;
    onRoleChange: (v: string) => void;
    onAddUser?: () => void;
};

export function UserFilters({ name, email, role, onNameChange, onEmailChange, onRoleChange, onAddUser }: UserFiltersProps) {
    return (
        <Card>
            <div className="flex items-end justify-between gap-6">
                <div className="grid grid-cols-[1fr_1fr_0.8fr] gap-4 flex-1">
                    <div>
                        <FieldLabel>Name:</FieldLabel>
                        <TextInput value={name} onChange={onNameChange} placeholder="Name..." />
                    </div>
                    <div>
                        <FieldLabel>Email:</FieldLabel>
                        <TextInput value={email} onChange={onEmailChange} placeholder="Email..." />
                    </div>
                    <div>
                        <FieldLabel>Role:</FieldLabel>
                        <SelectInput value={role} onChange={onRoleChange} options={ROLE_OPTIONS} />
                    </div>
                </div>
                {onAddUser && (
                    <button
                        onClick={onAddUser}
                        className="h-10 rounded-lg bg-teal-600 px-4 text-sm font-semibold text-white hover:bg-teal-700 transition"
                    >
                        Add User
                    </button>
                )}
            </div>
        </Card>
    );
}
