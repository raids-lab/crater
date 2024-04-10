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
  isPersonal: boolean;
  status: ProjectStatus;
}

export interface ICreateProject {
  name: string;
  description: string;
  quota: {
    cpu: number;
    memory: number;
    gpu: number;
    storage: number;
  };
}

export interface ICreateProjectResponse {
  id: number;
}

export const apiProjectCreate = (project: ICreateProject) =>
  instance.post<IResponse<ICreateProjectResponse>>(
    VERSION + "/projects",
    project,
  );

export const apiProjectList = () =>
  instance.get<IResponse<ProjectBasic[]>>(VERSION + "/projects");
