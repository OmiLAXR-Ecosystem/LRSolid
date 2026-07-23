// Example JS plugin: counts projects with at least one statement in the last 30 days.
// Plugin ID: "example_active_projects"  (derived from filename)
//
// Available globals:
//   db.query(sql, ...args) — synchronous, returns array of row objects
//   orgID                  — passed by fetchData(orgID, config)
//   config                 — per-widget config map (see schema())

function schema() {
    return [
        {
            key: "days",
            label: "Lookback days",
            type: "int",
            default: 30,
            min: 1,
            max: 365
        }
    ];
}

function fetchData(orgID, config) {
    var days = (config && config.days) ? config.days : 30;

    var rows = db.query(
        `SELECT COUNT(DISTINCT p.id)::int AS value
         FROM statements st
         JOIN lrs l ON l.id = st.tenant_id
         JOIN projects p ON p.id = l.project_id
         WHERE p.org_id = $1
           AND st.stored_at >= NOW() - ($2 || ' days')::interval`,
        orgID,
        String(days)
    );

    return { value: rows.length > 0 ? rows[0].value : 0 };
}
