package repository

import (
	"context"

	"github.com/aiplusall/aiplusall-kb/internal/types"

	"gorm.io/gorm"
)

type AuditLogRepository interface {
	Create(ctx context.Context, log *types.AuditLog) error
	List(ctx context.Context, tenantID uint64, page, pageSize int) ([]*types.AuditLog, int64, error)
}

type auditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(ctx context.Context, log *types.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *auditLogRepository) List(ctx context.Context, tenantID uint64, page, pageSize int) ([]*types.AuditLog, int64, error) {
	var logs []*types.AuditLog
	var total int64

	db := r.db.WithContext(ctx)
	if tenantID == 0 {
		db = db.InstanceSet("skip_tenant_isolation", true)
	}

	query := db.Model(&types.AuditLog{})
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
