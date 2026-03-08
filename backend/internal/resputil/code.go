package resputil

type ErrorCode int

const (
	OK ErrorCode = 0

	// General
	InvalidRequest     ErrorCode = 40001
	BusinessLogicError ErrorCode = 40002

	// Token (4010x)
	TokenExpired ErrorCode = 40101
	TokenInvalid ErrorCode = 40102

	// Auth/Login (4011x)
	InvalidCredentials      ErrorCode = 40111
	LdapError               ErrorCode = 40112
	LdapUserNotFound        ErrorCode = 40113
	LegacyTokenNotSupported ErrorCode = 40114

	// Registration/Uid (4012x)
	MustRegister    ErrorCode = 40121
	UidServiceError ErrorCode = 40122
	UidNotFound     ErrorCode = 40123

	// User is not allowed to access the resource
	UserNotAllowed ErrorCode = 40301

	// User's email is not verified
	UserEmailNotVerified ErrorCode = 40302

	// Container related
	ServiceSshdNotFound ErrorCode = 40401

	ServiceError ErrorCode = 50001

	// Indicates laziness of the developer
	// Frontend will directly print the message without any translation
	NotSpecified ErrorCode = 99999
)
