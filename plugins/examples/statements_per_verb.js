// Plugin: statements_per_verb
// Zeigt die Top-N Verben nach Statement-Anzahl.
//
// Server-side: läuft via goja im Go-Prozess.
// Zugriff auf DB via db.query(sql, ...args).
//
// UI-Seite: Das Frontend-Widget ist in
//   apps/web/src/plugins/builtin/ als TSX-Datei definiert.
// Für externe UI-Plugins muss ein separates ui.js bereitgestellt werden
// (siehe Dokumentation).

function schema() {
  return [
    { key: "limit", label: "Top N Verben", type: "int", default: 5, min: 1, max: 20 },
  ];
}

function fetchData(orgID, config) {
  var limit = (config && config.limit) ? Number(config.limit) : 5;

  var rows = db.query(
    "SELECT st.verb_id, COUNT(*) AS cnt " +
    "FROM statements st " +
    "JOIN lrs l ON l.id = st.tenant_id " +
    "JOIN projects p ON p.id = l.project_id " +
    "WHERE p.org_id = $1 AND st.verb_id IS NOT NULL " +
    "GROUP BY st.verb_id " +
    "ORDER BY cnt DESC " +
    "LIMIT $2",
    orgID,
    limit
  );

  return (rows || []).map(function(r) {
    var parts = String(r.verb_id).split("/");
    return {
      verb: parts[parts.length - 1] || r.verb_id,
      verbId: r.verb_id,
      count: Number(r.cnt),
    };
  });
}
