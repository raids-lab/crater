import { ErrorCode } from "./error_code";

// https://codevoweb.com/react-query-context-api-axios-interceptors-jwt-auth/
export interface IErrorResponse {
  error: string;
  error_code: ErrorCode;
}

export interface ITaskDelete {
  taskID: number;
}

export interface ITaskDeleteResponse {
  data: string;
  error: string;
  status: boolean;
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
    [key: string]: string;
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

export interface ITaskCreate {
  taskName: string;
  slo: number;
  taskType: string;
  resourceRequest: {
    [key: string]: string;
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

export interface ITaskCreateResponse {
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
