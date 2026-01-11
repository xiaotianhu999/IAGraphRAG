package service

import (
	"context"
	"fmt"

	"github.com/aiplusall/aiplusall-kb/internal/types"
	"github.com/aiplusall/aiplusall-kb/internal/types/interfaces"
	"gorm.io/gorm"
)

type systemInitializationService struct {
	db            *gorm.DB
	tenantService interfaces.TenantService
	userService   interfaces.UserService
}

// NewSystemInitializationService creates a new system initialization service
func NewSystemInitializationService(
	db *gorm.DB,
	tenantService interfaces.TenantService,
	userService interfaces.UserService,
) interfaces.SystemInitializationService {
	return &systemInitializationService{
		db:            db,
		tenantService: tenantService,
		userService:   userService,
	}
}

func (s *systemInitializationService) IsInitialized(ctx context.Context) (bool, error) {
	var count int64
	// Check if there are any users in the system
	// Use skip_tenant_isolation to check globally
	if err := s.db.WithContext(ctx).InstanceSet("skip_tenant_isolation", true).Model(&types.User{}).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *systemInitializationService) Initialize(ctx context.Context, req types.SystemInitRequest) error {
	// 1. Check if already initialized
	initialized, err := s.IsInitialized(ctx)
	if err != nil {
		return err
	}
	if initialized {
		return fmt.Errorf("system is already initialized")
	}

	// 2. Create the first tenant
	tenant := &types.Tenant{
		Name:        req.TenantName,
		Description: "Default System Tenant",
		Status:      "active",
	}
	_, err = s.tenantService.CreateTenant(ctx, tenant)
	if err != nil {
		return fmt.Errorf("failed to create initial tenant: %w", err)
	}

	// 3. Create the super admin user
	user := &types.User{
		Username:            req.AdminUsername,
		Email:               req.AdminEmail,
		TenantID:            tenant.ID,
		IsActive:            true,
		CanAccessAllTenants: true, // Super admin
	}

	// We need to set the password. Assuming UserService has a way to create user with password
	// or we use a lower-level method.
	if err := s.userService.CreateUser(ctx, user, req.AdminPassword); err != nil {
		return fmt.Errorf("failed to create super admin user: %w", err)
	}

	return nil
}
