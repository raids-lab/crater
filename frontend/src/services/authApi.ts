import axios, { AxiosRequestConfig, isAxiosError } from "axios";
import {
  IErrorResponse,
  ILogin,
  ILoginResponse,
  ISignup,
  ISignupResponse,
} from "./types";
const BASE_URL = "http://localhost:8000/";

export const authApi = axios.create({
  baseURL: BASE_URL,
  withCredentials: true,
});

authApi.defaults.headers.common["Content-Type"] = "application/json";

export const refreshAccessTokenFn = async () => {
  const response = await authApi.get<ILoginResponse>("refresh");
  return response.data;
};

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
      const errMessage = error.response?.data.message as string;
      if (
        errMessage.includes("Token is expired") &&
        originalRequest &&
        !originalRequest._retry
      ) {
        originalRequest._retry = true;
        await refreshAccessTokenFn();
        return authApi(originalRequest);
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
