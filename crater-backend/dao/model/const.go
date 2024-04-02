package model

// User role in platform and project
type Role uint8

const (
	RoleGuest Role = iota
	RoleUser
	RoleAdmin
)

// Project and user status
type Status uint8

const (
	StatusPending  Status = iota // Pending status, not yet activated
	StatusActive                 // Active status
	StatusInactive               // Inactive status
)

// Space access mode (read-write, read-only)
type AccessMode uint8

const (
	AccessModeRW AccessMode = iota // Read-write mode
	AccessModeRO                   // Read-only mode
	AccessModeAO                   // Append-only mode
)
