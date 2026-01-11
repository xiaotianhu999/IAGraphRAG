package handler

import (
	"net/http"

	"github.com/aiplusall/aiplusall-kb/internal/types"
	"github.com/aiplusall/aiplusall-kb/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// SystemInitializationHandler handles system-wide initialization requests
type SystemInitializationHandler struct {
	initService interfaces.SystemInitializationService
}

// NewSystemInitializationHandler creates a new system initialization handler
func NewSystemInitializationHandler(initService interfaces.SystemInitializationService) *SystemInitializationHandler {
	return &SystemInitializationHandler{
		initService: initService,
	}
}

// GetInitStatus returns the initialization status of the system
// @Summary Get system initialization status
// @Description Check if the system has been initialized (at least one user exists)
// @Tags System
// @Produce json
// @Success 200 {object} types.SystemInitStatus
// @Router /api/v1/system/init-status [get]
func (h *SystemInitializationHandler) GetInitStatus(c *gin.Context) {
	initialized, err := h.initService.IsInitialized(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, types.SystemInitStatus{
		IsInitialized: initialized,
	})
}

// Initialize performs the initial system setup
// @Summary Initialize system
// @Description Create the first tenant and super admin user
// @Tags System
// @Accept json
// @Produce json
// @Param request body types.SystemInitRequest true "Initialization request"
// @Success 200 {object} map[string]string
// @Router /api/v1/system/initialize [post]
func (h *SystemInitializationHandler) Initialize(c *gin.Context) {
	var req types.SystemInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.initService.Initialize(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "System initialized successfully"})
}
