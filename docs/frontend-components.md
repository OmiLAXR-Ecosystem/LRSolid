# Frontend Component Reference

Stack: React 19, TypeScript, Tailwind CSS, Vite, Lucide Icons

---

## Architektur-Übersicht

```
src/
  api/                  ← HTTP-Calls (typed, kein fetch() in Komponenten)
    client.ts           ← apiUrl(), apiFetch<T>()
    statements.ts       ← fetchStatementList, fetchStatementById, voidStatement, downloadStatementJson
    users.ts            ← fetchUsers
  types/                ← Shared TypeScript Types
    statement.ts        ← StatementRow, StatementDetail, ConnectionResponse
    user.ts             ← RestUser, RestUserResponse
  hooks/                ← Data-Fetching-Logik (State + Side Effects)
    useStatements.ts    ← Filters, Pagination, Detail-Modal-State
    useUsers.ts         ← Filters, Offset-Pagination
  components/
    ui/                 ← Primitive, wiederverwendbare UI-Bausteine
    statements/         ← Feature-Komponenten für Statements
    users/              ← Feature-Komponenten für Users
    layout/             ← App-Shell
  views/                ← Thin container — nur Hook + Feature-Komponenten
```

---

## UI-Basiskomponenten (`components/ui/`)

### `Card`
Universelle Karten-Wrapper-Komponente.

```tsx
<Card title="Optional Title" className="..." bodyClassName="...">
    {children}
</Card>
```

| Prop | Typ | Beschreibung |
|---|---|---|
| `title` | `string?` | Optionaler Titel mit Border-Divider |
| `className` | `string?` | Zusätzliche CSS-Klassen für das äußere div |
| `bodyClassName` | `string?` | Zusätzliche CSS-Klassen für den Content-Bereich |
| `children` | `ReactNode` | Inhalt |

---

### `FieldLabel`
Label für Formularfelder (konsistente Typografie).

```tsx
<FieldLabel>Actor:</FieldLabel>
```

---

### `TextInput`
Text-Input mit optionalem Icon.

```tsx
<TextInput
    value={value}
    onChange={(v) => setValue(v)}
    placeholder="Search..."
    icon={<Calendar className="h-4 w-4" />}
/>
```

| Prop | Typ | Beschreibung |
|---|---|---|
| `value` | `string` | Kontrollierter Wert |
| `onChange` | `(v: string) => void` | Change-Handler |
| `placeholder` | `string?` | Placeholder-Text |
| `type` | `string?` | Input-Typ (default: `"text"`) |
| `icon` | `ReactNode?` | Icon rechts im Input |

---

### `SelectInput`
Native `<select>` mit Chevron-Icon.

```tsx
<SelectInput
    value={role}
    onChange={setRole}
    options={[{ label: "All", value: "All" }, { label: "Admin", value: "admin" }]}
/>
```

| Prop | Typ | Beschreibung |
|---|---|---|
| `value` | `string` | Ausgewählter Wert |
| `onChange` | `(v: string) => void` | Change-Handler |
| `options` | `{ label: string; value: string }[]` | Optionsliste |

---

### `Pagination`
Wiederverwendbare Pagination-Controls mit Limit-Selector.
Funktioniert für Cursor-basierte (Statements) und Offset-basierte (Users) Paginierung.

```tsx
<Pagination
    label="Page 2"
    limit={25}
    onLimitChange={setLimit}
    onPrev={goPrev}
    onNext={goNext}
    prevDisabled={pageIndex === 0}
    nextDisabled={!hasNextPage}
    // Optional für Offset-Paginierung:
    currentPage={3}
    totalPages={10}
/>
```

| Prop | Typ | Beschreibung |
|---|---|---|
| `label` | `string` | Text links (z.B. "Page 2" oder "Showing 1-25 of 100") |
| `limit` | `number` | Aktuelles Limit |
| `onLimitChange` | `(n: number) => void` | Limit-Änderung |
| `onPrev/onNext` | `() => void` | Navigation |
| `prevDisabled/nextDisabled` | `boolean` | Disable-State |
| `currentPage` | `number?` | Aktuelle Seite (zeigt Seitenanzeige) |
| `totalPages` | `number?` | Gesamtseitenanzahl |

---

### `Modal`
Generischer Modal-Dialog mit Header, Content und optionalem Footer.
ESC schließt den Modal automatisch.

```tsx
<Modal
    open={isOpen}
    onClose={() => setOpen(false)}
    title="Statement Details"
    footer={<div>Footer Content</div>}
>
    {children}
</Modal>
```

| Prop | Typ | Beschreibung |
|---|---|---|
| `open` | `boolean` | Sichtbarkeit |
| `onClose` | `() => void` | Schließen-Handler (auch ESC) |
| `title` | `string` | Modal-Titel |
| `footer` | `ReactNode?` | Footer-Inhalt (border-top, flex-row) |
| `maxWidth` | `string?` | Tailwind max-width Klasse (default: `"max-w-5xl"`) |
| `children` | `ReactNode` | Body-Inhalt |

---

## Statement-Komponenten (`components/statements/`)

### `StatementFilters`
Filter-Leiste für die Statements-Ansicht (Statement-ID, Actor, Verb, Object, Datum).

```tsx
<StatementFilters
    filters={stmts.filters}
    onChange={stmts.setFilter}
    onApply={stmts.applyFilters}
    loading={stmts.loading}
/>
```

Bindet sich direkt an den `useStatements`-Hook.

---

### `StatementTable`
Tabelle mit konfigurierbaren Spalten, Kontext-Menü (View/Export/Void) und Pagination.
Spaltenvisibilität und Kontext-Menü-State sind intern (UI-only).

```tsx
<StatementTable
    items={stmts.items}
    loading={stmts.loading}
    error={stmts.error}
    limit={stmts.filters.limit}
    onLimitChange={(v) => stmts.setFilter("limit", v)}
    onRefresh={stmts.refresh}
    onViewDetails={stmts.openDetail}
    onExport={handleExport}
    onVoid={stmts.voidDetail}
    canExport={canExport}
    canVoid={canVoid}
    pageIndex={stmts.pageIndex}
    hasNextPage={stmts.hasNextPage}
    onNext={stmts.goNext}
    onPrev={stmts.goPrev}
/>
```

---

### `StatementDetailModal`
Modal zur Anzeige eines Statements als Raw JSON und interaktivem Tree.

```tsx
<StatementDetailModal
    open={stmts.detailOpen}
    detail={stmts.detail}
    loading={stmts.detailLoading}
    error={stmts.detailError}
    canExport={canExport}
    canVoid={canVoid}
    onClose={stmts.closeDetail}
    onExport={stmts.exportDetail}
    onVoid={stmts.voidDetail}
/>
```

---

### `JSONTree`
Rekursiver Tree-Renderer für beliebige JSON-Strukturen (`<details>`/`<summary>`).

```tsx
<JSONTree data={anyJsonValue} depth={0} />
```

---

## User-Komponenten (`components/users/`)

### `UserFilters`
Filter-Leiste für die Users-Ansicht (Name, Email, Rolle) mit optionalem "Add User"-Button.

```tsx
<UserFilters
    name={users.name}
    email={users.email}
    role={users.role}
    onNameChange={users.setName}
    onEmailChange={users.setEmail}
    onRoleChange={users.setRole}
    onAddUser={() => {/* TODO */}}
/>
```

---

### `UserTable`
Tabelle mit Offset-Paginierung für die Users-Ansicht.

---

## Layout (`components/layout/`)

### `AdminShell`
Haupt-Layout mit Sidebar-Navigation, Org-Selector und Header-Bar.
Wird in `App.tsx` einmal gerendert — alle Views sind `children`.

```tsx
<AdminShell activeKey="statements" searchPlaceholder="Search statements...">
    <StatementsView />
</AdminShell>
```

| Prop | Typ | Beschreibung |
|---|---|---|
| `activeKey` | `NavKey` | Aktiver Nav-Eintrag (hebt ihn in der Sidebar hervor) |
| `searchPlaceholder` | `string?` | Wenn gesetzt, zeigt die Header-Suchleiste |
| `children` | `ReactNode` | View-Inhalt |

`NavKey` = `"dashboard" | "statements" | "users" | "clients" | "reports" | "datasources" | "marketplace" | "settings"`

---

## Hooks

### `useStatements`
Verwaltet den gesamten State der Statements-Ansicht:
- Filter-State + Actions (`setFilter`, `applyFilters`, `refresh`)
- Cursor-Paginierung (`goNext`, `goPrev`, `pageIndex`, `hasNextPage`)
- Detail-Modal-State (`openDetail`, `closeDetail`, `exportDetail`, `voidDetail`)

### `useUsers`
Verwaltet den State der Users-Ansicht:
- Filter-State (`name`, `email`, `role` + Setter)
- Offset-Paginierung (`page`, `setPage`, `totalPages`, `pageStart`, `pageEnd`)
- Automatisches Re-Fetching bei Filter-/Limit-Änderungen

---

## API-Layer (`api/`)

### `api/client.ts`
- `apiUrl(path)` — baut URL mit `VITE_API_BASE` prefix
- `apiFetch<T>(path, init?)` — typisiertes fetch mit Fehlerbehandlung

### `api/statements.ts`
- `fetchStatementList(query, signal?)` → `ConnectionResponse`
- `fetchStatementById(id, signal?)` → `StatementDetail`
- `voidStatement(id)` — POST xAPI void statement
- `downloadStatementJson(detail)` — Browser-Download als `.json`

### `api/users.ts`
- `fetchUsers(query)` → `RestUserResponse`

---

## Environment Variables

| Variable | Default | Beschreibung |
|---|---|---|
| `VITE_API_BASE` | `""` | API-Base-URL (leer = gleicher Origin) |
| `VITE_IS_ADMIN` | `"true"` | Zeigt Admin-Panel-Button in Sidebar |
| `VITE_CAN_EXPORT` | `"true"` | Aktiviert Export-JSON-Aktion |
| `VITE_CAN_VOID` | `"true"` | Aktiviert Void-Statement-Aktion |
