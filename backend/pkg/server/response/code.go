package response

type ErrorCode int

const (
	OK ErrorCode = 0

	TokenExpired ErrorCode = 40101
	UserNotFound ErrorCode = 40102
	InvalidToken ErrorCode = 40103

	InvalidRole ErrorCode = 40301

	// Indicates laziness of the developer
	// Frontend will directly print the message without any translation
	NotSpecified ErrorCode = 99999
)
