package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/config"
	"github.com/SGoerzen/LRSolid/internal/secrets"
	"github.com/SGoerzen/LRSolid/internal/store"
)

func RestListDataStores(cfg config.Config, db *pgxpool.Pool, stores *store.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		includeInactive := r.URL.Query().Get("includeInactive") == "true"
		if includeInactive {
			if _, ok := requireAdminClaims(r.Context()); !ok {
				writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
				return
			}
		}
		where := "WHERE is_active = true"
		if includeInactive {
			where = ""
		}
		rows, err := db.Query(r.Context(), `
			SELECT id, name, database_type, org_id, is_default, is_active,
			       (
			         (connection_string IS NOT NULL AND connection_string != '')
			         OR (secret_ref IS NOT NULL AND secret_ref != '')
			       ) AS has_connection_string,
			       description, created_at, updated_at
			FROM data_stores
			`+where+`
			ORDER BY is_default DESC, name ASC
		`)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list data stores failed"})
			return
		}
		defer rows.Close()

		items := make([]restDataStore, 0, 8)
		for rows.Next() {
			var ds restDataStore
			if err := rows.Scan(
				&ds.ID, &ds.Name, &ds.DatabaseType, &ds.OrgID,
				&ds.IsDefault, &ds.IsActive, &ds.HasConnectionString,
				&ds.Description, &ds.CreatedAt, &ds.UpdatedAt,
			); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "scan data stores failed"})
				return
			}
			ds.Status, ds.StatusError, ds.Capabilities = dataStoreStatus(stores, ds)
			items = append(items, ds)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read data stores failed"})
			return
		}

		writeJSON(w, http.StatusOK, restDataStoreListResponse{Items: items})
	}
}

func dataStoreStatus(stores *store.Registry, ds restDataStore) (string, string, store.Capabilities) {
	if !ds.IsActive {
		return "inactive", "", store.ExternalCapabilities()
	}
	if stores != nil {
		if bundle, err := stores.Get(ds.ID); err == nil {
			caps := bundle.Capabilities
			if !caps.Statements {
				caps = store.ExternalCapabilities()
			}
			return "connected", "", caps
		}
		if _, msg := stores.Status(ds.ID); msg != "" {
			return "error", msg, store.ExternalCapabilities()
		}
	}
	if ds.IsDefault {
		return "control_plane", "", store.ControlPlaneCapabilities()
	}
	if !ds.HasConnectionString {
		return "not_configured", "", store.ExternalCapabilities()
	}
	return "unavailable", "", store.ExternalCapabilities()
}

func AdminCreateDataStore(cfg config.Config, db *pgxpool.Pool, dataSecrets secrets.Store, stores *store.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireAdminClaims(r.Context()); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}
		var req createDataStoreRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.ConnectionString = strings.TrimSpace(req.ConnectionString)
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name required"})
			return
		}
		if err := validateDataStoreConnection(req.DatabaseType, req.ConnectionString); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		if dataSecrets == nil {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "data store credential storage is not configured"})
			return
		}
		if stores == nil {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "data store registry is not configured"})
			return
		}

		id := uuid.NewString()
		if err := stores.Register(r.Context(), store.DataStoreConfig{
			ID: id, Name: req.Name, DatabaseType: req.DatabaseType, ConnectionString: req.ConnectionString,
		}); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		secretRef, err := newDataStoreSecretRef()
		if err != nil {
			stores.Unregister(id)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "generate secret reference failed"})
			return
		}
		if err := dataSecrets.Put(r.Context(), secretRef, []byte(req.ConnectionString)); err != nil {
			stores.Unregister(id)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "store data store credentials failed"})
			return
		}

		host, port, dbName := connectionParts(req.ConnectionString)
		var ds restDataStore
		err = db.QueryRow(r.Context(), `
			INSERT INTO data_stores (id, name, database_type, host, port, database_name, connection_string, secret_ref, description, org_id)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, NULL, $7, $8, $9::uuid)
			RETURNING id, name, database_type, org_id, is_default, is_active,
			          (
			            (connection_string IS NOT NULL AND connection_string != '')
			            OR (secret_ref IS NOT NULL AND secret_ref != '')
			          ),
			          description, created_at, updated_at
		`, id, req.Name, req.DatabaseType, host, port, dbName, secretRef, req.Description, req.OrgID).Scan(
			&ds.ID, &ds.Name, &ds.DatabaseType, &ds.OrgID,
			&ds.IsDefault, &ds.IsActive, &ds.HasConnectionString,
			&ds.Description, &ds.CreatedAt, &ds.UpdatedAt,
		)
		if err != nil {
			stores.Unregister(id)
			_ = dataSecrets.Delete(r.Context(), secretRef)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create data store failed"})
			return
		}
		ds.Status, ds.StatusError, ds.Capabilities = dataStoreStatus(stores, ds)
		writeJSON(w, http.StatusCreated, ds)
	}
}

func AdminUpdateDataStore(cfg config.Config, db *pgxpool.Pool, dataSecrets secrets.Store, stores *store.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireAdminClaims(r.Context()); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}
		id := chi.URLParam(r, "id")

		var req updateDataStoreRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
			return
		}
		if req.ConnectionString != nil {
			trimmed := strings.TrimSpace(*req.ConnectionString)
			if trimmed == "" {
				req.ConnectionString = nil
			} else {
				req.ConnectionString = &trimmed
			}
		}

		var existingType string
		var existingName string
		var existingActive bool
		var existingSecret *string
		var isDefault bool
		err := db.QueryRow(r.Context(), `
			SELECT name, database_type, is_active, is_default, secret_ref
			FROM data_stores WHERE id = $1
		`, id).Scan(&existingName, &existingType, &existingActive, &isDefault, &existingSecret)
		if err != nil {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "data store not found"})
			return
		}
		if isDefault && req.ConnectionString != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "cannot replace the control-plane data store connection"})
			return
		}

		nextActive := existingActive
		if req.IsActive != nil {
			nextActive = *req.IsActive
		}
		conn := ""
		if req.ConnectionString != nil {
			conn = *req.ConnectionString
			if err := validateDataStoreConnection(existingType, conn); err != nil {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
				return
			}
		}

		if nextActive && !isDefault && stores != nil {
			if conn == "" && existingSecret != nil && strings.TrimSpace(*existingSecret) != "" {
				if dataSecrets == nil {
					writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "data store credential storage is not configured"})
					return
				}
				plaintext, err := dataSecrets.Get(r.Context(), strings.TrimSpace(*existingSecret))
				if err != nil {
					writeJSON(w, http.StatusBadRequest, errorResponse{Error: "load data store credentials failed"})
					return
				}
				conn = string(plaintext)
			}
			if conn != "" {
				name := existingName
				if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
					name = strings.TrimSpace(*req.Name)
				}
				if err := stores.Register(r.Context(), store.DataStoreConfig{
					ID: id, Name: name, DatabaseType: existingType, ConnectionString: conn,
				}); err != nil {
					writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
					return
				}
			}
		}

		var secretRef *string
		if req.ConnectionString != nil {
			if dataSecrets == nil {
				writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "data store credential storage is not configured"})
				return
			}
			ref := ""
			if existingSecret != nil {
				ref = strings.TrimSpace(*existingSecret)
			}
			if ref == "" {
				ref, err = newDataStoreSecretRef()
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "generate secret reference failed"})
					return
				}
			}
			if err := dataSecrets.Put(r.Context(), ref, []byte(*req.ConnectionString)); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "store data store credentials failed"})
				return
			}
			secretRef = &ref
		}

		var ds restDataStore
		err = db.QueryRow(r.Context(), `
			UPDATE data_stores SET
			  name              = COALESCE($2, name),
			  connection_string = CASE WHEN $3::text IS NULL THEN connection_string ELSE NULL END,
			  secret_ref        = COALESCE($3, secret_ref),
			  description       = COALESCE($4, description),
			  is_active         = COALESCE($5, is_active),
			  updated_at        = now()
			WHERE id = $1
			RETURNING id, name, database_type, org_id, is_default, is_active,
			          (
			            (connection_string IS NOT NULL AND connection_string != '')
			            OR (secret_ref IS NOT NULL AND secret_ref != '')
			          ),
			          description, created_at, updated_at
		`, id, req.Name, secretRef, req.Description, req.IsActive).Scan(
			&ds.ID, &ds.Name, &ds.DatabaseType, &ds.OrgID,
			&ds.IsDefault, &ds.IsActive, &ds.HasConnectionString,
			&ds.Description, &ds.CreatedAt, &ds.UpdatedAt,
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "update data store failed"})
			return
		}
		if !ds.IsActive && stores != nil {
			stores.Unregister(id)
		}
		ds.Status, ds.StatusError, ds.Capabilities = dataStoreStatus(stores, ds)
		writeJSON(w, http.StatusOK, ds)
	}
}

func AdminDeleteDataStore(cfg config.Config, db *pgxpool.Pool, dataSecrets secrets.Store, stores *store.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireAdminClaims(r.Context()); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}
		id := chi.URLParam(r, "id")

		var isDefault bool
		var secretRef *string
		err := db.QueryRow(r.Context(), `SELECT is_default, secret_ref FROM data_stores WHERE id = $1`, id).Scan(&isDefault, &secretRef)
		if err != nil {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "data store not found"})
			return
		}
		if isDefault {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "cannot delete the default data store"})
			return
		}
		var used int
		if err := db.QueryRow(r.Context(), `SELECT COUNT(*) FROM lrs WHERE data_store_id = $1`, id).Scan(&used); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "delete data store failed"})
			return
		}
		if used > 0 {
			writeJSON(w, http.StatusConflict, errorResponse{Error: "data store is still used by a learning record store"})
			return
		}

		_, err = db.Exec(r.Context(), `DELETE FROM data_stores WHERE id = $1 AND is_default = false`, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "delete data store failed"})
			return
		}
		if stores != nil {
			stores.Unregister(id)
		}
		if dataSecrets != nil && secretRef != nil && strings.TrimSpace(*secretRef) != "" {
			_ = dataSecrets.Delete(r.Context(), *secretRef)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted"})
	}
}

func AdminTestDataStore(cfg config.Config, db *pgxpool.Pool, dataSecrets secrets.Store, stores *store.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireAdminClaims(r.Context()); !ok {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "admin access required"})
			return
		}
		if stores == nil {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "data store registry is not configured"})
			return
		}

		id := strings.TrimSpace(chi.URLParam(r, "id"))
		var cfgStore store.DataStoreConfig
		if id == "" {
			var req createDataStoreRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
				return
			}
			req.ConnectionString = strings.TrimSpace(req.ConnectionString)
			if err := validateDataStoreConnection(req.DatabaseType, req.ConnectionString); err != nil {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
				return
			}
			cfgStore = store.DataStoreConfig{Name: req.Name, DatabaseType: req.DatabaseType, ConnectionString: req.ConnectionString}
		} else {
			var dbType, name string
			var secretRef, connectionString *string
			err := db.QueryRow(r.Context(), `
				SELECT name, database_type, connection_string, secret_ref
				FROM data_stores WHERE id = $1
			`, id).Scan(&name, &dbType, &connectionString, &secretRef)
			if err == pgx.ErrNoRows {
				writeJSON(w, http.StatusNotFound, errorResponse{Error: "data store not found"})
				return
			}
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "load data store failed"})
				return
			}
			conn := ""
			if secretRef != nil && strings.TrimSpace(*secretRef) != "" {
				if dataSecrets == nil {
					writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "data store credential storage is not configured"})
					return
				}
				plaintext, err := dataSecrets.Get(r.Context(), strings.TrimSpace(*secretRef))
				if err != nil {
					writeJSON(w, http.StatusBadRequest, errorResponse{Error: "load data store credentials failed"})
					return
				}
				conn = string(plaintext)
			} else if connectionString != nil {
				conn = *connectionString
			}
			if err := validateDataStoreConnection(dbType, conn); err != nil {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
				return
			}
			cfgStore = store.DataStoreConfig{ID: id, Name: name, DatabaseType: dbType, ConnectionString: conn}
		}

		if err := stores.Probe(r.Context(), cfgStore); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func validateDataStoreConnection(databaseType, connectionString string) error {
	if databaseType != "postgres" && databaseType != "mongodb" {
		return errString("databaseType must be postgres or mongodb")
	}
	if connectionString == "" {
		return errString("connectionString required")
	}
	u, err := url.Parse(connectionString)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return errString("connectionString must be a URI with a host")
	}
	switch databaseType {
	case "postgres":
		if u.Scheme != "postgres" && u.Scheme != "postgresql" {
			return errString("postgres connectionString must use the postgres scheme")
		}
	case "mongodb":
		if u.Scheme != "mongodb" && u.Scheme != "mongodb+srv" {
			return errString("mongodb connectionString must use the mongodb scheme")
		}
	}
	return nil
}

type errString string

func (e errString) Error() string { return string(e) }

func connectionParts(cs string) (host string, port *int, dbName *string) {
	u, err := url.Parse(cs)
	if err != nil {
		return "unknown", nil, nil
	}
	host = u.Hostname()
	if host == "" {
		host = "unknown"
	}
	if p := u.Port(); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			port = &n
		}
	}
	name := strings.TrimPrefix(u.Path, "/")
	if i := strings.IndexByte(name, '?'); i >= 0 {
		name = name[:i]
	}
	if name != "" {
		dbName = &name
	}
	return host, port, dbName
}
