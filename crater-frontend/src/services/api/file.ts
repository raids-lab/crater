import instance from "../axios";
import { IResponse } from "@/services/types";

export interface FileResp {
  name: string;
  size: number;
  isdir: boolean;
  modifytime: string;
  sys: string;
}

export const apiGetFiles = (path: string) =>
  instance.get<IResponse<FileResp[]>>(`ss/files${path}`);

export const apiMkdir = async (path: string) => {
  await instance.request({
    method: "MKCOL",
    url: `ss${path}`,
  });
};

export const apiUploadFile = (path: string, filedata: ArrayBuffer) =>
  instance.put(`ss${path}`, filedata);
