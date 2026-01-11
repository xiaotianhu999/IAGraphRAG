package interfaces

import (
	"context"

	"github.com/aiplusall/aiplusall-kb/internal/types"
)

// DashboardService defines the interface for dashboard operations
type DashboardService interface {
	// GetSystemStats returns system-wide statistics (for super admin)
	GetSystemStats(ctx context.Context) (*types.DashboardStats, error)

	// GetTenantStats returns statistics for a specific tenant
	GetTenantStats(ctx context.Context, tenantID uint64) (*types.TenantStats, error)

	// GetDashboardData returns full dashboard data based on user context
	GetDashboardData(ctx context.Context) (*types.DashboardData, error)
}
