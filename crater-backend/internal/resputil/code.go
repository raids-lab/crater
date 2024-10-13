package resputil

type ErrorCode int

const (
	OK ErrorCode = 0

	InvalidRequest ErrorCode = 40001
	TokenExpired   ErrorCode = 40101
	TokenInvalid   ErrorCode = 40102

	// User is not allowed to access the resource
	UserNotAllowed ErrorCode = 40301

	// Indicates laziness of the developer
	// Frontend will directly print the message without any translation
	NotSpecified ErrorCode = 99999
)
