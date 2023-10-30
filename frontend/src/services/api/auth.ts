import {
  ACCESS_TOKEN_KEY,
  ILogin,
  ILoginResponse,
  ISignup,
  ISignupResponse,
  REFRESH_TOKEN_KEY,
} from "../types";
import instance from "../axios";

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
