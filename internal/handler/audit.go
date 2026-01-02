package handler

import (
	"net/http"
	"strconv"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type AuditLogHandler struct {
	service     interfaces.AuditLogService
	userService interfaces.UserService
}

func NewAuditLogHandler(service interfaces.AuditLogService, userService interfaces.UserService) *AuditLogHandler {
	return &AuditLogHandler{
		service:     service,
		userService: userService,
	}
}

func (h *AuditLogHandler) GetLogs(c *gin.Context) {
	ctx := c.Request.Context()

	// Check if user has permission
	currentUser, err := h.userService.GetCurrentUser(ctx)
	if err != nil {
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	tenantIDStr := c.Query("tenant_id")

	var tenantID uint64
	if tenantIDStr != "" {
		tenantID, _ = strconv.ParseUint(tenantIDStr, 10, 64)
	}

	// Permission check: Super Admin can see all, Tenant Admin can only see their own tenant
	if !currentUser.CanAccessAllTenants {
		if currentUser.Role != types.RoleAdmin {
			c.Error(errors.NewForbiddenError("Insufficient permissions"))
			return
		}
		// Tenant Admin can only see their own tenant's logs
		if tenantID != 0 && tenantID != currentUser.TenantID {
			c.Error(errors.NewForbiddenError("Cannot access other tenant's audit logs"))
			return
		}
		tenantID = currentUser.TenantID
	}

	logs, total, err := h.service.ListAuditLogs(ctx, interfaces.AuditLogFilter{
		TenantID: tenantID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.Error(errors.NewInternalServerError("Failed to get audit logs").WithDetails(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items": logs,
			"total": total,
		},
	})
}
