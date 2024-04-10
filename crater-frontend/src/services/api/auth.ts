import instance, { VERSION } from "../axios";
import { ACCESS_TOKEN_KEY, REFRESH_TOKEN_KEY } from "@/utils/store";
import { IResponse } from "../types";

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

// const (
// 	RoleGuest Role = iota
// 	RoleUser
// 	RoleAdmin
// )
export enum Role {
  Guest = 1,
  User,
  Admin,
}

export interface UserContext {
  projectID: number;
  projectRole: Role;
  platformRole: Role;
}

export interface IAuthResponse {
  accessToken: string;
  refreshToken: string;
  context: UserContext;
}

export const apiUserSignup = async (user: ISignup) => {
  const response = await instance.post<IAuthResponse>("signup", user);
  const { accessToken, refreshToken } = response.data;
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  return response.data;
};

export const apiUserLogin = async (user: ILogin) => {
  const response = await instance.post<IResponse<IAuthResponse>>("login", user);
  const { accessToken, refreshToken } = response.data.data;
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  return response.data.data;
};

export const apiProjectSwitch = async (id: number) => {
  const response = await instance.post<IResponse<IAuthResponse>>(
    VERSION + "/switch",
    {
      id,
    },
  );
  const { accessToken, refreshToken } = response.data.data;
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  return response.data.data;
};
