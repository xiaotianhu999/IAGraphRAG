package types

import "time"

// DashboardStats represents the system-wide statistics for the dashboard
type DashboardStats struct {
	TotalTenants        int64 `json:"total_tenants"`
	TotalUsers          int64 `json:"total_users"`
	TotalKnowledgeBases int64 `json:"total_knowledge_bases"`
	TotalDocuments      int64 `json:"total_documents"`
	TotalStorageUsed    int64 `json:"total_storage_used"`
	TotalStorageQuota   int64 `json:"total_storage_quota"`
}

// TenantStats represents the statistics for a specific tenant
type TenantStats struct {
	TotalUsers          int64 `json:"total_users"`
	TotalKnowledgeBases int64 `json:"total_knowledge_bases"`
	TotalDocuments      int64 `json:"total_documents"`
	StorageUsed         int64 `json:"storage_used"`
	StorageQuota        int64 `json:"storage_quota"`
}

// DashboardData represents the full data for the dashboard
type DashboardData struct {
	Stats           interface{} `json:"stats"` // Can be DashboardStats or TenantStats
	RecentAuditLogs []*AuditLog `json:"recent_audit_logs"`
	SystemInfo      SystemInfo  `json:"system_info"`
}

// SystemInfo represents basic system information
type SystemInfo struct {
	Version   string    `json:"version"`
	GoVersion string    `json:"go_version"`
	StartTime time.Time `json:"start_time"`
	Uptime    string    `json:"uptime"`
}
