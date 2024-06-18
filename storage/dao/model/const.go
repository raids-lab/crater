package model

// User role in platform and project
type Role uint8

const (
	_ Role = iota
	RoleGuest
	RoleUser
	RoleAdmin
)

// Project and user status
type Status uint8

const (
	_              Status = iota
	StatusPending         // Pending status, not yet activated
	StatusActive          // Active status
	StatusInactive        // Inactive status
)

// Space access mode (read-only, append-only, read-write)
type AccessMode uint8

const (
	_            AccessMode = iota
	AccessModeNA            // Not-allowed mode
	AccessModeRO            // Read-only mode
	AccessModeRW            // Read-write mode
	AccessModeAO            // Append-only mode
)

type FilePermission int

const (
	_ FilePermission = iota
	NotAllowed
	ReadOnly
	ReadWrite
)

type TokenResp struct {
	Code int `json:"code"`
	Data struct {
		UserID     uint           `json:"userId"`
		RootPath   string         `json:"rootPath"`
		Permission FilePermission `json:"permission"`
	} `json:"data"`
	Msg string `json:"msg"`
}
type FilePermissionType int

const (
	OtherMode FilePermissionType = iota
	PublicMode
	QueueMOde
	UserMode
)
