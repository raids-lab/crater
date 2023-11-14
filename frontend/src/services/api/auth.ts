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

export const apiUserSignup = async (user: ISignup) => {
  const response = await instance.post<ISignupResponse>("signup", user);
  const { accessToken, refreshToken } = response.data;
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  return response.data;
};

export const apiUserLogin = async (user: ILogin): Promise<ILoginResponse> => {
  const response = await instance.post<ILoginResponse>("login", user);
  const { accessToken, refreshToken } = response.data;
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  return response.data;
};
