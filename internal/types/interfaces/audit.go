package interfaces

import (
	"context"

	"github.com/aiplusall/aiplusall-kb/internal/types"
)

type AuditLogFilter struct {
	TenantID uint64
	UserID   string
	Action   string
	Status   string
	Page     int
	PageSize int
}

type AuditLogService interface {
	RecordLog(ctx context.Context, log *types.AuditLog) error
	ListAuditLogs(ctx context.Context, filter AuditLogFilter) ([]*types.AuditLog, int64, error)
}
