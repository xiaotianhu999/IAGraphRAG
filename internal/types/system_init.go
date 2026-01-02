package types

// SystemInitRequest represents the request to initialize the system
type SystemInitRequest struct {
	AdminUsername string `json:"admin_username" binding:"required"`
	AdminEmail    string `json:"admin_email" binding:"required,email"`
	AdminPassword string `json:"admin_password" binding:"required,min=8"`
	TenantName    string `json:"tenant_name" binding:"required"`
}

// SystemInitStatus represents the initialization status of the system
type SystemInitStatus struct {
	IsInitialized bool `json:"is_initialized"`
}
