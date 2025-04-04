import { IUserAttributes } from "./admin/user";
import { Role } from "./auth";
import { ProjectStatus } from "./account";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

export interface User {
  id: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt: string;
  Name: string;
  Nickname: string;
  Password: string;
  role: Role;
  status: ProjectStatus;
  Space: string;
  ImageQuota: number;
  Attributes: IUserAttributes;
}

export const apiGetUser = (userName: string) =>
  instance.get<IResponse<User>>(`${VERSION}/users/${userName}`);
