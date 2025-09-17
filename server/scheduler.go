package server

import "context"

type ModelScheduler interface {
	// Lock waits for the model to be ready.
	// Scheduler implementations should keep the model loaded until Unlock is called.
	// Lock does not imply access to the backend will be mutually exclusive.
	Lock(ctx context.Context, model string) (*backend, error)

	// Unlock must after a successful Lock call to signal that the backend is no longer in use.
	Unlock(*backend)
}
