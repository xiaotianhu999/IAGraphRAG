package interfaces

import (
	"context"

	"github.com/aiplusall/aiplusall-kb/internal/types"
)

// SystemInitializationService defines the interface for system-wide initialization
type SystemInitializationService interface {
	// IsInitialized checks if the system has been initialized
	IsInitialized(ctx context.Context) (bool, error)

	// Initialize performs the initial system setup
	Initialize(ctx context.Context, req types.SystemInitRequest) error
}
