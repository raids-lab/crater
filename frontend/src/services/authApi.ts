import axios, { AxiosRequestConfig, isAxiosError } from "axios";
import {
  IErrorResponse,
  ILogin,
  ILoginResponse,
  IRefresh,
  IRefreshResponse,
  ISignup,
  ISignupResponse,
} from "./types";

const BASE_URL = "http://localhost:8000/";
const ACCESS_TOKEN_KEY = "access_token";
const REFRESH_TOKEN_KEY = "refresh_token";

export const authApi = axios.create({
  baseURL: BASE_URL,
  withCredentials: true,
});

authApi.defaults.headers.common["Content-Type"] = "application/json";

// Function to refresh the JWT token
const refreshTokenFn = async (): Promise<string> => {
  // Make a request to the server to refresh the token
  try {
    const data: IRefresh = {
      refreshToken: localStorage.getItem("REFRESH_TOKEN_KEY") || "",
    };
    const response = await authApi.post<IRefreshResponse>("/refresh", data);
    const { accessToken, refreshToken } = response.data;
    localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
    return accessToken;
  } catch (error) {
    throw error;
  }
};

// Add a request interceptor
authApi.interceptors.request.use(
  async (config) => {
    // Get the JWT token from local storage
    const token = localStorage.getItem("jwtToken");

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

interface AxiosRetryRequestConfig extends AxiosRequestConfig {
  _retry: boolean;
}

authApi.interceptors.response.use(
  (response) => {
    return response;
  },
  async (error) => {
    if (isAxiosError<IErrorResponse>(error)) {
      const originalRequest = error.config as AxiosRetryRequestConfig;
      const errMessage = (error.response?.data.message || "") as string;
      if (
        errMessage.includes("Token is expired") &&
        originalRequest &&
        !originalRequest._retry
      ) {
        originalRequest._retry = true;
        try {
          const token = await refreshTokenFn();
          authApi.defaults.headers.common["Authorization"] = `Bearer ${token}`;
          return authApi(originalRequest);
        } catch (refreshError) {
          return Promise.reject({ message: "" });
        }
      }
    }
    return Promise.reject(error);
  },
);

export const signUpUserFn = async (user: ISignup) => {
  const response = await authApi.post<ISignupResponse>("signup", user);
  return response.data;
};

export const loginUserFn = async (user: ILogin) => {
  const response = await authApi.post<ILoginResponse>("login", user);
  return response.data;
};

export const testFn = async () => {
  const response = await authApi.get<IErrorResponse>("huojian");
  return response.data;
};
