package service

import (
	"context"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type auditLogService struct {
	repo repository.AuditLogRepository
}

func NewAuditLogService(repo repository.AuditLogRepository) interfaces.AuditLogService {
	return &auditLogService{repo: repo}
}

func (s *auditLogService) RecordLog(ctx context.Context, log *types.AuditLog) error {
	return s.repo.Create(ctx, log)
}

func (s *auditLogService) ListAuditLogs(ctx context.Context, filter interfaces.AuditLogFilter) ([]*types.AuditLog, int64, error) {
	return s.repo.List(ctx, filter.TenantID, filter.Page, filter.PageSize)
}
