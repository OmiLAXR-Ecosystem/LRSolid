import { useUsers } from "../hooks/useUsers";
import { UserFilters } from "../components/users/UserFilters";
import { UserTable } from "../components/users/UserTable";

export function UsersView() {
    const users = useUsers();

    return (
        <div className="space-y-6">
            <UserFilters
                name={users.name}
                email={users.email}
                role={users.role}
                onNameChange={users.setName}
                onEmailChange={users.setEmail}
                onRoleChange={users.setRole}
            />
            <UserTable
                items={users.items}
                loading={users.loading}
                error={users.error}
                limit={users.limit}
                onLimitChange={users.setLimit}
                page={users.page}
                totalPages={users.totalPages}
                pageStart={users.pageStart}
                pageEnd={users.pageEnd}
                total={users.total}
                onPrev={() => users.setPage((p) => Math.max(0, p - 1))}
                onNext={() => users.setPage((p) => Math.min(users.totalPages - 1, p + 1))}
            />
        </div>
    );
}
