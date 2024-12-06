package resputil

type ErrorCode int

const (
	OK ErrorCode = 0

	InvalidRequest    ErrorCode = 40001
	TokenExpired      ErrorCode = 40101
	TokenInvalid      ErrorCode = 40102
	MustRegister      ErrorCode = 40103
	UIDServerTimeout  ErrorCode = 40104
	UIDServerNotFound ErrorCode = 40105

	// User is not allowed to access the resource
	UserNotAllowed ErrorCode = 40301

	// Indicates laziness of the developer
	// Frontend will directly print the message without any translation
	NotSpecified ErrorCode = 99999
)
