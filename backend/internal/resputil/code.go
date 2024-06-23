package resputil

type ErrorCode int

const (
	OK ErrorCode = 0

	InvalidRequest ErrorCode = 40001
	TokenExpired   ErrorCode = 40101
	UserNotFound   ErrorCode = 40102

	// When request do not contain required queue
	QueueNotFound ErrorCode = 40401

	// Indicates laziness of the developer
	// Frontend will directly print the message without any translation
	NotSpecified ErrorCode = 99999
)
