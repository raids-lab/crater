import instance from "../axios";
import { ACCESS_TOKEN_KEY, REFRESH_TOKEN_KEY } from "@/utils/store";

/**
 * Signup
 */
export interface ISignup {
  userName: string;
  role: string;
  password: string;
}

/**
 * Login
 */
export interface ILogin {
  username: string;
  password: string;
  auth: string;
}

export interface IAuthResponse {
  accessToken: string;
  refreshToken: string;
  role: string;
}

export const apiUserSignup = async (user: ISignup) => {
  const response = await instance.post<IAuthResponse>("signup", user);
  const { accessToken, refreshToken } = response.data;
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  return response.data;
};

export const apiUserLogin = async (user: ILogin) => {
  const response = await instance.post<IAuthResponse>("login", user);
  const { accessToken, refreshToken } = response.data;
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  return response.data;
};
