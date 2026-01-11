package handler

import (
	"net/http"

	"github.com/aiplusall/aiplusall-kb/internal/errors"
	"github.com/aiplusall/aiplusall-kb/internal/types"
	"github.com/aiplusall/aiplusall-kb/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// DashboardHandler handles dashboard related requests
type DashboardHandler struct {
	dashboardService interfaces.DashboardService
	userService      interfaces.UserService
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(dashboardService interfaces.DashboardService, userService interfaces.UserService) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
		userService:      userService,
	}
}

// checkAdmin checks if the current user has admin or super admin role
func (h *DashboardHandler) checkAdmin(c *gin.Context) bool {
	ctx := c.Request.Context()
	user, err := h.userService.GetCurrentUser(ctx)
	if err != nil {
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return false
	}
	if !user.CanAccessAllTenants && user.Role != types.RoleAdmin {
		c.Error(errors.NewForbiddenError("Insufficient permissions"))
		return false
	}
	return true
}

// GetDashboardData returns dashboard data
// @Summary Get dashboard data
// @Description Get statistics and recent activities for the dashboard
// @Tags Dashboard
// @Produce json
// @Success 200 {object} types.DashboardData
// @Router /api/v1/dashboard [get]
func (h *DashboardHandler) GetDashboardData(c *gin.Context) {
	// Check if user has permission
	if !h.checkAdmin(c) {
		return
	}

	data, err := h.dashboardService.GetDashboardData(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}
