package types

import (
	"time"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID         uint64    `json:"id" gorm:"primaryKey"`
	UserID     string    `json:"user_id" gorm:"index"`
	Username   string    `json:"username"`
	TenantID   uint64    `json:"tenant_id" gorm:"index"`
	Action     string    `json:"action" gorm:"index"` // e.g., "login", "create_user", "delete_tenant"
	Resource   string    `json:"resource"`            // e.g., "user", "tenant", "knowledge_base"
	ResourceID string    `json:"resource_id"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	Status     string    `json:"status"` // "success" or "failure"
	Details    string    `json:"details" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at" gorm:"index"`
}
