package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/aiplusall/aiplusall-kb/internal/config"
	"github.com/aiplusall/aiplusall-kb/internal/errors"
	"github.com/aiplusall/aiplusall-kb/internal/types"
	"github.com/aiplusall/aiplusall-kb/internal/types/interfaces"
)

// UserHandler implements HTTP request handlers for user management
type UserHandler struct {
	service         interfaces.UserService
	auditLogService interfaces.AuditLogService
	config          *config.Config
}

// NewUserHandler creates a new user handler instance
func NewUserHandler(service interfaces.UserService,
	auditLogService interfaces.AuditLogService,
	config *config.Config) *UserHandler {
	return &UserHandler{
		service:         service,
		auditLogService: auditLogService,
		config:          config,
	}
}

// ListUsers godoc
// @Summary      获取用户列表
// @Description  获取系统中的用户列表（仅限超级管理员）
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        tenant_id  query     int  false  "租户ID过滤"
// @Param        page       query     int  false  "页码"
// @Param        page_size  query     int  false  "每页数量"
// @Success      200        {object}  map[string]interface{}
// @Security     Bearer
// @Router       /users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	ctx := c.Request.Context()

	// Check if user has permission
	currentUser, err := h.service.GetCurrentUser(ctx)
	if err != nil {
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return
	}

	tenantIDStr := c.Query("tenant_id")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

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
		// Tenant Admin can only see their own tenant's users
		if tenantID != 0 && tenantID != currentUser.TenantID {
			c.Error(errors.NewForbiddenError("Cannot access other tenant's users"))
			return
		}
		tenantID = currentUser.TenantID
	}

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	users, total, err := h.service.ListUsers(ctx, tenantID, page, pageSize)
	if err != nil {
		c.Error(errors.NewInternalServerError("Failed to list users").WithDetails(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items": users,
			"total": total,
		},
	})
}

// UpdateUserStatus godoc
// @Summary      更新用户状态
// @Description  启用或停用用户（仅限超级管理员）
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        id      path      string  true  "用户ID"
// @Param        request body      map[string]bool true "状态"
// @Success      200     {object}  map[string]interface{}
// @Security     Bearer
// @Router       /users/{id}/status [put]
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Check if user has permission
	currentUser, err := h.service.GetCurrentUser(ctx)
	if err != nil {
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return
	}

	userID := c.Param("id")
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request parameters"))
		return
	}

	user, err := h.service.GetUserByID(ctx, userID)
	if err != nil {
		c.Error(errors.NewNotFoundError("User not found"))
		return
	}

	// Permission check: Super Admin can update anyone, Tenant Admin can only update their own tenant's users
	if !currentUser.CanAccessAllTenants {
		if currentUser.Role != types.RoleAdmin || user.TenantID != currentUser.TenantID {
			c.Error(errors.NewForbiddenError("Insufficient permissions"))
			return
		}
		// Tenant Admin cannot deactivate themselves or other admins?
		// For now, let's just allow it if it's the same tenant.
	}

	user.IsActive = req.IsActive
	if err := h.service.UpdateUser(ctx, user); err != nil {
		c.Error(errors.NewInternalServerError("Failed to update user status"))
		return
	}
	// Record audit log
	h.auditLogService.RecordLog(ctx, &types.AuditLog{
		UserID:     currentUser.ID,
		Username:   currentUser.Username,
		TenantID:   currentUser.TenantID,
		Action:     "update",
		Resource:   "user",
		ResourceID: user.ID,
		IP:         c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		Status:     "success",
		Details:    fmt.Sprintf("Updated user: %s", user.Username),
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User status updated successfully",
	})
}

// CreateUser godoc
// @Summary      创建用户
// @Description  创建新用户（仅限超级管理员）
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        request body      map[string]interface{} true "用户信息"
// @Success      200     {object}  map[string]interface{}
// @Security     Bearer
// @Router       /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	ctx := c.Request.Context()

	// Check if user has permission
	currentUser, err := h.service.GetCurrentUser(ctx)
	if err != nil {
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return
	}

	var req struct {
		Username            string               `json:"username" binding:"required"`
		Email               string               `json:"email" binding:"required,email"`
		Password            string               `json:"password" binding:"required,min=6"`
		TenantID            uint64               `json:"tenant_id"`
		Role                types.UserRole       `json:"role"`
		MenuConfig          types.UserMenuConfig `json:"menu_config"`
		IsActive            bool                 `json:"is_active"`
		CanAccessAllTenants bool                 `json:"can_access_all_tenants"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request parameters"))
		return
	}

	// Permission check
	if !currentUser.CanAccessAllTenants {
		if currentUser.Role != types.RoleAdmin {
			c.Error(errors.NewForbiddenError("Insufficient permissions"))
			return
		}
		// Tenant Admin can only create users for their own tenant
		req.TenantID = currentUser.TenantID
		// Tenant Admin cannot create Super Admins
		req.CanAccessAllTenants = false
	}

	if req.Role == "" {
		req.Role = types.RoleUser
	}

	user := &types.User{
		Username:            req.Username,
		Email:               req.Email,
		TenantID:            req.TenantID,
		Role:                req.Role,
		MenuConfig:          req.MenuConfig,
		IsActive:            req.IsActive,
		CanAccessAllTenants: req.CanAccessAllTenants,
	}

	if err := h.service.CreateUser(ctx, user, req.Password); err != nil {
		c.Error(errors.NewInternalServerError("Failed to create user").WithDetails(err.Error()))
		return
	}

	// Record audit log
	h.auditLogService.RecordLog(ctx, &types.AuditLog{
		UserID:     currentUser.ID,
		Username:   currentUser.Username,
		TenantID:   currentUser.TenantID,
		Action:     "create",
		Resource:   "user",
		ResourceID: user.ID,
		IP:         c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		Status:     "success",
		Details:    fmt.Sprintf("Created user: %s", user.Username),
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// UpdateUser godoc
// @Summary      更新用户信息
// @Description  更新用户信息（仅限超级管理员）
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        id      path      string  true  "用户ID"
// @Param        request body      map[string]interface{} true "用户信息"
// @Success      200     {object}  map[string]interface{}
// @Security     Bearer
// @Router       /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	ctx := c.Request.Context()

	// Check if user has permission
	currentUser, err := h.service.GetCurrentUser(ctx)
	if err != nil {
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return
	}

	userID := c.Param("id")
	var req struct {
		Username            string                `json:"username"`
		Email               string                `json:"email"`
		Password            string                `json:"password"`
		TenantID            uint64                `json:"tenant_id"`
		Role                types.UserRole        `json:"role"`
		MenuConfig          *types.UserMenuConfig `json:"menu_config"`
		IsActive            bool                  `json:"is_active"`
		CanAccessAllTenants bool                  `json:"can_access_all_tenants"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request parameters"))
		return
	}

	user, err := h.service.GetUserByID(ctx, userID)
	if err != nil {
		c.Error(errors.NewNotFoundError("User not found"))
		return
	}

	// Permission check
	if !currentUser.CanAccessAllTenants {
		if currentUser.Role != types.RoleAdmin || user.TenantID != currentUser.TenantID {
			c.Error(errors.NewForbiddenError("Insufficient permissions"))
			return
		}
		// Tenant Admin cannot change tenant_id or make someone Super Admin
		req.TenantID = user.TenantID
		req.CanAccessAllTenants = user.CanAccessAllTenants
	}

	// 构建更新字段的 map，只更新提供的字段
	updates := make(map[string]interface{})

	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.TenantID != 0 {
		updates["tenant_id"] = req.TenantID
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.MenuConfig != nil {
		updates["menu_config"] = *req.MenuConfig
	}

	// 对于布尔字段，我们需要特殊处理以支持 false 值
	// 检查请求中是否明确包含这些字段（通过检查是否为指针类型）
	updates["is_active"] = req.IsActive
	updates["can_access_all_tenants"] = req.CanAccessAllTenants

	// 只有在提供了新密码时才更新密码
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.Error(errors.NewInternalServerError("Failed to process password"))
			return
		}
		updates["password_hash"] = string(hashedPassword)
	}

	// 使用 updates map 而不是直接保存 user 对象
	if err := h.service.UpdateUserFields(ctx, userID, updates); err != nil {
		c.Error(errors.NewInternalServerError("Failed to update user"))
		return
	}

	// 重新获取更新后的用户信息返回给前端
	updatedUser, err := h.service.GetUserByID(ctx, userID)
	if err != nil {
		c.Error(errors.NewNotFoundError("Failed to retrieve updated user"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    updatedUser,
	})
}

// DeleteUser godoc
// @Summary      删除用户
// @Description  删除指定用户（仅限超级管理员）
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "用户ID"
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	ctx := c.Request.Context()

	// Check if user has permission
	currentUser, err := h.service.GetCurrentUser(ctx)
	if err != nil {
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return
	}

	userID := c.Param("id")
	if userID == currentUser.ID {
		c.Error(errors.NewBadRequestError("Cannot delete yourself"))
		return
	}

	user, err := h.service.GetUserByID(ctx, userID)
	if err != nil {
		c.Error(errors.NewNotFoundError("User not found"))
		return
	}

	// Permission check
	if !currentUser.CanAccessAllTenants {
		if currentUser.Role != types.RoleAdmin || user.TenantID != currentUser.TenantID {
			c.Error(errors.NewForbiddenError("Insufficient permissions"))
			return
		}
	}

	if err := h.service.DeleteUser(ctx, userID); err != nil {
		c.Error(errors.NewInternalServerError("Failed to delete user"))
		return
	}
	// Record audit log
	h.auditLogService.RecordLog(ctx, &types.AuditLog{
		UserID:     currentUser.ID,
		Username:   currentUser.Username,
		TenantID:   currentUser.TenantID,
		Action:     "delete",
		Resource:   "user",
		ResourceID: userID,
		IP:         c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		Status:     "success",
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deleted successfully",
	})
}
