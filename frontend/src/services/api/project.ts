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

export interface ICreateProject {
  name: string;
  quota: string;
}

export interface ICreateProjectResponse {
  id: number;
}

export const apiProjectCreate = (project: ICreateProject) =>
  instance.post<IResponse<ICreateProjectResponse>>(
    VERSION + "/projects",
    project,
  );

export const apiProjectUpdate = (project: ICreateProject) =>
  instance.put<IResponse<ICreateProjectResponse>>(
    VERSION + "/admin/projects/" + project.name + "/quotas",
    project,
  );

export const apiProjectList = () =>
  instance.get<IResponse<ProjectBasic[]>>(VERSION + "/projects");

export interface IDeleteProjectReq {
  id: string;
}

export interface IDeleteProjectResp {
  name: string;
}
export const apiProjectDelete = (project: IDeleteProjectReq) =>
  instance.delete<IResponse<IDeleteProjectResp>>(
    VERSION + "/admin/projects/" + project.id,
  );

export interface User {
  id: number;
  name: string;
  role: number;
  accessmode: number;
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
