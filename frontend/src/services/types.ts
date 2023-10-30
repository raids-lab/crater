import { ErrorCode } from "./error_code";

export const ACCESS_TOKEN_KEY = "access_token";
export const REFRESH_TOKEN_KEY = "refresh_token";

// https://codevoweb.com/react-query-context-api-axios-interceptors-jwt-auth/
export interface IErrorResponse {
  error: string;
  error_code: ErrorCode;
}

export interface ITaskListResponse {
  data: {
    Tasks: ITask[];
  };
}

export interface ITask {
  taskName: string;
  UserName: string;
  slo: number;
  taskType: string;
  image: string;
  resourceRequest: {
    cpu: string;
    gpu: string;
    memory: string;
  };
  command: string;
  args: {
    [key: string]: string;
  };
  workingDir: string;
  ShareDirs: string[];
  id: number;
  Namespace: string;
  Status: string;
  createdAt: string;
  updatedAt: string;
}

export interface ICreateTask {
  taskName: string;
  slo: number;
  taskType: string;
  resourceRequest: {
    gpu: number;
    memory: string;
    cpu: number;
  };
  image: string;
  dir: string;
  share_dir: string[];
  command: string;
  args: {
    [key: string]: string;
  };
  priority: string;
}

export interface ICreateTaskResponse {
  data: string;
  error: string;
  status: boolean;
}

/**
 * Signup
 */
export interface ISignup {
  userName: string;
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
