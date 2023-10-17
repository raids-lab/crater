import axios, { AxiosRequestConfig, isAxiosError } from "axios";
import { useNavigate } from "react-router-dom";
import {
  ACCESS_TOKEN_KEY,
  IErrorResponse,
  IRefresh,
  IRefreshResponse,
  REFRESH_TOKEN_KEY,
} from "./types";

const BASE_URL = "http://192.168.103.128:8888/api/";

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
        const errMessage = error.response?.data.message || "";
        if (
          errMessage.includes("Token is expired") &&
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
        } else {
          navigate("/login");
        }
      }
      return Promise.reject(error);
    },
  );
  return { instance };
};

export default useAxios;
