package response

type ErrorCode int

const (
	OK ErrorCode = 0

	TokenExpired ErrorCode = 40105
	NotAdmin     ErrorCode = 40107
)
