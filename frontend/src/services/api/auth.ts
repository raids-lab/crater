import instance from "../axios";
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
  Guest,
  User,
  Admin,
}

// const (
// 	StatusPending  Status = iota // Pending status, not yet activated
// 	StatusActive                 // Active status
// 	StatusInactive               // Inactive status
// )
export enum ProjectStatus {
  Pending,
  Active,
  Inactive,
}

export interface ProjectBasic {
  id: number;
  name: string;
  role: Role;
  isPersonal: boolean;
  status: ProjectStatus;
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
  projects: ProjectBasic[];
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
