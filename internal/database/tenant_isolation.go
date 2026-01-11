package database

import (
	"context"
	"reflect"

	"github.com/aiplusall/aiplusall-kb/internal/types"
	"gorm.io/gorm"
)

// WithTenantID returns a new context with the tenant ID
func WithTenantID(ctx context.Context, tenantID uint64) context.Context {
	return context.WithValue(ctx, types.TenantIDContextKey, tenantID)
}

// GetTenantID returns the tenant ID from the context
func GetTenantID(ctx context.Context) (uint64, bool) {
	tenantID, ok := ctx.Value(types.TenantIDContextKey).(uint64)
	return tenantID, ok
}

// IsSuperAdmin checks if the current user is a super admin
func IsSuperAdmin(ctx context.Context) bool {
	if user, ok := ctx.Value("user").(*types.User); ok && user != nil {
		return user.CanAccessAllTenants
	}
	return false
}

// TenantIsolationMiddleware is a GORM middleware for tenant isolation
func TenantIsolationMiddleware(db *gorm.DB) {
	// Query
	db.Callback().Query().Before("gorm:query").Register("tenant_isolation:query", func(d *gorm.DB) {
		if skip, _ := d.InstanceGet("skip_tenant_isolation"); skip == true {
			return
		}
		// Super admins can bypass isolation unless they are explicitly acting on a tenant
		if IsSuperAdmin(d.Statement.Context) {
			// If a specific tenant ID is set in context (e.g. via X-Tenant-ID), we still apply it
			// to keep the UI consistent with the selected tenant.
			// But if it's the super admin's own default tenant, we might want to allow seeing everything?
			// Actually, the best way is to let the repository decide via InstanceSet.
			// For now, let's just respect the skip flag.
			return
		}
		if d.Statement.Schema != nil {
			// Check if the model has a tenant_id field
			if _, ok := d.Statement.Schema.FieldsByDBName["tenant_id"]; ok {
				if tenantID, ok := GetTenantID(d.Statement.Context); ok && tenantID > 0 {
					d.Statement.Where("tenant_id = ?", tenantID)
				}
			}
		}
	})

	// Create
	db.Callback().Create().Before("gorm:create").Register("tenant_isolation:create", func(d *gorm.DB) {
		if skip, _ := d.InstanceGet("skip_tenant_isolation"); skip == true {
			return
		}
		if d.Statement.Schema != nil {
			if field, ok := d.Statement.Schema.FieldsByDBName["tenant_id"]; ok {
				if tenantID, ok := GetTenantID(d.Statement.Context); ok && tenantID > 0 {
					// Automatically set tenant_id if not already set
					// Check if ReflectValue is valid before accessing it
					if d.Statement.ReflectValue.IsValid() && d.Statement.ReflectValue.Kind() != 0 {
						// For batch operations, we need to handle slice types
						reflectValue := d.Statement.ReflectValue
						if reflectValue.Kind() == reflect.Slice {
							// For slice (batch create), iterate and set tenant_id for each element
							for i := 0; i < reflectValue.Len(); i++ {
								elem := reflectValue.Index(i)
								if elem.Kind() == reflect.Ptr {
									elem = elem.Elem()
								}
								if elem.IsValid() && elem.Kind() == reflect.Struct {
									if _, isZero := field.ValueOf(d.Statement.Context, elem); isZero {
										// Set tenant_id via reflection
										if tenantField := elem.FieldByName("TenantID"); tenantField.IsValid() && tenantField.CanSet() {
											tenantField.SetUint(tenantID)
										}
									}
								}
							}
						} else {
							// For single create
							if _, isZero := field.ValueOf(d.Statement.Context, d.Statement.ReflectValue); isZero {
								d.Statement.SetColumn("tenant_id", tenantID)
							}
						}
					}
				}
			}
		}
	})

	// Update
	db.Callback().Update().Before("gorm:update").Register("tenant_isolation:update", func(d *gorm.DB) {
		if skip, _ := d.InstanceGet("skip_tenant_isolation"); skip == true {
			return
		}
		if IsSuperAdmin(d.Statement.Context) {
			return
		}
		if d.Statement.Schema != nil {
			if _, ok := d.Statement.Schema.FieldsByDBName["tenant_id"]; ok {
				if tenantID, ok := GetTenantID(d.Statement.Context); ok && tenantID > 0 {
					d.Statement.Where("tenant_id = ?", tenantID)
				}
			}
		}
	})

	// Delete
	db.Callback().Delete().Before("gorm:delete").Register("tenant_isolation:delete", func(d *gorm.DB) {
		if skip, _ := d.InstanceGet("skip_tenant_isolation"); skip == true {
			return
		}
		if IsSuperAdmin(d.Statement.Context) {
			return
		}
		if d.Statement.Schema != nil {
			if _, ok := d.Statement.Schema.FieldsByDBName["tenant_id"]; ok {
				if tenantID, ok := GetTenantID(d.Statement.Context); ok && tenantID > 0 {
					d.Statement.Where("tenant_id = ?", tenantID)
				}
			}
		}
	})
}
