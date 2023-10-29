import axios, { AxiosRequestConfig, isAxiosError } from "axios";
import { useNavigate } from "react-router-dom";
import {
  ACCESS_TOKEN_KEY,
  IErrorResponse,
  IRefresh,
  IRefreshResponse,
  REFRESH_TOKEN_KEY,
} from "./types";
import { ERROR_TOKEN_EXPIRED } from "./error_code";

const BASE_URL = "http://192.168.5.60:8078/";
export const VERSION = "v1";

interface AxiosRetryRequestConfig extends AxiosRequestConfig {
  _retry: boolean;
}

const useAxios = () => {
  const navigate = useNavigate();

  const instance = axios.create({
    baseURL: BASE_URL,
    withCredentials: true,
  });

  instance.defaults.headers.common["Content-Type"] = "application/json";

  const refreshTokenFn = async (): Promise<string> => {
    // Make a request to the server to refresh the token
    const data: IRefresh = {
      refreshToken: localStorage.getItem(REFRESH_TOKEN_KEY) || "",
    };
    const response = await instance.post<IRefreshResponse>("/refresh", data);
    const { accessToken, refreshToken } = response.data;
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
          error.response?.data.error_code === ERROR_TOKEN_EXPIRED &&
          originalRequest &&
          !originalRequest._retry
        ) {
          originalRequest._retry = true;
          try {
            const token = await refreshTokenFn();
            instance.defaults.headers.common[
              "Authorization"
            ] = `Bearer ${token}`;
            return instance(originalRequest);
          } catch (refreshError) {
            navigate("/login");
          }
        }
      }
      return Promise.reject(error);
    },
  );
  return { instance };
};

export default useAxios;
