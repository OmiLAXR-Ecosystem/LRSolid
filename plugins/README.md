# Plugins

Lege `.js`-Dateien in dieses Verzeichnis. Sie werden beim Start der Binary automatisch geladen.

## Plugin-Format

```js
// Pflichtfeld: id des Plugins (= Widget-ID im Dashboard)
// Wird aus dem Dateinamen abgeleitet (z.B. my_widget.js → id = "my_widget")

// Optional: Schema für konfigurierbare Felder
function schema() {
  return [
    { key: "limit", label: "Anzahl", type: "int", default: 5, min: 1, max: 50 },
  ];
}

// Pflicht: Daten liefern
// orgID: UUID der aktuellen Organisation
// config: { key: value } aus dem Dashboard-Config-Modal
function fetchData(orgID, config) {
  var rows = db.query(
    "SELECT COUNT(*) AS n FROM statements st " +
    "JOIN lrs l ON l.id = st.tenant_id " +
    "JOIN projects p ON p.id = l.project_id " +
    "WHERE p.org_id = $1",
    orgID
  );
  return { value: Number(rows[0].n) };
}
```

## Verfügbare Host-APIs

| API | Beschreibung |
|-----|-------------|
| `db.query(sql, ...args)` | SELECT-Abfragen auf der LRS-Datenbank |

## UI-Komponente registrieren

Das `.js`-Plugin liefert nur Daten. Die zugehörige React-Komponente muss im Frontend
registriert werden:

```ts
// apps/web/src/plugins/builtin/MyWidget.tsx
import { Card } from "../../components/Card";
import type { WidgetPlugin } from "../types";

type Data = { value: number };

export const MyWidget: WidgetPlugin<Data> = {
  id: "my_widget",       // muss mit dem Dateinamen übereinstimmen
  name: "Mein Widget",
  colSpan: "col-span-1",
  configSchema: [
    { key: "limit", label: "Anzahl", type: "int", default: 5 },
  ],
  component: ({ data }) => (
    <Card className="p-5">
      <div className="text-3xl font-extrabold">{data?.value ?? 0}</div>
    </Card>
  ),
};
```

```ts
// apps/web/src/plugins/builtin/index.ts — Plugin registrieren:
import { MyWidget } from "./MyWidget";
registerWidget(MyWidget);
```

## Docker

Das Verzeichnis wird automatisch als Volume gemountet:
```yaml
volumes:
  - ./plugins:/plugins
```
