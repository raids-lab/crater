// Error Code from server
// 40105: Token is expired
// def type of error code as number
export type ErrorCode = number;

export const ERROR_TOKEN_EXPIRED: ErrorCode = 40105;

// error: "User not found with the given name"
// error_code: 40401
export const ERROR_USER_NOT_FOUND: ErrorCode = 40401;

/**
{
  "error": "User already exists with the given Name",
  "error_code": 40901
}
 */
export const ERROR_USER_ALREADY_EXISTS: ErrorCode = 40901;
