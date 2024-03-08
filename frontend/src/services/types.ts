import { ErrorCode } from "./error_code";

// https://codevoweb.com/react-query-context-api-axios-interceptors-jwt-auth/

export interface IResponse<T> {
  data: T;
  code: ErrorCode;
  msg: string;
}

export type IErrorResponse = IResponse<never>;

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
