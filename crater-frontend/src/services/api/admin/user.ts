import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
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
  uid?: string;
  gid?: string;
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

export const apiAdminUpdateUserAttributes = (
  username: string,
  data: IUserAttributes,
) =>
  instance.put<IResponse<string>>(
    `${VERSION}/admin/users/${username}/attributes`,
    data,
  );

export const apiAdminUserUpdateRole = (userName: string, role: Role) =>
  instance.put<IResponse<string>>(`${VERSION}/admin/users/${userName}/role`, {
    role,
  });
