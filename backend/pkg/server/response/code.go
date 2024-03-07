package response

type ErrorCode int

const (
	OK         ErrorCode = 0
	BadRequest ErrorCode = 400
	NotFound   ErrorCode = 404
)
