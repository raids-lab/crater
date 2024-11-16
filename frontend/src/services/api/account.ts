import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { IUserAttributes } from "./admin/user";
import { Role } from "./auth";
import { QuotaResp } from "./context";

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
  admins?: number[];
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
  instance.get<IResponse<IAccount[]>>(`${VERSION}/admin/accounts`);

export const apiAccountCreate = (account: ICreateOrUpdateAccount) =>
  instance.post<IResponse<ICreateProjectResponse>>(
    VERSION + "/admin/accounts",
    account,
  );

export const apiAccountUpdate = (id: number, account: ICreateOrUpdateAccount) =>
  instance.put<IResponse<ICreateProjectResponse>>(
    `${VERSION}/admin/accounts/${id}`,
    account,
  );

export interface IDeleteProjectResp {
  name: string;
}
export const apiProjectDelete = (id: number) =>
  instance.delete<IResponse<IDeleteProjectResp>>(
    `${VERSION}/admin/accounts/${id}`,
  );

export interface IUserInAccountCreate {
  id: number;
  name: string;
  role: string;
  accessmode: string;
}

export type IUserInAccount = IUserInAccountCreate & {
  userInfo: IUserAttributes;
  quota: IQuota;
};

export enum Access {
  NA = 1,
  RO,
  RW,
  AO,
}

export const apiAddUser = async (pid: number, user: IUserInAccountCreate) =>
  instance.post<IResponse<IUserInAccount>>(
    VERSION + "/admin/accounts/add/" + pid + "/" + user.id,
    { role: user.role, accessmode: user.accessmode },
  );

export const apiUpdateUser = async (pid: number, user: IUserInAccountCreate) =>
  instance.post<IResponse<IUserInAccount>>(
    VERSION + "/admin/accounts/update/" + pid + "/" + user.id,
    { role: user.role, accessmode: user.accessmode },
  );

export const apiRemoveUser = async (pid: number, user: IUserInAccountCreate) =>
  instance.delete<IResponse<IUserInAccount>>(
    VERSION + "/admin/accounts/" + pid + "/" + user.id,
  );

export const apiUserInProjectList = (pid: number) =>
  instance.get<IResponse<IUserInAccount[]>>(
    VERSION + "/admin/accounts/userIn/" + pid,
  );

export const apiUserOutOfProjectList = (pid: number) =>
  instance.get<IResponse<IUserInAccount[]>>(
    VERSION + "/admin/accounts/userOutOf/" + pid,
  );

export const apiAccountQuotaGet = (pid: number) => {
  return instance.get<IResponse<QuotaResp>>(
    `${VERSION}/admin/accounts/${pid}/quota`,
  );
};

export const apiAccountGet = (pid: number) => {
  return instance.get<IResponse<IAccount>>(`${VERSION}/admin/accounts/${pid}`);
};
