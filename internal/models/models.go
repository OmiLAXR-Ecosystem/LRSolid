package models

import "time"

type Organisation struct {
	ID        string    `json:"id"` // uuid as string
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Project struct {
	ID          string    `json:"id"` // uuid
	OrgID       string    `json:"orgId"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type LRS struct {
	ID        string    `json:"id"` // tenant_id (text)
	ProjectID string    `json:"projectId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type User struct {
	ID           string     `json:"id"` // uuid
	Email        string     `json:"email"`
	DisplayName  *string    `json:"displayName,omitempty"`
	PasswordHash *string    `json:"-"`
	AuthProvider *string    `json:"authProvider,omitempty"`
	AuthSubject  *string    `json:"authSubject,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	DisabledAt   *time.Time `json:"disabledAt,omitempty"`
}

type OrgMembership struct {
	OrgID     string    `json:"orgId"`
	UserID    string    `json:"userId"`
	Role      string    `json:"role"` // owner/admin/member/viewer
	CreatedAt time.Time `json:"createdAt"`
}

type ProjectMembership struct {
	ProjectID string    `json:"projectId"`
	UserID    string    `json:"userId"`
	Role      string    `json:"role"` // owner/admin/member/viewer
	CreatedAt time.Time `json:"createdAt"`
}

type LRSMembership struct {
	LRSID     string    `json:"lrsId"`
	UserID    string    `json:"userId"`
	Role      string    `json:"role"` // admin/writer/reader
	CreatedAt time.Time `json:"createdAt"`
}

type Client struct {
	ID         string     `json:"id"` // uuid
	ProjectID  string     `json:"projectId"`
	LRSID      *string    `json:"lrsId,omitempty"`
	Name       string     `json:"name"`
	KeyID      string     `json:"keyId"`
	SecretHash string     `json:"-"`
	Scopes     any        `json:"scopes"` // map[string]bool or json.RawMessage
	CreatedAt  time.Time  `json:"createdAt"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}
