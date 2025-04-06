package resputil

type ErrorCode int

const (
	OK ErrorCode = 0

	// General
	InvalidRequest ErrorCode = 40001

	// Token
	TokenExpired ErrorCode = 40101
	TokenInvalid ErrorCode = 40102

	// Login
	MustRegister       ErrorCode = 40103
	RegisterTimeout    ErrorCode = 40104
	RegisterNotFound   ErrorCode = 40105
	InvalidCredentials ErrorCode = 40106

	// User is not allowed to access the resource
	UserNotAllowed ErrorCode = 40301

	// User's email is not verified
	UserEmailNotVerified ErrorCode = 40302

	// Indicates laziness of the developer
	// Frontend will directly print the message without any translation
	NotSpecified ErrorCode = 99999
)
