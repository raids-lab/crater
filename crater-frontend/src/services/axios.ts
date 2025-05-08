import axios, { AxiosRequestConfig, isAxiosError } from "axios";
import { IErrorResponse, IRefresh, IRefreshResponse, IResponse } from "./types";
import {
  ERROR_INVALID_REQUEST,
  ERROR_NOT_SPECIFIED,
  ERROR_TOKEN_EXPIRED,
  ERROR_TOKEN_INVALID,
  ERROR_USER_NOT_ALLOWED,
  ERROR_USER_EMAIL_NOT_VERIFIED,
  ErrorCode,
} from "./error_code";
import { ACCESS_TOKEN_KEY, REFRESH_TOKEN_KEY } from "@/utils/store";
import { showErrorToast } from "@/utils/toast";
import { getDefaultStore } from "jotai";
import { configUrlApiBaseAtom, configVersionAtom } from "@/utils/store/config";

const store = getDefaultStore();

interface AxiosRetryRequestConfig extends AxiosRequestConfig {
  _retry: boolean;
}

// 异步函数初始化 Axios 实例
export async function initializeApiClient() {
  // 从 Jotai store 中获取 apiBaseUrl
  const baseURL = await store.get(configUrlApiBaseAtom);

  const apiClient = axios.create({
    baseURL,
    withCredentials: true,
  });

  const version = await store.get(configVersionAtom);

  return [apiClient, version] as const;
}

export const [instance, VERSION] = await initializeApiClient();

instance.defaults.headers.common["Content-Type"] = "application/json";

const refreshTokenFn = async (): Promise<string> => {
  // Make a request to the server to refresh the token
  const data: IRefresh = {
    refreshToken: localStorage.getItem(REFRESH_TOKEN_KEY) || "",
  };
  const response = await instance.post<IResponse<IRefreshResponse>>(
    "auth/refresh",
    data,
  );
  const { accessToken, refreshToken } = response.data.data;
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  return accessToken;
};

instance.interceptors.request.use(
  (config) => {
    // Get the JWT token from local storage
    const token = localStorage.getItem(ACCESS_TOKEN_KEY);
    // If the token exists, add it to the request headers
    if (token) {
      config.headers["Authorization"] = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  },
);

instance.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (!isAxiosError<IErrorResponse>(error)) {
      showErrorToast(error);
      return Promise.reject(error);
    }

    const originalRequest = error.config as AxiosRetryRequestConfig;
    const errorCode = error.response?.data.code;

    if (
      errorCode === ERROR_TOKEN_EXPIRED &&
      originalRequest &&
      !originalRequest._retry
    ) {
      originalRequest._retry = true;
      try {
        const token = await refreshTokenFn();
        instance.defaults.headers.common["Authorization"] = `Bearer ${token}`;
        return instance(originalRequest);
      } catch {
        window.location.href = "/login";
      }
    } else if (errorCode === ERROR_TOKEN_INVALID) {
      alert("登录过期，请重新登录");
      window.location.href = "/login";
    } else if (errorCode === ERROR_INVALID_REQUEST) {
      showErrorToast(`请求参数有误, ${error.response?.data.msg}`);
    } else if (errorCode === ERROR_USER_NOT_ALLOWED) {
      showErrorToast("用户激活成功，但无关联账户，请联系平台管理员");
    } else if (errorCode === ERROR_USER_EMAIL_NOT_VERIFIED) {
      showErrorToast("接收通知需要验证邮箱，请前往个人主页验证");
    } else if (errorCode === ERROR_NOT_SPECIFIED) {
      showErrorToast(error);
    }

    return Promise.reject(error);
  },
);

export const getErrorCode = (error: unknown): [ErrorCode, string] => {
  if (isAxiosError<IErrorResponse>(error)) {
    return [
      error.response?.data.code ?? ERROR_NOT_SPECIFIED,
      error.response?.data.msg ?? error.message,
    ];
  }
  return [ERROR_NOT_SPECIFIED, (error as Error).message];
};

export default instance;
