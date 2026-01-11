package service

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/aiplusall/aiplusall-kb/internal/types"
	"github.com/aiplusall/aiplusall-kb/internal/types/interfaces"
	"gorm.io/gorm"
)

type dashboardService struct {
	db           *gorm.DB
	auditService interfaces.AuditLogService
	startTime    time.Time
	version      string
}

// NewDashboardService creates a new dashboard service
func NewDashboardService(db *gorm.DB, auditService interfaces.AuditLogService, version string) interfaces.DashboardService {
	return &dashboardService{
		db:           db,
		auditService: auditService,
		startTime:    time.Now(),
		version:      version,
	}
}

func (s *dashboardService) GetSystemStats(ctx context.Context) (*types.DashboardStats, error) {
	var stats types.DashboardStats

	// Use a raw DB session to skip tenant isolation for system-wide stats
	db := s.db.WithContext(ctx).InstanceSet("skip_tenant_isolation", true)

	if err := db.Model(&types.Tenant{}).Count(&stats.TotalTenants).Error; err != nil {
		return nil, err
	}

	if err := db.Model(&types.User{}).Count(&stats.TotalUsers).Error; err != nil {
		return nil, err
	}

	if err := db.Model(&types.KnowledgeBase{}).Count(&stats.TotalKnowledgeBases).Error; err != nil {
		return nil, err
	}

	// For documents, we might need to check if the table exists or use a generic query
	// Assuming KnowledgeBase has a way to count documents or we count from Document model
	// For now, let's just count KnowledgeBases as a placeholder if Document model is complex
	// Actually, let's try to count from 'documents' table if it exists
	var docCount int64
	if err := db.Table("documents").Count(&docCount).Error; err == nil {
		stats.TotalDocuments = docCount
	}

	// Storage stats from tenants
	var storageStats struct {
		TotalUsed  int64
		TotalQuota int64
	}
	if err := db.Model(&types.Tenant{}).Select("SUM(storage_used) as total_used, SUM(storage_quota) as total_quota").Scan(&storageStats).Error; err != nil {
		return nil, err
	}
	stats.TotalStorageUsed = storageStats.TotalUsed
	stats.TotalStorageQuota = storageStats.TotalQuota

	return &stats, nil
}

func (s *dashboardService) GetTenantStats(ctx context.Context, tenantID uint64) (*types.TenantStats, error) {
	var stats types.TenantStats

	// Skip tenant isolation to allow cross-tenant stats retrieval (e.g. by super admin)
	db := s.db.WithContext(ctx).InstanceSet("skip_tenant_isolation", true)

	if err := db.Model(&types.User{}).Where("tenant_id = ?", tenantID).Count(&stats.TotalUsers).Error; err != nil {
		return nil, err
	}

	if err := db.Model(&types.KnowledgeBase{}).Where("tenant_id = ?", tenantID).Count(&stats.TotalKnowledgeBases).Error; err != nil {
		return nil, err
	}

	// Count documents for this tenant
	var docCount int64
	if err := db.Table("documents").Where("tenant_id = ?", tenantID).Count(&docCount).Error; err == nil {
		stats.TotalDocuments = docCount
	}

	var tenant types.Tenant
	if err := db.Where("id = ?", tenantID).First(&tenant).Error; err == nil {
		stats.StorageUsed = tenant.StorageUsed
		stats.StorageQuota = tenant.StorageQuota
	}

	return &stats, nil
}

func (s *dashboardService) GetDashboardData(ctx context.Context) (*types.DashboardData, error) {
	data := &types.DashboardData{
		SystemInfo: types.SystemInfo{
			Version:   s.version,
			GoVersion: runtime.Version(),
			StartTime: s.startTime,
			Uptime:    time.Since(s.startTime).String(),
		},
	}

	// Get user from context to determine if they are super admin
	user, ok := ctx.Value("user").(*types.User)
	if !ok {
		return nil, fmt.Errorf("user not found in context")
	}

	var err error
	if user.CanAccessAllTenants {
		data.Stats, err = s.GetSystemStats(ctx)
	} else {
		data.Stats, err = s.GetTenantStats(ctx, user.TenantID)
	}

	if err != nil {
		return nil, err
	}

	// Get recent audit logs
	logs, _, err := s.auditService.ListAuditLogs(ctx, interfaces.AuditLogFilter{
		Page:     1,
		PageSize: 10,
	})
	if err == nil {
		data.RecentAuditLogs = logs
	}

	return data, nil
}
