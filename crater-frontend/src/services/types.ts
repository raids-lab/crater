import { ErrorCode } from "./error_code";

export const ACCESS_TOKEN_KEY = "access_token";
export const REFRESH_TOKEN_KEY = "refresh_token";

// https://codevoweb.com/react-query-context-api-axios-interceptors-jwt-auth/
export interface IErrorResponse {
  error: string;
  error_code: ErrorCode;
}

// {
//   "userName": "hjx",
//   "taskName": "xxx",
//   "slo":7,
//   "taskType":"ii",
//   "resourceRequest": {
// 	"gpu": 1,
// 	"memory": "10Gi",
//         "cpu": 10
//   },
//   "image": "xxx",
//   "dir": "/home/zhuangkr/DNN",
//   "share_dir": [
//     "/mnt/nfs/opt/xxx",
//     "/mnt/nfs/dataset/minst"
//   ],
//   "command": "python dnn.py",
//   "args": {
//       "--batch-size": "100",
//       "--epochs": "10",
//       "--workers": "5"
//    },
//   "priority": "high"
// }
export interface ITaskAttr {
  userName: string;
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

export interface ITaskAttrResponse {}

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
