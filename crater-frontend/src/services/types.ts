import { ErrorCode } from "./error_code";

// https://codevoweb.com/react-query-context-api-axios-interceptors-jwt-auth/
export interface IErrorResponse {
  error: string;
  errorCode: ErrorCode;
  status: boolean;
}

export interface IResponse<T> {
  data: T;
  error: string;
  status: boolean;
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
