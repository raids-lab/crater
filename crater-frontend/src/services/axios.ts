import axios, { AxiosRequestConfig, isAxiosError } from "axios";
import { IErrorResponse, IRefresh, IRefreshResponse, IResponse } from "./types";
import {
  ERROR_INVALID_REQUEST,
  ERROR_INVALID_ROLE,
  ERROR_NOT_SPECIFIED,
  ERROR_QUEUE_NOT_FOUND,
  ERROR_TOKEN_EXPIRED,
} from "./error_code";
import { ACCESS_TOKEN_KEY, REFRESH_TOKEN_KEY } from "@/utils/store";
import { showErrorToast } from "@/utils/toast";

export const VERSION = import.meta.env.VITE_API_VERSION;

interface AxiosRetryRequestConfig extends AxiosRequestConfig {
  _retry: boolean;
}

const instance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  withCredentials: true,
});

instance.defaults.headers.common["Content-Type"] = "application/json";

const refreshTokenFn = async (): Promise<string> => {
  // Make a request to the server to refresh the token
  const data: IRefresh = {
    refreshToken: localStorage.getItem(REFRESH_TOKEN_KEY) || "",
  };
  const response = await instance.post<IResponse<IRefreshResponse>>(
    "/refresh",
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
  (response) => {
    return response;
  },
  async (error) => {
    if (isAxiosError<IErrorResponse>(error)) {
      const originalRequest = error.config as AxiosRetryRequestConfig;
      if (
        error.response?.data.code === ERROR_TOKEN_EXPIRED &&
        originalRequest &&
        !originalRequest._retry
      ) {
        originalRequest._retry = true;
        try {
          const token = await refreshTokenFn();
          instance.defaults.headers.common["Authorization"] = `Bearer ${token}`;
          return instance(originalRequest);
        } catch (refreshError) {
          window.location.href = "/login";
        }
      } else if (error.response?.data.code === ERROR_INVALID_ROLE) {
        showErrorToast("暂无权限访问，将跳转到默认页面");
        window.location.href = "/portal";
      } else if (error.response?.data.code === ERROR_NOT_SPECIFIED) {
        showErrorToast(error);
      } else if (error.response?.data.code === ERROR_INVALID_REQUEST) {
        showErrorToast(`请求参数有误, ${error.response?.data.msg}`);
      } else if (error.response?.data.code === ERROR_QUEUE_NOT_FOUND) {
        showErrorToast("用户激活成功，但无关联账户，请联系平台管理员");
      }
    } else {
      showErrorToast(error);
    }
    return Promise.reject(error);
  },
);

export default instance;
