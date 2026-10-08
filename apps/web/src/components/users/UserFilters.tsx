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
            <div className="flex flex-col items-stretch gap-4 lg:flex-row lg:items-end">
                <div className="grid flex-1 grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
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
                        className="h-10 w-full rounded-lg bg-teal-600 px-4 text-sm font-semibold text-white transition hover:bg-teal-700 lg:w-auto"
                    >
                        Add User
                    </button>
                )}
            </div>
        </Card>
    );
}
