import { ErrorCode } from "./error_code";

// https://codevoweb.com/react-query-context-api-axios-interceptors-jwt-auth/
export interface IErrorResponse {
  error: string;
  error_code: ErrorCode;
}

/**
 * Refresh
 */
export interface IRefresh {
  refreshToken: string;
}

export interface IRefreshResponse {
  accessToken: string;
  refreshToken: string;
}
