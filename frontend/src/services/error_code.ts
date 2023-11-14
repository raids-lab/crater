// Error Code from server
// def type of error code as number
export type ErrorCode = number;

// 40105: Token is expired
export const ERROR_TOKEN_EXPIRED: ErrorCode = 40105;
export const ERROR_NOT_ADMIN: ErrorCode = 40107;

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
