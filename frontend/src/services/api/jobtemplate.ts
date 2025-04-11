import instance, { VERSION } from "../axios";
import { IResponse } from "@/services/types";
import { IUserInfo } from "./vcjob";
export interface JobTemplate {
  id: number;
  name: string;
  describe: string;
  document: string;
  createdAt: string;
  template: string;
  userInfo: IUserInfo;
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
export const getJobTemplate = (id: number) => {
  return instance.get<IResponse<JobTemplate>>(VERSION + `/jobtemplate/${id}`);
};
export const deleteJobTemplate = (id: number) => {
  return instance.delete<IResponse<string>>(
    VERSION + `/jobtemplate/delete/${id}`,
  );
};
export const updateJobTemplate = (data: JobTemplateReq & { id: number }) => {
  return instance.put<IResponse<string>>(
    VERSION + `/jobtemplate/update/${data.id}`,
    data,
  );
};
