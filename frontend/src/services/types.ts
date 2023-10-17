export const ACCESS_TOKEN_KEY = "access_token";
export const REFRESH_TOKEN_KEY = "refresh_token";

// https://codevoweb.com/react-query-context-api-axios-interceptors-jwt-auth/
export interface IErrorResponse {
  message: string;
}

/**
 * Signup
 */
export interface ISignup {
  name: string;
  role: string;
  password: string;
}

export interface ISignupResponse {
  accessToken: string;
  refreshToken: string;
}

/**
 * Login
 */
export interface ILogin {
  username: string;
  password: string;
}

export interface ILoginResponse {
  accessToken: string;
  refreshToken: string;
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
