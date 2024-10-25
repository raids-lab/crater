import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { Role } from "./auth";

// const (
// 	StatusPending  Status = iota // Pending status, not yet activated
// 	StatusActive                 // Active status
// 	StatusInactive               // Inactive status
// )
export enum ProjectStatus {
  Pending = 1,
  Active,
  Inactive,
}

export interface ProjectBasic {
  id: number;
  name: string;
  role: Role;
  accessmode: Access;
}

export interface ICreateOrUpdateAccount {
  name: string;
  quota: IQuota;
  expiredAt: Date | undefined;
  withoutVolcano: boolean;
}

export interface ICreateProjectResponse {
  id: number;
}

export interface IQuota {
  guaranteed?: Record<string, string>;
  deserved?: Record<string, string>;
  capability?: Record<string, string>;
}

export interface IAccount {
  id: number;
  name: string;
  nickname: string;
  space: string;
  quota: IQuota;
  expiredAt?: string;
}

export const apiAdminAccountList = () =>
  instance.get<IResponse<IAccount[]>>(`${VERSION}/admin/projects`);

export const apiAccountCreate = (account: ICreateOrUpdateAccount) =>
  instance.post<IResponse<ICreateProjectResponse>>(
    VERSION + "/admin/projects",
    account,
  );

export const apiAccountUpdate = (id: number, account: ICreateOrUpdateAccount) =>
  instance.put<IResponse<ICreateProjectResponse>>(
    `${VERSION}/admin/projects/${id}`,
    account,
  );

export interface IDeleteProjectResp {
  name: string;
}
export const apiProjectDelete = (id: number) =>
  instance.delete<IResponse<IDeleteProjectResp>>(
    `${VERSION}/admin/projects/${id}`,
  );

export interface User {
  id: number;
  name: string;
  role: string;
  accessmode: string;
}

export enum Access {
  NA = 1,
  RO,
  RW,
  AO,
}

export const apiAddUser = async (pid: number, user: User) =>
  instance.post<IResponse<User>>(
    VERSION + "/admin/projects/add/" + pid + "/" + user.id,
    { role: user.role, accessmode: user.accessmode },
  );

export const apiUpdateUser = async (pid: number, user: User) =>
  instance.post<IResponse<User>>(
    VERSION + "/admin/projects/update/" + pid + "/" + user.id,
    { role: user.role, accessmode: user.accessmode },
  );

export const apiRemoveUser = async (pid: number, user: User) =>
  instance.delete<IResponse<User>>(
    VERSION + "/admin/projects/" + pid + "/" + user.id,
  );

export const apiUserInProjectList = (pid: number) =>
  instance.get<IResponse<User[]>>(VERSION + "/admin/projects/userIn/" + pid);

export const apiUserOutOfProjectList = (pid: number) =>
  instance.get<IResponse<User[]>>(VERSION + "/admin/projects/userOutOf/" + pid);
