import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import { K8sResources } from "@/utils/resource";
import { Role } from "../auth";
import { ProjectStatus } from "../account";

export interface IUserAttributes {
  id: number;
  name: string;
  nickname: string;
  email?: string;
  teacher?: string;
  group?: string;
  expiredAt?: string;
  phone?: string;
  avatar?: string;
}

export interface IUser {
  id: number;
  name: string;
  role: Role;
  status: ProjectStatus;
  attributes: IUserAttributes;
}

export const apiAdminUserList = () =>
  instance.get<IResponse<IUser[]>>(`${VERSION}/admin/users`);

export const apiAdminUserDelete = (userName: string) =>
  instance.delete<IResponse<string>>(`${VERSION}/admin/users/${userName}`);

export interface IAdminUserUpdate {
  userName: string;
  hardQuota: K8sResources;
}

export const apiAdminUserUpdateQuota = (data: IAdminUserUpdate) =>
  instance.put<IResponse<string>>(
    `${VERSION}/admin/quotas/${data.userName}`,
    data,
  );

export const apiAdminUserUpdateRole = (userName: string, role: Role) =>
  instance.put<IResponse<string>>(`${VERSION}/admin/users/${userName}/role`, {
    role,
  });
