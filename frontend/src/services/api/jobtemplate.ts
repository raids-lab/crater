import instance, { VERSION } from "../axios";
import { IResponse } from "@/services/types";
import { IUserAttributes } from "./admin/user";
export interface JobTemplate {
  id: number;
  name: string;
  describe: string;
  document: string;
  createdAt: string;
  template: string;
  IUserAttributes: IUserAttributes;
}
export interface JobTemplateReq {
  describe: string;
  name: string;
  document: string;
  template: string;
}
export const listJobTemplate = () => {
  return instance.get<IResponse<JobTemplate[]>>(VERSION + `/jobtemplate/list`);
};
export const createJobTemplate = (data: JobTemplateReq) => {
  return instance.post<IResponse<string>>(
    VERSION + `/jobtemplate/create`,
    data,
  );
};
