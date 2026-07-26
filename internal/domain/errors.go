package domain

import "errors"

// Sentinel errors mirror the former exception hierarchy. Callers use errors.Is
// to map them to HTTP status codes at the API layer.
var (
	// ErrNotFound indicates a missing resource (→ 404).
	ErrNotFound = errors.New("resource not found")
	// ErrProvider indicates a provider fetch/parse failure.
	ErrProvider = errors.New("provider error")
	// ErrNetworkOperation indicates a privileged network operation failure.
	ErrNetworkOperation = errors.New("network operation failed")
	// ErrOperationConflict indicates a mutually exclusive operation is running (→ 409).
	ErrOperationConflict = errors.New("network operation already running")
	// ErrDisabled indicates proxy connections are disabled.
	ErrDisabled = errors.New("proxy connections are disabled")
	// ErrRoutingMismatch indicates a node does not match current routing settings.
	ErrRoutingMismatch = errors.New("node does not match routing settings")
)
