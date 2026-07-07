package interfaces

import "context"

// Executable represents an executable task.
// Execute runs until the task completes or the context is cancelled.
type Executable interface {
	Execute(ctx context.Context) error
}
